package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ExtractText(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".txt", ".md", ".json", ".csv":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return string(data), nil

	case ".pdf":
		return extractPDF(filePath)

	case ".docx":
		return extractDocx(filePath)

	default:
		return "", fmt.Errorf("فرمت %s پشتیبانی نمی‌شود", ext)
	}
}
