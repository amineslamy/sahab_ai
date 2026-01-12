package graphmem

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/net/html"
)

// Modality represents a content modality type.
type Modality string

const (
	ModalityText     Modality = "text"
	ModalityMarkdown Modality = "markdown"
	ModalityPDF      Modality = "pdf"
	ModalityImage    Modality = "image"
	ModalityAudio    Modality = "audio"
	ModalityWebpage  Modality = "webpage"
	ModalityCode     Modality = "code"
	ModalityJSON     Modality = "json"
	ModalityCSV      Modality = "csv"
)

// MultiModalInput represents input for multi-modal processing.
type MultiModalInput struct {
	Content   []byte            // Raw content bytes
	Text      string            // Text content (if applicable)
	Modality  Modality          // Content modality type
	SourceURI string            // Source location (file path or URL)
	Metadata  map[string]any    // Additional metadata
}

// ProcessedDocument represents the result of multi-modal processing.
type ProcessedDocument struct {
	ID        string            // Unique document ID
	Chunks    []*DocumentChunk  // Processed chunks
	Modality  Modality          // Original modality
	SourceURI string            // Source location
	Metadata  map[string]any    // Metadata
	RawText   string            // Extracted raw text
	Images    []ImageData       // Extracted images (for PDF/image modality)
	Tables    []TableData       // Extracted tables (for PDF modality)
}

// ImageData represents extracted image data.
type ImageData struct {
	Page   int               // Page number (1-indexed)
	Index  int               // Image index on page
	Format string            // Image format (png, jpg, etc.)
	Data   string            // Base64 encoded image data
	Width  int               // Image width
	Height int               // Image height
}

// TableData represents extracted table data.
type TableData struct {
	Page int        // Page number
	Data [][]string // Table rows and columns
}

// VisionAnalyzer is an interface for LLMs that support image analysis.
type VisionAnalyzer interface {
	AnalyzeImage(ctx context.Context, imageBase64, prompt string) (string, error)
}

// MultiModalProcessor processes different data modalities for memory ingestion.
type MultiModalProcessor struct {
	llm             LLMProvider
	visionAnalyzer  VisionAnalyzer
	chunker         *DocumentChunker
	markdownChunker *MarkdownChunker
	codeChunker     *CodeChunker
	chunkSize       int
	chunkOverlap    int
}

// MultiModalConfig holds configuration for the processor.
type MultiModalConfig struct {
	ChunkSize    int
	ChunkOverlap int
}

// NewMultiModalProcessor creates a new multi-modal processor.
func NewMultiModalProcessor(llm LLMProvider, config *MultiModalConfig) *MultiModalProcessor {
	if config == nil {
		config = &MultiModalConfig{
			ChunkSize:    1000,
			ChunkOverlap: 200,
		}
	}

	var visionAnalyzer VisionAnalyzer
	if va, ok := llm.(VisionAnalyzer); ok {
		visionAnalyzer = va
	}

	opts := &ChunkerOptions{
		ChunkSize:    config.ChunkSize,
		ChunkOverlap: config.ChunkOverlap,
	}

	return &MultiModalProcessor{
		llm:             llm,
		visionAnalyzer:  visionAnalyzer,
		chunker:         NewDocumentChunker(opts),
		markdownChunker: NewMarkdownChunker(opts),
		codeChunker:     NewCodeChunker(opts),
		chunkSize:       config.ChunkSize,
		chunkOverlap:    config.ChunkOverlap,
	}
}

// Process processes multi-modal input and returns a processed document.
func (p *MultiModalProcessor) Process(ctx context.Context, input *MultiModalInput) (*ProcessedDocument, error) {
	if input == nil {
		return nil, fmt.Errorf("input is required")
	}

	modality := input.Modality
	if modality == "" {
		modality = p.detectModality(input)
	}

	var doc *ProcessedDocument
	var err error

	switch modality {
	case ModalityText:
		doc, err = p.processText(input)
	case ModalityMarkdown:
		doc, err = p.processMarkdown(input)
	case ModalityPDF:
		doc, err = p.processPDF(ctx, input)
	case ModalityImage:
		doc, err = p.processImage(ctx, input)
	case ModalityAudio:
		doc, err = p.processAudio(ctx, input)
	case ModalityWebpage:
		doc, err = p.processWebpage(input)
	case ModalityCode:
		doc, err = p.processCode(input)
	case ModalityJSON:
		doc, err = p.processJSON(input)
	case ModalityCSV:
		doc, err = p.processCSV(input)
	default:
		doc, err = p.processText(input)
	}

	if err != nil {
		return nil, err
	}

	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}

	return doc, nil
}

// detectModality attempts to detect the modality from input.
func (p *MultiModalProcessor) detectModality(input *MultiModalInput) Modality {
	if input.SourceURI != "" {
		ext := strings.ToLower(filepath.Ext(input.SourceURI))
		switch ext {
		case ".md", ".markdown":
			return ModalityMarkdown
		case ".pdf":
			return ModalityPDF
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
			return ModalityImage
		case ".mp3", ".wav", ".m4a", ".flac", ".ogg":
			return ModalityAudio
		case ".py", ".go", ".js", ".ts", ".java", ".cpp", ".c", ".rs":
			return ModalityCode
		case ".json":
			return ModalityJSON
		case ".csv":
			return ModalityCSV
		case ".html", ".htm":
			return ModalityWebpage
		}

		if strings.HasPrefix(input.SourceURI, "http://") || strings.HasPrefix(input.SourceURI, "https://") {
			return ModalityWebpage
		}
	}

	// Check content type
	if len(input.Content) > 0 {
		// Check for PDF magic bytes
		if len(input.Content) >= 4 && string(input.Content[:4]) == "%PDF" {
			return ModalityPDF
		}
		// Check for PNG magic bytes
		if len(input.Content) >= 8 && string(input.Content[:8]) == "\x89PNG\r\n\x1a\n" {
			return ModalityImage
		}
		// Check for JPEG magic bytes
		if len(input.Content) >= 2 && input.Content[0] == 0xFF && input.Content[1] == 0xD8 {
			return ModalityImage
		}
	}

	return ModalityText
}

// getContent returns the text content from input.
func (p *MultiModalProcessor) getContent(input *MultiModalInput) string {
	if input.Text != "" {
		return input.Text
	}
	if len(input.Content) > 0 {
		return string(input.Content)
	}
	return ""
}

// processText processes plain text content.
func (p *MultiModalProcessor) processText(input *MultiModalInput) (*ProcessedDocument, error) {
	content := p.getContent(input)

	chunks := p.chunker.ChunkText(content, input.SourceURI, input.Metadata)

	return &ProcessedDocument{
		ID:        uuid.New().String(),
		Chunks:    chunks,
		Modality:  ModalityText,
		SourceURI: input.SourceURI,
		Metadata:  input.Metadata,
		RawText:   content,
	}, nil
}

// processMarkdown processes Markdown documents.
func (p *MultiModalProcessor) processMarkdown(input *MultiModalInput) (*ProcessedDocument, error) {
	content := p.getContent(input)

	chunks := p.markdownChunker.ChunkText(content, input.SourceURI, input.Metadata)

	return &ProcessedDocument{
		ID:        uuid.New().String(),
		Chunks:    chunks,
		Modality:  ModalityMarkdown,
		SourceURI: input.SourceURI,
		Metadata:  input.Metadata,
		RawText:   content,
	}, nil
}

// processPDF processes PDF documents.
func (p *MultiModalProcessor) processPDF(ctx context.Context, input *MultiModalInput) (*ProcessedDocument, error) {
	// Note: Full PDF processing requires external libraries like pdfcpu or unipdf
	// This is a simplified implementation that extracts basic text

	content := input.Content
	if len(content) == 0 && input.SourceURI != "" {
		// Try to read from file
		data, err := os.ReadFile(input.SourceURI)
		if err != nil {
			return nil, fmt.Errorf("failed to read PDF file: %w", err)
		}
		content = data
	}

	// Simple PDF text extraction (basic implementation)
	// For production use, integrate a proper PDF library
	text := p.extractPDFText(content)
	if text == "" {
		log.Printf("Warning: Could not extract text from PDF, using placeholder")
		text = "[PDF content - text extraction requires PDF library]"
	}

	chunks := p.chunker.ChunkText(text, input.SourceURI, input.Metadata)

	return &ProcessedDocument{
		ID:        uuid.New().String(),
		Chunks:    chunks,
		Modality:  ModalityPDF,
		SourceURI: input.SourceURI,
		Metadata:  input.Metadata,
		RawText:   text,
		Images:    []ImageData{},
		Tables:    []TableData{},
	}, nil
}

// extractPDFText extracts text from PDF bytes (basic implementation).
func (p *MultiModalProcessor) extractPDFText(content []byte) string {
	// This is a very basic text extractor that looks for text streams
	// For production, use a proper PDF library like pdfcpu or unipdf

	var textParts []string
	contentStr := string(content)

	// Look for text between BT and ET markers (basic PDF text objects)
	// This is a simplified approach and won't work for all PDFs
	inText := false
	var currentText strings.Builder

	for i := 0; i < len(contentStr)-1; i++ {
		if i < len(contentStr)-2 && contentStr[i:i+2] == "BT" {
			inText = true
			continue
		}
		if i < len(contentStr)-2 && contentStr[i:i+2] == "ET" {
			if currentText.Len() > 0 {
				textParts = append(textParts, currentText.String())
				currentText.Reset()
			}
			inText = false
			continue
		}
		if inText {
			// Look for text in parentheses
			if contentStr[i] == '(' {
				j := i + 1
				for j < len(contentStr) && contentStr[j] != ')' {
					if contentStr[j] == '\\' && j+1 < len(contentStr) {
						j += 2
						continue
					}
					currentText.WriteByte(contentStr[j])
					j++
				}
				i = j
			}
		}
	}

	return strings.Join(textParts, " ")
}

// processImage processes images using vision LLM or OCR.
func (p *MultiModalProcessor) processImage(ctx context.Context, input *MultiModalInput) (*ProcessedDocument, error) {
	content := input.Content
	if len(content) == 0 && input.SourceURI != "" {
		// Try to read from file
		data, err := os.ReadFile(input.SourceURI)
		if err != nil {
			return nil, fmt.Errorf("failed to read image file: %w", err)
		}
		content = data
	}

	imageBase64 := base64.StdEncoding.EncodeToString(content)

	var text string
	var err error

	// Try vision analysis first
	if p.visionAnalyzer != nil {
		text, err = p.visionAnalyzer.AnalyzeImage(ctx,
			imageBase64,
			"Describe this image in detail, including any text, diagrams, charts, or important visual elements. Extract all visible text.",
		)
		if err != nil {
			log.Printf("Vision analysis failed: %v, falling back to OCR", err)
		}
	}

	// Fallback: No vision available
	if text == "" {
		text = "[Image content - vision analysis not available]"
		if input.SourceURI != "" {
			text = fmt.Sprintf("[Image: %s - vision analysis not available]", filepath.Base(input.SourceURI))
		}
	}

	metadata := input.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["image_analysis"] = true

	chunks := p.chunker.ChunkText(text, input.SourceURI, metadata)

	return &ProcessedDocument{
		ID:        uuid.New().String(),
		Chunks:    chunks,
		Modality:  ModalityImage,
		SourceURI: input.SourceURI,
		Metadata:  metadata,
		RawText:   text,
		Images:    []ImageData{{Data: imageBase64}},
	}, nil
}

// processAudio processes audio files via transcription.
func (p *MultiModalProcessor) processAudio(ctx context.Context, input *MultiModalInput) (*ProcessedDocument, error) {
	// Audio transcription requires external service (Whisper API, etc.)
	// This is a placeholder implementation

	text := "[Audio content - transcription not available]"
	if input.SourceURI != "" {
		text = fmt.Sprintf("[Audio: %s - transcription not available]", filepath.Base(input.SourceURI))
	}

	metadata := input.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["transcribed"] = false

	chunks := p.chunker.ChunkText(text, input.SourceURI, metadata)

	return &ProcessedDocument{
		ID:        uuid.New().String(),
		Chunks:    chunks,
		Modality:  ModalityAudio,
		SourceURI: input.SourceURI,
		Metadata:  metadata,
		RawText:   text,
	}, nil
}

// processWebpage processes web pages.
func (p *MultiModalProcessor) processWebpage(input *MultiModalInput) (*ProcessedDocument, error) {
	var htmlContent string

	if input.Text != "" {
		htmlContent = input.Text
	} else if len(input.Content) > 0 {
		htmlContent = string(input.Content)
	} else if input.SourceURI != "" && (strings.HasPrefix(input.SourceURI, "http://") || strings.HasPrefix(input.SourceURI, "https://")) {
		// Fetch URL
		resp, err := http.Get(input.SourceURI)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch webpage: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read webpage: %w", err)
		}
		htmlContent = string(body)
	}

	// Parse HTML and extract text
	text, title, description := p.extractHTMLContent(htmlContent)

	metadata := input.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	if title != "" {
		metadata["title"] = title
	}
	if description != "" {
		metadata["description"] = description
	}

	chunks := p.markdownChunker.ChunkText(text, input.SourceURI, metadata)

	return &ProcessedDocument{
		ID:        uuid.New().String(),
		Chunks:    chunks,
		Modality:  ModalityWebpage,
		SourceURI: input.SourceURI,
		Metadata:  metadata,
		RawText:   text,
	}, nil
}

// extractHTMLContent extracts text content from HTML.
func (p *MultiModalProcessor) extractHTMLContent(htmlContent string) (text, title, description string) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent, "", ""
	}

	var textParts []string
	var extractText func(*html.Node)

	extractText = func(n *html.Node) {
		// Skip script, style, nav, footer, header tags
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "nav", "footer", "header", "aside":
				return
			case "title":
				if n.FirstChild != nil {
					title = n.FirstChild.Data
				}
				return
			case "meta":
				for _, attr := range n.Attr {
					if attr.Key == "name" && attr.Val == "description" {
						for _, a := range n.Attr {
							if a.Key == "content" {
								description = a.Val
							}
						}
					}
				}
				return
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				textParts = append(textParts, text)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}

	extractText(doc)
	text = strings.Join(textParts, "\n")
	return text, title, description
}

// processCode processes source code files.
func (p *MultiModalProcessor) processCode(input *MultiModalInput) (*ProcessedDocument, error) {
	content := p.getContent(input)

	// Detect language from file extension
	language := "text"
	if input.SourceURI != "" {
		ext := strings.ToLower(filepath.Ext(input.SourceURI))
		switch ext {
		case ".py":
			language = "python"
		case ".go":
			language = "go"
		case ".js":
			language = "javascript"
		case ".ts":
			language = "typescript"
		case ".java":
			language = "java"
		case ".cpp", ".cc", ".cxx":
			language = "cpp"
		case ".c":
			language = "c"
		case ".rs":
			language = "rust"
		case ".rb":
			language = "ruby"
		case ".php":
			language = "php"
		}
	}

	metadata := input.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["language"] = language

	chunks := p.codeChunker.ChunkCode(content, language, input.SourceURI, metadata)

	return &ProcessedDocument{
		ID:        uuid.New().String(),
		Chunks:    chunks,
		Modality:  ModalityCode,
		SourceURI: input.SourceURI,
		Metadata:  metadata,
		RawText:   content,
	}, nil
}

// processJSON processes JSON data.
func (p *MultiModalProcessor) processJSON(input *MultiModalInput) (*ProcessedDocument, error) {
	content := p.getContent(input)

	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		// Fall back to text processing
		return p.processText(input)
	}

	// Convert JSON to readable text
	text := p.jsonToText(data, 0)

	metadata := input.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["structured"] = true

	chunks := p.chunker.ChunkText(text, input.SourceURI, metadata)

	return &ProcessedDocument{
		ID:        uuid.New().String(),
		Chunks:    chunks,
		Modality:  ModalityJSON,
		SourceURI: input.SourceURI,
		Metadata:  metadata,
		RawText:   text,
	}, nil
}

// jsonToText converts JSON to readable text.
func (p *MultiModalProcessor) jsonToText(data any, indent int) string {
	indentStr := strings.Repeat("  ", indent)

	switch v := data.(type) {
	case map[string]any:
		var lines []string
		for k, val := range v {
			valText := p.jsonToText(val, indent+1)
			lines = append(lines, fmt.Sprintf("%s%s: %s", indentStr, k, valText))
		}
		return "\n" + strings.Join(lines, "\n")

	case []any:
		if len(v) == 0 {
			return "[]"
		}
		var lines []string
		for i, item := range v {
			if i >= 100 { // Limit items
				lines = append(lines, fmt.Sprintf("%s... and %d more items", indentStr, len(v)-i))
				break
			}
			lines = append(lines, fmt.Sprintf("%s[%d]: %s", indentStr, i, p.jsonToText(item, indent+1)))
		}
		return "\n" + strings.Join(lines, "\n")

	default:
		return fmt.Sprintf("%v", v)
	}
}

// processCSV processes CSV data.
func (p *MultiModalProcessor) processCSV(input *MultiModalInput) (*ProcessedDocument, error) {
	content := p.getContent(input)

	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		// Fall back to text processing
		return p.processText(input)
	}

	var textParts []string
	headers := []string{}

	for i, row := range records {
		if i >= 1000 { // Limit rows
			textParts = append(textParts, fmt.Sprintf("... and %d more rows", len(records)-i))
			break
		}

		if i == 0 {
			headers = row
			continue
		}

		var rowParts []string
		for j, cell := range row {
			if cell != "" {
				header := fmt.Sprintf("col%d", j)
				if j < len(headers) {
					header = headers[j]
				}
				rowParts = append(rowParts, fmt.Sprintf("%s: %s", header, cell))
			}
		}
		textParts = append(textParts, fmt.Sprintf("Row %d: %s", i, strings.Join(rowParts, ", ")))
	}

	text := strings.Join(textParts, "\n")

	metadata := input.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["structured"] = true
	metadata["row_count"] = len(records)

	chunks := p.chunker.ChunkText(text, input.SourceURI, metadata)

	return &ProcessedDocument{
		ID:        uuid.New().String(),
		Chunks:    chunks,
		Modality:  ModalityCSV,
		SourceURI: input.SourceURI,
		Metadata:  metadata,
		RawText:   text,
	}, nil
}

// ProcessFile is a convenience method to process a file by path.
func (p *MultiModalProcessor) ProcessFile(ctx context.Context, filePath string) (*ProcessedDocument, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	input := &MultiModalInput{
		Content:   content,
		SourceURI: filePath,
		Metadata:  map[string]any{"file_path": filePath},
	}

	return p.Process(ctx, input)
}

// ProcessURL is a convenience method to process a URL.
func (p *MultiModalProcessor) ProcessURL(ctx context.Context, url string) (*ProcessedDocument, error) {
	input := &MultiModalInput{
		SourceURI: url,
		Modality:  ModalityWebpage,
		Metadata:  map[string]any{"url": url},
	}

	return p.Process(ctx, input)
}

// ProcessBytes is a convenience method to process raw bytes.
func (p *MultiModalProcessor) ProcessBytes(ctx context.Context, content []byte, modality Modality, sourceURI string) (*ProcessedDocument, error) {
	input := &MultiModalInput{
		Content:   content,
		Modality:  modality,
		SourceURI: sourceURI,
	}

	return p.Process(ctx, input)
}

// IsImageBytes checks if bytes are a valid image (PNG, JPEG, GIF, or WebP).
func IsImageBytes(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	// PNG
	if bytes.Equal(data[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return true
	}
	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 {
		return true
	}
	// GIF
	if bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a")) {
		return true
	}
	// WebP
	if len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
		return true
	}
	return false
}

