package worker

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/flancast90/GraphMem-go/pkg/config"
	"github.com/flancast90/GraphMem-go/pkg/extractor"
	"github.com/flancast90/GraphMem-go/pkg/graph"
)

type DataWorker struct {
	cfg   *config.Config
	graph *graph.GraphService
}

func NewDataWorker(cfg *config.Config, g *graph.GraphService) *DataWorker {
	return &DataWorker{cfg: cfg, graph: g}
}

func (w *DataWorker) Start() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		w.processDataDir()
	}
}

func (w *DataWorker) processDataDir() {
	files, err := os.ReadDir(w.cfg.DataDir)
	if err != nil {
		log.Printf("خطا در خواندن پوشه data: %v", err)
		return
	}

	for _, file := range files {
		// نادیده گرفتن پوشه‌ها (از جمله processed و failed)
		if file.IsDir() {
			continue
		}

		srcPath := filepath.Join(w.cfg.DataDir, file.Name())
		log.Printf("📄 فایل جدید دریافت شد: %s", file.Name())

		content, err := extractor.ExtractText(srcPath)
		if err != nil {
			log.Printf("❌ خطا در استخراج متن %s: %v", file.Name(), err)
			w.moveFile(srcPath, filepath.Join(w.cfg.FailedDir, file.Name()))
			continue
		}

		log.Printf("⏳ ارسال به GraphMem جهت تحلیل و ذخیره‌سازی...")
		err = w.graph.IngestDocument(file.Name(), content)
		if err != nil {
			log.Printf("❌ خطا در ثبت گراف برای %s: %v", file.Name(), err)
			w.moveFile(srcPath, filepath.Join(w.cfg.FailedDir, file.Name()))
			continue
		}

		log.Printf("✅ فایل %s با موفقیت پردازش شد.", file.Name())
		w.moveFile(srcPath, filepath.Join(w.cfg.ProcessedDir, file.Name()))
	}
}

func (w *DataWorker) moveFile(src, dst string) {
	if err := os.Rename(src, dst); err != nil {
		_ = copyAndDelete(src, dst)
	}
}

func copyAndDelete(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	in.Close()
	return os.Remove(src)
}
