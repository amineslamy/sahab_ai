package graphmem

import (
	"regexp"
	"strings"
)

// DocumentChunk represents a chunk of a document.
type DocumentChunk struct {
	ID         string         `json:"id"`
	Content    string         `json:"content"`
	ChunkIndex int            `json:"chunk_index"`
	StartChar  int            `json:"start_char"`
	EndChar    int            `json:"end_char"`
	Metadata   map[string]any `json:"metadata"`
	SourceID   string         `json:"source_id,omitempty"`
	Modality   string         `json:"modality"`

	// Semantic metadata
	Summary    string   `json:"summary,omitempty"`
	Entities   []string `json:"entities,omitempty"`
	KeyPhrases []string `json:"key_phrases,omitempty"`
}

// DocumentChunker provides intelligent document chunking.
type DocumentChunker struct {
	ChunkSize         int
	ChunkOverlap      int
	MinChunkSize      int
	RespectSentences  bool
	RespectParagraphs bool

	// Patterns
	sentenceEndings  *regexp.Regexp
	paragraphPattern *regexp.Regexp
}

// ChunkerOptions contains options for DocumentChunker.
type ChunkerOptions struct {
	ChunkSize         int
	ChunkOverlap      int
	MinChunkSize      int
	RespectSentences  bool
	RespectParagraphs bool
}

// NewDocumentChunker creates a new DocumentChunker.
func NewDocumentChunker(opts *ChunkerOptions) *DocumentChunker {
	if opts == nil {
		opts = &ChunkerOptions{
			ChunkSize:         1000,
			ChunkOverlap:      200,
			MinChunkSize:      100,
			RespectSentences:  true,
			RespectParagraphs: true,
		}
	}

	return &DocumentChunker{
		ChunkSize:         opts.ChunkSize,
		ChunkOverlap:      opts.ChunkOverlap,
		MinChunkSize:      opts.MinChunkSize,
		RespectSentences:  opts.RespectSentences,
		RespectParagraphs: opts.RespectParagraphs,
		sentenceEndings:   regexp.MustCompile(`[.!?]\s+[A-Z]`), // Simplified pattern (Go doesn't support lookbehind)
		paragraphPattern:  regexp.MustCompile(`\n\s*\n`),
	}
}

// ChunkText chunks text into semantic units.
func (dc *DocumentChunker) ChunkText(text string, sourceID string, metadata map[string]any) []*DocumentChunk {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	if metadata == nil {
		metadata = make(map[string]any)
	}

	// First try paragraph-based chunking
	if dc.RespectParagraphs {
		paragraphs := dc.paragraphPattern.Split(text, -1)
		if len(paragraphs) > 1 {
			return dc.chunkByUnits(paragraphs, sourceID, metadata, text)
		}
	}

	// Fall back to sentence-based chunking
	if dc.RespectSentences {
		sentences := dc.sentenceEndings.Split(text, -1)
		if len(sentences) > 1 {
			return dc.chunkByUnits(sentences, sourceID, metadata, text)
		}
	}

	// Fall back to character-based chunking
	return dc.chunkByCharacters(text, sourceID, metadata)
}

// chunkByUnits chunks by semantic units (paragraphs or sentences).
func (dc *DocumentChunker) chunkByUnits(units []string, sourceID string, metadata map[string]any, originalText string) []*DocumentChunk {
	chunks := make([]*DocumentChunk, 0)
	currentChunk := ""
	currentStart := 0
	chunkIndex := 0

	for _, unit := range units {
		unit = strings.TrimSpace(unit)
		if unit == "" {
			continue
		}

		// Check if adding unit exceeds chunk size
		if len(currentChunk)+len(unit)+1 > dc.ChunkSize && currentChunk != "" {
			// Save current chunk
			chunks = append(chunks, &DocumentChunk{
				ID:         GenerateID(),
				Content:    strings.TrimSpace(currentChunk),
				ChunkIndex: chunkIndex,
				StartChar:  currentStart,
				EndChar:    currentStart + len(currentChunk),
				SourceID:   sourceID,
				Metadata:   copyMetadata(metadata),
				Modality:   "text",
			})
			chunkIndex++

			// Start new chunk with overlap
			overlapText := dc.getOverlap(currentChunk)
			currentStart = currentStart + len(currentChunk) - len(overlapText)
			if overlapText != "" {
				currentChunk = overlapText + " " + unit
			} else {
				currentChunk = unit
			}
		} else {
			if currentChunk != "" {
				currentChunk = currentChunk + " " + unit
			} else {
				currentChunk = unit
			}
		}
	}

	// Add final chunk
	if strings.TrimSpace(currentChunk) != "" {
		chunks = append(chunks, &DocumentChunk{
			ID:         GenerateID(),
			Content:    strings.TrimSpace(currentChunk),
			ChunkIndex: chunkIndex,
			StartChar:  currentStart,
			EndChar:    currentStart + len(currentChunk),
			SourceID:   sourceID,
			Metadata:   copyMetadata(metadata),
			Modality:   "text",
		})
	}

	return chunks
}

// chunkByCharacters performs simple character-based chunking.
func (dc *DocumentChunker) chunkByCharacters(text string, sourceID string, metadata map[string]any) []*DocumentChunk {
	chunks := make([]*DocumentChunk, 0)
	chunkIndex := 0
	start := 0

	for start < len(text) {
		end := start + dc.ChunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to find a good break point
		if end < len(text) {
			bestBreak := end
			minPos := start + dc.MinChunkSize
			if end-200 > minPos {
				minPos = end - 200
			}

			for i := end; i > minPos; i-- {
				if i-1 >= 0 && (text[i-1] == '.' || text[i-1] == '!' || text[i-1] == '?') {
					if i >= len(text) || (text[i] == ' ' || text[i] == '\n' || text[i] == '\t') {
						bestBreak = i
						break
					}
				}
			}
			end = bestBreak
		}

		chunkText := strings.TrimSpace(text[start:end])
		if chunkText != "" {
			chunks = append(chunks, &DocumentChunk{
				ID:         GenerateID(),
				Content:    chunkText,
				ChunkIndex: chunkIndex,
				StartChar:  start,
				EndChar:    end,
				SourceID:   sourceID,
				Metadata:   copyMetadata(metadata),
				Modality:   "text",
			})
			chunkIndex++
		}

		if end < len(text) {
			start = end - dc.ChunkOverlap
		} else {
			start = end
		}
	}

	return chunks
}

// getOverlap gets overlap text from end of chunk.
func (dc *DocumentChunker) getOverlap(text string) string {
	if len(text) <= dc.ChunkOverlap {
		return text
	}

	overlapText := text[len(text)-dc.ChunkOverlap:]

	// Try to start at a sentence boundary
	if dc.RespectSentences {
		re := regexp.MustCompile(`[.!?]\s+`)
		loc := re.FindStringIndex(overlapText)
		if loc != nil {
			return overlapText[loc[1]:]
		}
	}

	// Try to start at a word boundary
	spaceIdx := strings.Index(overlapText, " ")
	if spaceIdx > 0 {
		return overlapText[spaceIdx+1:]
	}

	return overlapText
}

func copyMetadata(m map[string]any) map[string]any {
	if m == nil {
		return make(map[string]any)
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// MarkdownChunker is specialized for Markdown documents.
type MarkdownChunker struct {
	*DocumentChunker
	headerPattern *regexp.Regexp
}

// NewMarkdownChunker creates a new MarkdownChunker.
func NewMarkdownChunker(opts *ChunkerOptions) *MarkdownChunker {
	return &MarkdownChunker{
		DocumentChunker: NewDocumentChunker(opts),
		headerPattern:   regexp.MustCompile(`(?m)^#{1,6}\s+`),
	}
}

// ChunkText chunks markdown by headers.
func (mc *MarkdownChunker) ChunkText(text string, sourceID string, metadata map[string]any) []*DocumentChunk {
	sections := mc.headerPattern.Split(text, -1)
	headers := mc.headerPattern.FindAllString(text, -1)

	if len(sections) <= 1 {
		return mc.DocumentChunker.ChunkText(text, sourceID, metadata)
	}

	chunks := make([]*DocumentChunk, 0)
	for i, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		sectionMetadata := copyMetadata(metadata)
		if i > 0 && i-1 < len(headers) {
			headerLevel := len(strings.TrimRight(strings.TrimSpace(headers[i-1]), " "))
			sectionMetadata["header_level"] = headerLevel
		}

		// Recursively chunk large sections
		if len(section) > mc.ChunkSize {
			subChunks := mc.DocumentChunker.ChunkText(section, sourceID, sectionMetadata)
			chunks = append(chunks, subChunks...)
		} else {
			chunks = append(chunks, &DocumentChunk{
				ID:         GenerateID(),
				Content:    section,
				ChunkIndex: len(chunks),
				SourceID:   sourceID,
				Metadata:   sectionMetadata,
				Modality:   "text",
			})
		}
	}

	return chunks
}

// CodeChunker is specialized for code files.
type CodeChunker struct {
	*DocumentChunker
	patterns map[string]*regexp.Regexp
}

// NewCodeChunker creates a new CodeChunker.
func NewCodeChunker(opts *ChunkerOptions) *CodeChunker {
	return &CodeChunker{
		DocumentChunker: NewDocumentChunker(opts),
		patterns: map[string]*regexp.Regexp{
			"python":     regexp.MustCompile(`(?m)^(?:def|class|async def)\s+\w+`),
			"javascript": regexp.MustCompile(`(?m)^(?:function|class|const|let|var)\s+\w+`),
			"typescript": regexp.MustCompile(`(?m)^(?:function|class|const|let|interface|type)\s+\w+`),
			"go":         regexp.MustCompile(`(?m)^(?:func|type)\s+\w+`),
		},
	}
}

// ChunkCode chunks code by function/class boundaries.
func (cc *CodeChunker) ChunkCode(code string, language string, sourceID string, metadata map[string]any) []*DocumentChunk {
	pattern := cc.patterns[strings.ToLower(language)]
	if pattern == nil {
		return cc.DocumentChunker.ChunkText(code, sourceID, metadata)
	}

	matches := pattern.FindAllStringIndex(code, -1)
	if len(matches) == 0 {
		return cc.DocumentChunker.ChunkText(code, sourceID, metadata)
	}

	chunks := make([]*DocumentChunk, 0)
	for i, match := range matches {
		start := match[0]
		var end int
		if i+1 < len(matches) {
			end = matches[i+1][0]
		} else {
			end = len(code)
		}

		section := strings.TrimSpace(code[start:end])
		if section == "" {
			continue
		}

		sectionMetadata := copyMetadata(metadata)
		sectionMetadata["language"] = language
		sectionMetadata["symbol"] = strings.TrimSpace(code[match[0]:match[1]])

		chunks = append(chunks, &DocumentChunk{
			ID:         GenerateID(),
			Content:    section,
			ChunkIndex: len(chunks),
			StartChar:  start,
			EndChar:    end,
			SourceID:   sourceID,
			Metadata:   sectionMetadata,
			Modality:   "code",
		})
	}

	return chunks
}
