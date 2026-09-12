package extractor

import (
	"github.com/nguyenthenguyen/docx"
)

func extractDocx(filePath string) (string, error) {
	r, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	docxData := r.Editable()
	return docxData.GetContent(), nil
}
