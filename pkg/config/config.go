package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DataDir      string
	ProcessedDir string
	FailedDir    string
	OllamaURL    string
	OllamaModel  string
}

func LoadConfig() *Config {
	baseDir := "."
	dataDir := filepath.Join(baseDir, "data")
	return &Config{
		DataDir:      dataDir,
		ProcessedDir: filepath.Join(dataDir, "processed"),
		FailedDir:    filepath.Join(dataDir, "failed"),
		OllamaURL:    "http://localhost:11434/v1",
		OllamaModel:  "qwen2.5:3b",
	}
}

func (c *Config) EnsureDirs() error {
	dirs := []string{c.DataDir, c.ProcessedDir, c.FailedDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
