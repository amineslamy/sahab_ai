package graphmem

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// PipelineConfig contains configuration for the ingestion pipeline.
type PipelineConfig struct {
	MaxExtractionWorkers int
	MaxEmbeddingWorkers  int
	EmbeddingBatchSize   int
	LLMRateLimit         int
	EmbeddingRateLimit   int
	MaxRetries           int
	RetryDelay           time.Duration
	ChunkSize            int
	ChunkOverlap         int
	Streaming            bool
}

// DefaultPipelineConfig returns default pipeline configuration.
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		MaxExtractionWorkers: 10,
		MaxEmbeddingWorkers:  8,
		EmbeddingBatchSize:   100,
		LLMRateLimit:         60,
		EmbeddingRateLimit:   3000,
		MaxRetries:           3,
		RetryDelay:           time.Second,
		ChunkSize:            2000,
		ChunkOverlap:         200,
		Streaming:            true,
	}
}

// PipelineIngestResult represents the result of document ingestion in the pipeline.
type PipelineIngestResult struct {
	DocID            string
	Success          bool
	Entities         int
	Relationships    int
	Clusters         int
	Embeddings       int
	Error            error
	ElapsedSeconds   float64
	MemoryID         string
	ProcessingTimeMS float64
}

// PipelineStats holds pipeline performance statistics.
type PipelineStats struct {
	DocumentsProcessed int64
	DocumentsFailed    int64
	TotalEntities      int64
	TotalRelationships int64
	TotalEmbeddings    int64
	TotalTimeSeconds   float64
	AvgDocTimeMS       float64
	DocsPerSecond      float64
}

// HighPerformancePipeline provides high-performance document ingestion.
type HighPerformancePipeline struct {
	config     *PipelineConfig
	llm        LLMProvider
	embeddings EmbeddingProvider
	kg         *KnowledgeGraph
	cache      Cache
	onProgress func(processed, total int)
	stats      PipelineStats
	startTime  time.Time
	mu         sync.RWMutex
}

// NewHighPerformancePipeline creates a new high-performance pipeline.
func NewHighPerformancePipeline(llm LLMProvider, embeddings EmbeddingProvider, config *PipelineConfig, cache Cache) *HighPerformancePipeline {
	if config == nil {
		config = DefaultPipelineConfig()
	}

	kgConfig := &KnowledgeGraphConfig{
		ChunkSize:    config.ChunkSize,
		ChunkOverlap: config.ChunkOverlap,
		MaxWorkers:   config.MaxExtractionWorkers,
	}

	return &HighPerformancePipeline{
		config:     config,
		llm:        llm,
		embeddings: embeddings,
		kg:         NewKnowledgeGraph(llm, embeddings, kgConfig),
		cache:      cache,
	}
}

// SetProgressCallback sets the progress callback function.
func (p *HighPerformancePipeline) SetProgressCallback(fn func(processed, total int)) {
	p.onProgress = fn
}

// IngestDocuments ingests multiple documents with high performance.
func (p *HighPerformancePipeline) IngestDocuments(ctx context.Context, documents []Document, memoryID, userID string) ([]*PipelineIngestResult, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	p.startTime = time.Now()

	log.Printf("=" + strings.Repeat("=", 59))
	log.Printf("🚀 HIGH-PERFORMANCE INGESTION PIPELINE")
	log.Printf("=" + strings.Repeat("=", 59))
	log.Printf("   Documents: %d", len(documents))
	log.Printf("   Extraction workers: %d", p.config.MaxExtractionWorkers)
	log.Printf("   Embedding workers: %d", p.config.MaxEmbeddingWorkers)
	log.Printf("   Embedding batch size: %d", p.config.EmbeddingBatchSize)
	log.Printf("=" + strings.Repeat("=", 59))

	// Phase 1: Chunk all documents
	log.Printf("\n📋 PHASE 1: Chunking documents")
	allChunks, docChunkMap := p.chunkDocuments(documents)
	log.Printf("   Created %d chunks from %d documents", len(allChunks), len(documents))

	// Phase 2: Extract entities/relationships (concurrent)
	log.Printf("\n🔍 PHASE 2: Extracting entities/relationships (concurrent)")
	extractionResults := p.extractConcurrent(ctx, allChunks, memoryID, userID)

	// Phase 3: Generate embeddings (batch parallel)
	log.Printf("\n🔢 PHASE 3: Generating embeddings (batch parallel)")
	embeddingResults := p.embedBatch(ctx, extractionResults)

	// Phase 4: Compile results
	log.Printf("\n📊 PHASE 4: Compiling results")
	results := p.compileResults(documents, extractionResults, embeddingResults, docChunkMap)

	// Update stats
	p.updateStats(results)

	elapsed := time.Since(p.startTime).Seconds()
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	log.Printf("\n" + strings.Repeat("=", 60))
	log.Printf("✅ INGESTION COMPLETE")
	log.Printf("=" + strings.Repeat("=", 59))
	log.Printf("   Documents: %d (%d successful)", len(results), successCount)
	log.Printf("   Entities: %d", p.stats.TotalEntities)
	log.Printf("   Relationships: %d", p.stats.TotalRelationships)
	log.Printf("   Embeddings: %d", p.stats.TotalEmbeddings)
	log.Printf("   Time: %.2fs", elapsed)
	log.Printf("   Throughput: %.2f docs/sec", float64(len(documents))/elapsed)
	log.Printf("=" + strings.Repeat("=", 59))

	return results, nil
}

// Document represents a document for ingestion.
type Document struct {
	ID       string
	Content  string
	Metadata map[string]any
}

type chunk struct {
	ID   string
	Text string
}

type extractionResult struct {
	ChunkID string
	Nodes   []*MemoryNode
	Edges   []*MemoryEdge
	Error   error
}

func (p *HighPerformancePipeline) chunkDocuments(documents []Document) ([]chunk, map[string]string) {
	allChunks := make([]chunk, 0)
	docChunkMap := make(map[string]string) // chunkID -> docID

	for _, doc := range documents {
		chunks := p.chunkText(doc.Content, doc.ID)
		for _, c := range chunks {
			allChunks = append(allChunks, c)
			docChunkMap[c.ID] = doc.ID
		}
	}

	return allChunks, docChunkMap
}

func (p *HighPerformancePipeline) chunkText(text, docID string) []chunk {
	chunks := make([]chunk, 0)
	chunkSize := p.config.ChunkSize
	overlap := p.config.ChunkOverlap

	if len(text) <= chunkSize {
		return []chunk{{ID: docID + "_0", Text: text}}
	}

	start := 0
	chunkIdx := 0

	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunkText := text[start:end]
		chunks = append(chunks, chunk{
			ID:   fmt.Sprintf("%s_%d", docID, chunkIdx),
			Text: chunkText,
		})

		start = end - overlap
		chunkIdx++
	}

	return chunks
}

func (p *HighPerformancePipeline) extractConcurrent(ctx context.Context, chunks []chunk, memoryID, userID string) []extractionResult {
	results := make([]extractionResult, len(chunks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.config.MaxExtractionWorkers)

	for i, c := range chunks {
		wg.Add(1)
		go func(idx int, ch chunk) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			nodes, edges, err := p.kg.Extract(ctx, ch.Text, memoryID, userID)
			results[idx] = extractionResult{
				ChunkID: ch.ID,
				Nodes:   nodes,
				Edges:   edges,
				Error:   err,
			}

			if err != nil {
				log.Printf("   ⚠️  Chunk %s extraction failed: %v", ch.ID, err)
			}
		}(i, c)
	}

	wg.Wait()

	successCount := 0
	totalNodes := 0
	totalEdges := 0
	for _, r := range results {
		if r.Error == nil {
			successCount++
			totalNodes += len(r.Nodes)
			totalEdges += len(r.Edges)
		}
	}

	log.Printf("   Extracted %d entities, %d relationships from %d/%d chunks",
		totalNodes, totalEdges, successCount, len(chunks))

	return results
}

func (p *HighPerformancePipeline) embedBatch(ctx context.Context, extractionResults []extractionResult) map[string][]float32 {
	// Collect all texts that need embeddings
	textsToEmbed := make([]string, 0)
	textIDs := make([]string, 0)

	for _, result := range extractionResults {
		if result.Error != nil {
			continue
		}

		// Embed entity names
		for _, node := range result.Nodes {
			text := node.Description
			if text == "" {
				text = node.Name
			}
			if len(text) > 2000 {
				text = text[:2000]
			}
			textsToEmbed = append(textsToEmbed, text)
			textIDs = append(textIDs, "entity:"+node.ID)
		}
	}

	if len(textsToEmbed) == 0 {
		return make(map[string][]float32)
	}

	// Batch embed
	embeddings := make(map[string][]float32)
	batchSize := p.config.EmbeddingBatchSize

	for i := 0; i < len(textsToEmbed); i += batchSize {
		end := i + batchSize
		if end > len(textsToEmbed) {
			end = len(textsToEmbed)
		}

		batch := textsToEmbed[i:end]
		batchIDs := textIDs[i:end]

		embResults, err := p.embeddings.EmbedBatch(ctx, batch)
		if err != nil {
			log.Printf("   ⚠️  Batch embedding failed: %v", err)
			continue
		}

		for j, emb := range embResults {
			if j < len(batchIDs) {
				embeddings[batchIDs[j]] = emb
			}
		}
	}

	log.Printf("   Generated %d embeddings", len(embeddings))
	return embeddings
}

func (p *HighPerformancePipeline) compileResults(documents []Document, extractionResults []extractionResult, embeddings map[string][]float32, docChunkMap map[string]string) []*PipelineIngestResult {
	// Group results by document
	docResults := make(map[string]*PipelineIngestResult)
	for _, doc := range documents {
		docResults[doc.ID] = &PipelineIngestResult{
			DocID:   doc.ID,
			Success: true,
		}
	}

	// Aggregate extraction results
	for _, result := range extractionResults {
		docID := docChunkMap[result.ChunkID]
		if docID == "" {
			continue
		}

		dr, ok := docResults[docID]
		if !ok {
			continue
		}

		if result.Error != nil {
			dr.Success = false
			dr.Error = result.Error
		}

		dr.Entities += len(result.Nodes)
		dr.Relationships += len(result.Edges)
	}

	// Count embeddings per document
	for range embeddings {
		// textID format: "entity:nodeID"
		// We'd need to map nodeID back to docID
		// For now, just count total
	}

	// Convert to slice
	results := make([]*PipelineIngestResult, 0, len(documents))
	for _, doc := range documents {
		if dr, ok := docResults[doc.ID]; ok {
			if !dr.Success && dr.Entities == 0 && dr.Relationships == 0 {
				dr.Success = false
			} else if dr.Entities > 0 || dr.Relationships > 0 {
				dr.Success = true
				dr.Error = nil
			}
			results = append(results, dr)
		}
	}

	return results
}

func (p *HighPerformancePipeline) updateStats(results []*PipelineIngestResult) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, result := range results {
		if result.Success {
			p.stats.DocumentsProcessed++
		} else {
			p.stats.DocumentsFailed++
		}
		p.stats.TotalEntities += int64(result.Entities)
		p.stats.TotalRelationships += int64(result.Relationships)
		p.stats.TotalEmbeddings += int64(result.Embeddings)
	}

	p.stats.TotalTimeSeconds = time.Since(p.startTime).Seconds()
	totalDocs := p.stats.DocumentsProcessed + p.stats.DocumentsFailed
	if totalDocs > 0 {
		p.stats.AvgDocTimeMS = (p.stats.TotalTimeSeconds * 1000) / float64(totalDocs)
		p.stats.DocsPerSecond = float64(totalDocs) / p.stats.TotalTimeSeconds
	}
}

// Stats returns the pipeline statistics.
func (p *HighPerformancePipeline) Stats() PipelineStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// IngestStream streams ingestion results as they complete.
func (p *HighPerformancePipeline) IngestStream(ctx context.Context, documents []Document, memoryID, userID string) <-chan *PipelineIngestResult {
	resultChan := make(chan *PipelineIngestResult)

	go func() {
		defer close(resultChan)

		// Process in smaller batches for streaming
		batchSize := p.config.MaxExtractionWorkers
		if batchSize < 1 {
			batchSize = 1
		}

		for i := 0; i < len(documents); i += batchSize {
			end := i + batchSize
			if end > len(documents) {
				end = len(documents)
			}
			batch := documents[i:end]

			results, err := p.IngestDocuments(ctx, batch, memoryID, userID)
			if err != nil {
				log.Printf("Batch ingestion failed: %v", err)
				continue
			}

			for _, result := range results {
				select {
				case resultChan <- result:
				case <-ctx.Done():
					return
				}

				if p.onProgress != nil {
					p.onProgress(i+1, len(documents))
				}
			}
		}
	}()

	return resultChan
}

// BatchEmbedder provides efficient batch embedding generation.
type BatchEmbedder struct {
	provider        EmbeddingProvider
	batchSize       int
	maxWorkers      int
	maxRetries      int
	retryDelay      time.Duration
	rateLimitPerMin int
	cache           Cache
}

// NewBatchEmbedder creates a new batch embedder.
func NewBatchEmbedder(provider EmbeddingProvider, batchSize, maxWorkers, maxRetries int, retryDelay time.Duration, rateLimitPerMin int, cache Cache) *BatchEmbedder {
	if batchSize <= 0 {
		batchSize = 100
	}
	if maxWorkers <= 0 {
		maxWorkers = 8
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if retryDelay <= 0 {
		retryDelay = time.Second
	}
	if rateLimitPerMin <= 0 {
		rateLimitPerMin = 3000
	}

	return &BatchEmbedder{
		provider:        provider,
		batchSize:       batchSize,
		maxWorkers:      maxWorkers,
		maxRetries:      maxRetries,
		retryDelay:      retryDelay,
		rateLimitPerMin: rateLimitPerMin,
		cache:           cache,
	}
}

// EmbedBatch embeds multiple texts in batches.
func (be *BatchEmbedder) EmbedBatch(ctx context.Context, texts []string, textIDs []string) (map[string][]float32, error) {
	results := make(map[string][]float32)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, be.maxWorkers)

	// Process in batches
	for i := 0; i < len(texts); i += be.batchSize {
		end := i + be.batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batchTexts := texts[i:end]
		batchIDs := textIDs[i:end]

		wg.Add(1)
		go func(bt []string, bi []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			embeddings, err := be.provider.EmbedBatch(ctx, bt)
			if err != nil {
				log.Printf("Batch embedding failed: %v", err)
				return
			}

			mu.Lock()
			for j, emb := range embeddings {
				if j < len(bi) {
					results[bi[j]] = emb
				}
			}
			mu.Unlock()
		}(batchTexts, batchIDs)
	}

	wg.Wait()
	return results, nil
}
