package graph

import (
	"errors"
	"fmt"

	"github.com/flancast90/GraphMem-go/pkg/config"
	"github.com/flancast90/GraphMem-go/pkg/graphmem"
)

type GraphService struct {
	Engine *graphmem.GraphMem
}

func NewGraphService(cfg *config.Config) (*GraphService, error) {
	gmConfig := graphmem.NewConfig()

	// مپ کردن فیلدها از config.Config برنامه شما
	gmConfig.LLMModel = cfg.OllamaModel
	gmConfig.LLMAPIBase = cfg.OllamaURL

	engine, err := graphmem.New(gmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GraphMem engine: %w", err)
	}

	return &GraphService{
		Engine: engine,
	}, nil
}

func (s *GraphService) IngestDocument(filename string, content string) error {
	if s.Engine == nil {
		return errors.New("graphmem engine is not initialized")
	}

	result, err := s.Engine.Ingest(content)
	if err != nil {
		return fmt.Errorf("ingestion failed for %s: %w", filename, err)
	}

	// اگر هیچ موجودیتی استخراج نشد، عملیات را ناموفق فرض کن
	if result.Entities == 0 && result.Relationships == 0 {
		return fmt.Errorf("no entities or relationships extracted from %s (LLM or connection issue)", filename)
	}

	return nil
}

func (s *GraphService) Close() error {
	if s.Engine != nil {
		return s.Engine.Close()
	}
	return nil
}
