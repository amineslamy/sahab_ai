package graphmem

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/sashabaranov/go-openai"
)

// EmbeddingProvider defines the interface for embedding generation.
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// EmbeddingOptions holds configuration for embedding providers.
type EmbeddingOptions struct {
	Provider   string
	APIKey     string
	APIBase    string
	APIVersion string
	Model      string
	Deployment string
	Dimensions int
	Timeout    time.Duration
}

// EmbeddingCache defines the interface for embedding caching.
type EmbeddingCache interface {
	Get(key string) ([]float32, bool)
	Set(key string, embedding []float32)
}

// openAIEmbedding implements EmbeddingProvider for OpenAI and compatible APIs.
type openAIEmbedding struct {
	client     *openai.Client
	model      string
	deployment string
	cache      EmbeddingCache
}

// NewEmbeddingProvider creates a new embedding provider.
func NewEmbeddingProvider(opts *EmbeddingOptions) (EmbeddingProvider, error) {
	if opts == nil {
		return nil, fmt.Errorf("options required")
	}

	switch opts.Provider {
	case "openai", "openai_compatible", "":
		return newOpenAIEmbedding(opts)
	case "azure_openai":
		return newAzureOpenAIEmbedding(opts)
	case "openrouter":
		return newOpenRouterEmbedding(opts)
	case "together", "together_ai":
		return newTogetherEmbedding(opts)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", opts.Provider)
	}
}

func newOpenAIEmbedding(opts *EmbeddingOptions) (*openAIEmbedding, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key required")
	}

	config := openai.DefaultConfig(opts.APIKey)
	if opts.APIBase != "" {
		config.BaseURL = opts.APIBase
	}

	client := openai.NewClientWithConfig(config)

	model := opts.Model
	if model == "" {
		model = "text-embedding-3-small"
	}

	return &openAIEmbedding{
		client: client,
		model:  model,
	}, nil
}

func newAzureOpenAIEmbedding(opts *EmbeddingOptions) (*openAIEmbedding, error) {
	if opts.APIKey == "" || opts.APIBase == "" || opts.Deployment == "" {
		return nil, fmt.Errorf("API key, base URL, and deployment required for Azure OpenAI")
	}

	config := openai.DefaultAzureConfig(opts.APIKey, opts.APIBase)
	config.APIVersion = opts.APIVersion
	if config.APIVersion == "" {
		config.APIVersion = "2024-12-01-preview"
	}

	client := openai.NewClientWithConfig(config)

	return &openAIEmbedding{
		client:     client,
		model:      opts.Deployment,
		deployment: opts.Deployment,
	}, nil
}

// Embed generates embedding for a single text.
func (e *openAIEmbedding) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, nil
	}

	// Check cache
	if e.cache != nil {
		key := hashText(text)
		if cached, found := e.cache.Get(key); found {
			return cached, nil
		}
	}

	resp, err := e.client.CreateEmbeddings(
		ctx,
		openai.EmbeddingRequest{
			Model: openai.EmbeddingModel(e.model),
			Input: []string{text},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("embedding generation failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, nil
	}

	embedding := resp.Data[0].Embedding

	// Save to cache
	if e.cache != nil {
		key := hashText(text)
		e.cache.Set(key, embedding)
	}

	return embedding, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *openAIEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Filter empty texts
	validTexts := make([]string, 0, len(texts))
	validIndices := make([]int, 0, len(texts))
	for i, text := range texts {
		if text != "" {
			validTexts = append(validTexts, text)
			validIndices = append(validIndices, i)
		}
	}

	if len(validTexts) == 0 {
		return make([][]float32, len(texts)), nil
	}

	// Process in batches
	const batchSize = 100
	results := make([][]float32, len(texts))

	for i := 0; i < len(validTexts); i += batchSize {
		end := i + batchSize
		if end > len(validTexts) {
			end = len(validTexts)
		}
		batch := validTexts[i:end]

		resp, err := e.client.CreateEmbeddings(
			ctx,
			openai.EmbeddingRequest{
				Model: openai.EmbeddingModel(e.model),
				Input: batch,
			},
		)
		if err != nil {
			// Continue with empty embeddings for failed batch
			continue
		}

		for j, item := range resp.Data {
			idx := validIndices[i+j]
			results[idx] = item.Embedding
		}
	}

	return results, nil
}

// CosineSimilarity calculates cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func hashText(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// =============================================================================
// OpenRouter Embedding Provider
// =============================================================================

func newOpenRouterEmbedding(opts *EmbeddingOptions) (*openAIEmbedding, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key required for OpenRouter")
	}

	config := openai.DefaultConfig(opts.APIKey)
	config.BaseURL = "https://openrouter.ai/api/v1"

	client := openai.NewClientWithConfig(config)

	model := opts.Model
	if model == "" {
		model = "openai/text-embedding-3-small"
	}

	return &openAIEmbedding{
		client: client,
		model:  model,
	}, nil
}

// =============================================================================
// Together AI Embedding Provider
// =============================================================================

func newTogetherEmbedding(opts *EmbeddingOptions) (*openAIEmbedding, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key required for Together AI")
	}

	config := openai.DefaultConfig(opts.APIKey)
	config.BaseURL = "https://api.together.xyz/v1"

	client := openai.NewClientWithConfig(config)

	model := opts.Model
	if model == "" {
		model = "togethercomputer/m2-bert-80M-8k-retrieval"
	}

	return &openAIEmbedding{
		client: client,
		model:  model,
	}, nil
}
