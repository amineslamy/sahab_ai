package graphmem

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ExtractWebpage extracts text content from a webpage.
func ExtractWebpage(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty URL")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("invalid URL scheme")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with user agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Fetch webpage
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch webpage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body: %w", err)
	}

	// Extract text from HTML (simple implementation)
	text := extractTextFromHTML(string(body))

	return text, nil
}

// extractTextFromHTML extracts text from HTML content.
// This is a simple implementation - for production, use a proper HTML parser.
func extractTextFromHTML(html string) string {
	// Remove script and style elements
	scriptPattern := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptPattern.ReplaceAllString(html, "")

	stylePattern := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = stylePattern.ReplaceAllString(html, "")

	// Remove navigation, header, footer, aside elements
	navPattern := regexp.MustCompile(`(?is)<(?:nav|header|footer|aside)[^>]*>.*?</(?:nav|header|footer|aside)>`)
	html = navPattern.ReplaceAllString(html, "")

	// Try to find main content
	mainPattern := regexp.MustCompile(`(?is)<(?:main|article)[^>]*>(.*?)</(?:main|article)>`)
	if match := mainPattern.FindStringSubmatch(html); len(match) > 1 {
		html = match[1]
	}

	// Remove HTML comments
	commentPattern := regexp.MustCompile(`<!--.*?-->`)
	html = commentPattern.ReplaceAllString(html, "")

	// Remove HTML tags
	tagPattern := regexp.MustCompile(`<[^>]+>`)
	text := tagPattern.ReplaceAllString(html, " ")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Clean up whitespace
	multiSpace := regexp.MustCompile(`\s+`)
	text = multiSpace.ReplaceAllString(text, " ")

	multiNewline := regexp.MustCompile(`\n{3,}`)
	text = multiNewline.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// CheckWebpageURL checks and extracts webpage if valid URL.
func CheckWebpageURL(url string) string {
	if url == "" {
		return ""
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return ""
	}

	text, err := ExtractWebpage(url)
	if err != nil {
		return ""
	}

	return text
}

// TextExtractor provides simple text extraction utilities.
type TextExtractor struct{}

// NewTextExtractor creates a new TextExtractor.
func NewTextExtractor() *TextExtractor {
	return &TextExtractor{}
}

// ExtractFromURL extracts text from a URL.
func (te *TextExtractor) ExtractFromURL(url string) (string, error) {
	return ExtractWebpage(url)
}

// ExtractFromText returns the text as-is (for interface compatibility).
func (te *TextExtractor) ExtractFromText(text string) (string, error) {
	return text, nil
}

// CleanText cleans and normalizes text content.
func (te *TextExtractor) CleanText(text string) string {
	// Remove control characters
	controlPattern := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	text = controlPattern.ReplaceAllString(text, "")

	// Normalize whitespace
	multiSpace := regexp.MustCompile(`[ \t]+`)
	text = multiSpace.ReplaceAllString(text, " ")

	// Normalize newlines
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove excessive newlines
	multiNewline := regexp.MustCompile(`\n{3,}`)
	text = multiNewline.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

