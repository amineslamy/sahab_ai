package graphmem

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GraphMem is the main interface for the GraphMem memory system.
type GraphMem struct {
	config     *Config
	memoryID   string
	userID     string
	autoEvolve bool

	memory     *Memory
	store      Store
	cache      Cache
	llm        LLMProvider
	embeddings EmbeddingProvider
	kg         *KnowledgeGraph
	retriever  *Retriever
	evolution  *Evolution

	initialized bool
	initMu      sync.Mutex
	memoryMu    sync.RWMutex

	metrics *Metrics
}

// Metrics holds runtime metrics for GraphMem.
type Metrics struct {
	Ingestions    int64
	Queries       int64
	Evolutions    int64
	TotalNodes    int64
	TotalEdges    int64
	TotalClusters int64
	mu            sync.RWMutex
}

// Option is a functional option for GraphMem.
type Option func(*GraphMem)

// WithMemoryID sets the memory ID.
func WithMemoryID(id string) Option {
	return func(g *GraphMem) {
		g.memoryID = id
	}
}

// WithUserID sets the user ID for multi-tenant isolation.
func WithUserID(id string) Option {
	return func(g *GraphMem) {
		g.userID = id
	}
}

// WithAutoEvolve enables automatic memory evolution.
func WithAutoEvolve(enabled bool) Option {
	return func(g *GraphMem) {
		g.autoEvolve = enabled
	}
}

// New creates a new GraphMem instance.
func New(config *Config, opts ...Option) (*GraphMem, error) {
	if config == nil {
		config = NewConfig()
	}

	gm := &GraphMem{
		config:  config,
		userID:  config.UserID,
		metrics: &Metrics{},
	}

	// Apply options
	for _, opt := range opts {
		opt(gm)
	}

	// Set defaults
	if gm.userID == "" {
		gm.userID = "default"
	}

	return gm, nil
}

// ensureInitialized lazily initializes all components.
func (gm *GraphMem) ensureInitialized() error {
	if gm.initialized {
		return nil
	}

	gm.initMu.Lock()
	defer gm.initMu.Unlock()

	if gm.initialized {
		return nil
	}

	if err := gm.initializeComponents(); err != nil {
		return fmt.Errorf("failed to initialize GraphMem: %w", err)
	}

	gm.initialized = true
	return nil
}

// initializeComponents initializes all internal components.
func (gm *GraphMem) initializeComponents() error {
	// Initialize LLM provider
	llmOpts := &LLMOptions{
		Provider:    gm.config.LLMProvider,
		APIKey:      gm.config.LLMAPIKey,
		APIBase:     gm.config.LLMAPIBase,
		APIVersion:  gm.config.AzureAPIVersion,
		Model:       gm.config.LLMModel,
		Deployment:  gm.config.AzureDeployment,
		Temperature: gm.config.LLMTemperature,
		MaxTokens:   gm.config.LLMMaxTokens,
		Timeout:     gm.config.QueryTimeout,
	}
	llmProvider, err := NewLLMProvider(llmOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}
	gm.llm = llmProvider

	// Initialize embedding provider
	embOpts := &EmbeddingOptions{
		Provider:   gm.config.EmbeddingProvider,
		APIKey:     gm.config.EmbeddingAPIKey,
		APIBase:    gm.config.EmbeddingAPIBase,
		APIVersion: gm.config.AzureAPIVersion,
		Model:      gm.config.EmbeddingModel,
		Deployment: gm.config.AzureEmbeddingDeployment,
		Dimensions: gm.config.EmbeddingDimensions,
		Timeout:    gm.config.QueryTimeout,
	}
	embProvider, err := NewEmbeddingProvider(embOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize embeddings: %w", err)
	}
	gm.embeddings = embProvider

	// Initialize store
	gm.store = NewInMemoryStore()

	// Initialize cache
	gm.cache = NewInMemoryCache(int(gm.config.QueryTimeout.Seconds()))

	// Initialize knowledge graph
	kgConfig := &KnowledgeGraphConfig{
		ChunkSize:           gm.config.ChunkSize,
		ChunkOverlap:        gm.config.ChunkOverlap,
		MaxTripletsPerChunk: gm.config.MaxTripletsPerChunk,
		MaxWorkers:          gm.config.MaxWorkers,
	}
	gm.kg = NewKnowledgeGraph(gm.llm, gm.embeddings, kgConfig)

	// Initialize retriever
	gm.retriever = NewRetriever(
		gm.embeddings,
		gm.config.SimilarityTopK,
		gm.config.MinSimilarityThreshold,
	)

	// Initialize evolution
	if gm.config.EvolutionEnabled {
		evolConfig := &EvolutionConfig{
			ConsolidationThreshold: gm.config.ConsolidationThreshold,
			DecayEnabled:           gm.config.DecayEnabled,
			DecayHalfLifeDays:      gm.config.DecayHalfLifeDays,
			MinEvolutionInterval:   time.Hour,
			MaxWorkers:             gm.config.MaxWorkers,
		}
		gm.evolution = NewEvolution(gm.llm, gm.embeddings, evolConfig)
	}

	// Initialize or load memory
	if gm.memoryID != "" {
		if err := gm.loadMemory(); err != nil {
			gm.createNewMemory()
		}
	} else {
		gm.createNewMemory()
	}

	return nil
}

// createNewMemory creates a new memory instance.
func (gm *GraphMem) createNewMemory() {
	memoryID := gm.memoryID
	if memoryID == "" {
		memoryID = uuid.New().String()
	}
	gm.memory = NewMemory(memoryID)
	gm.memoryID = gm.memory.ID
}

// loadMemory loads an existing memory from storage.
func (gm *GraphMem) loadMemory() error {
	loaded, err := gm.store.LoadMemory(gm.memoryID, gm.userID)
	if err != nil {
		return err
	}
	if loaded == nil {
		return fmt.Errorf("memory not found: %s", gm.memoryID)
	}
	gm.memory = loaded
	return nil
}

// Ingest ingests text content into memory.
func (gm *GraphMem) Ingest(content string) (*IngestResult, error) {
	return gm.IngestWithContext(context.Background(), content, nil)
}

// IngestWithContext ingests text content into memory with context and metadata.
func (gm *GraphMem) IngestWithContext(ctx context.Context, content string, metadata map[string]any) (*IngestResult, error) {
	if err := gm.ensureInitialized(); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Extract knowledge graph
	nodes, edges, err := gm.kg.Extract(ctx, content, gm.memoryID, gm.userID)
	if err != nil {
		return nil, NewIngestionError("failed to extract knowledge graph").
			WithStage("extraction").
			WithCause(err)
	}

	gm.memoryMu.Lock()
	// Add nodes and edges to memory
	for _, node := range nodes {
		for k, v := range metadata {
			node.Properties[k] = v
		}
		gm.memory.AddNode(node)
	}

	for _, edge := range edges {
		gm.memory.AddEdge(edge)
	}
	gm.memoryMu.Unlock()

	// Save to store
	if err := gm.store.SaveMemory(gm.memory); err != nil {
		return nil, NewStorageError("failed to save memory").
			WithOperation("save").
			WithCause(err)
	}

	// Invalidate cache
	_ = gm.cache.Invalidate(gm.memoryID, gm.userID)

	// Update metrics
	gm.metrics.mu.Lock()
	gm.metrics.Ingestions++
	gm.metrics.TotalNodes = int64(gm.memory.NodeCount())
	gm.metrics.TotalEdges = int64(gm.memory.EdgeCount())
	gm.metrics.mu.Unlock()

	// Auto-evolve if enabled
	if gm.autoEvolve && gm.evolution != nil {
		_, _ = gm.Evolve()
	}

	elapsed := time.Since(startTime).Seconds()

	return &IngestResult{
		Success:        true,
		MemoryID:       gm.memoryID,
		Entities:       len(nodes),
		Relationships:  len(edges),
		Clusters:       0,
		ElapsedSeconds: elapsed,
	}, nil
}

// Query queries the memory.
func (gm *GraphMem) Query(query string) (*MemoryResponse, error) {
	return gm.QueryWithContext(context.Background(), query)
}

// QueryWithContext queries the memory with context.
func (gm *GraphMem) QueryWithContext(ctx context.Context, query string) (*MemoryResponse, error) {
	if err := gm.ensureInitialized(); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Build query
	memQuery := NewMemoryQuery(query)
	memQuery.MemoryID = gm.memoryID

	gm.memoryMu.RLock()
	memoryCopy := gm.memory.Clone()
	gm.memoryMu.RUnlock()

	// Retrieve relevant context
	retrievalResult, err := gm.retriever.Retrieve(ctx, memQuery, memoryCopy)
	if err != nil {
		return nil, NewQueryError("retrieval failed").
			WithQuery(query).
			WithCause(err)
	}

	// Generate answer using LLM
	var answer string
	if len(retrievalResult.Nodes) > 0 {
		prompt := fmt.Sprintf(`Based on the following context, answer the question.

Context:
%s

Question: %s

Answer:`, retrievalResult.Context, query)

		answer, err = gm.llm.Complete(ctx, prompt)
		if err != nil {
			// Return context-based response even if LLM fails
			answer = "Based on the retrieved context: " + retrievalResult.Context
		}
	} else {
		answer = "I don't have information about that in my memory."
	}

	// Record access on retrieved nodes
	gm.memoryMu.Lock()
	for _, node := range retrievalResult.Nodes {
		if memNode := gm.memory.GetNode(node.ID); memNode != nil {
			memNode.RecordAccess()
		}
	}
	gm.memoryMu.Unlock()

	// Update metrics
	gm.metrics.mu.Lock()
	gm.metrics.Queries++
	gm.metrics.mu.Unlock()

	elapsed := time.Since(startTime).Seconds() * 1000

	response := &MemoryResponse{
		Query:      query,
		Answer:     answer,
		Confidence: calculateConfidence(retrievalResult.Scores),
		Nodes:      retrievalResult.Nodes,
		Edges:      retrievalResult.Edges,
		Clusters:   retrievalResult.Clusters,
		Context:    retrievalResult.Context,
		Sources:    extractSources(retrievalResult.Nodes),
		Metadata:   make(map[string]any),
		LatencyMS:  elapsed,
		Timestamp:  time.Now().UTC(),
	}

	return response, nil
}

// Evolve triggers memory evolution.
func (gm *GraphMem) Evolve() ([]*EvolutionEvent, error) {
	return gm.EvolveWithTypes(nil, false)
}

// EvolveWithTypes evolves the memory with specific evolution types.
func (gm *GraphMem) EvolveWithTypes(evolutionTypes []EvolutionType, force bool) ([]*EvolutionEvent, error) {
	if err := gm.ensureInitialized(); err != nil {
		return nil, err
	}

	if gm.evolution == nil {
		return nil, nil
	}

	gm.memoryMu.Lock()
	events := gm.evolution.Evolve(gm.memory, evolutionTypes, force)
	gm.memoryMu.Unlock()

	if len(events) > 0 {
		// Save updated memory
		if err := gm.store.SaveMemory(gm.memory); err != nil {
			return events, NewStorageError("failed to save evolved memory").
				WithOperation("save").
				WithCause(err)
		}

		// Invalidate cache
		_ = gm.cache.Invalidate(gm.memoryID, gm.userID)
	}

	// Update metrics
	gm.metrics.mu.Lock()
	gm.metrics.Evolutions += int64(len(events))
	gm.metrics.mu.Unlock()

	return events, nil
}

// GetStats returns memory statistics.
func (gm *GraphMem) GetStats() map[string]any {
	if err := gm.ensureInitialized(); err != nil {
		return map[string]any{"error": err.Error()}
	}

	gm.memoryMu.RLock()
	defer gm.memoryMu.RUnlock()

	gm.metrics.mu.RLock()
	defer gm.metrics.mu.RUnlock()

	return map[string]any{
		"memory_id":  gm.memoryID,
		"user_id":    gm.userID,
		"nodes":      gm.memory.NodeCount(),
		"edges":      gm.memory.EdgeCount(),
		"clusters":   gm.memory.ClusterCount(),
		"created_at": gm.memory.CreatedAt.Format(time.RFC3339),
		"updated_at": gm.memory.UpdatedAt.Format(time.RFC3339),
		"version":    gm.memory.Version,
		"ingestions": gm.metrics.Ingestions,
		"queries":    gm.metrics.Queries,
		"evolutions": gm.metrics.Evolutions,
	}
}

// GetGraph returns the full knowledge graph.
func (gm *GraphMem) GetGraph() map[string]any {
	if err := gm.ensureInitialized(); err != nil {
		return map[string]any{"error": err.Error()}
	}

	gm.memoryMu.RLock()
	defer gm.memoryMu.RUnlock()

	nodes := make([]map[string]any, 0)
	edges := make([]map[string]any, 0)
	clusters := make([]map[string]any, 0)

	for _, node := range gm.memory.GetAllNodes() {
		nodeMap := map[string]any{
			"id":          node.ID,
			"name":        node.Name,
			"entity_type": node.EntityType,
			"description": node.Description,
			"importance":  node.Importance.String(),
			"state":       node.State.String(),
		}
		nodes = append(nodes, nodeMap)
	}

	for _, edge := range gm.memory.GetAllEdges() {
		edgeMap := map[string]any{
			"id":            edge.ID,
			"source_id":     edge.SourceID,
			"target_id":     edge.TargetID,
			"relation_type": edge.RelationType,
			"description":   edge.Description,
			"weight":        edge.Weight,
		}
		edges = append(edges, edgeMap)
	}

	for _, cluster := range gm.memory.Clusters {
		clusterMap := map[string]any{
			"id":       cluster.ID,
			"summary":  cluster.Summary,
			"entities": cluster.Entities,
		}
		clusters = append(clusters, clusterMap)
	}

	return map[string]any{
		"memory_id": gm.memoryID,
		"nodes":     nodes,
		"edges":     edges,
		"clusters":  clusters,
	}
}

// Clear clears all memory data.
func (gm *GraphMem) Clear() error {
	if err := gm.ensureInitialized(); err != nil {
		return err
	}

	gm.memoryMu.Lock()
	defer gm.memoryMu.Unlock()

	if err := gm.store.ClearMemory(gm.memoryID); err != nil {
		return NewStorageError("failed to clear memory").
			WithOperation("clear").
			WithCause(err)
	}

	_ = gm.cache.Invalidate(gm.memoryID, gm.userID)

	gm.createNewMemory()

	return nil
}

// Save saves the memory to persistent storage.
func (gm *GraphMem) Save() error {
	if err := gm.ensureInitialized(); err != nil {
		return err
	}

	gm.memoryMu.RLock()
	defer gm.memoryMu.RUnlock()

	return gm.store.SaveMemory(gm.memory)
}

// Close closes connections and cleanup resources.
func (gm *GraphMem) Close() error {
	if gm.store != nil {
		if err := gm.store.Close(); err != nil {
			return err
		}
	}
	if gm.cache != nil {
		gm.cache.Close()
	}
	return nil
}

// MemoryID returns the current memory ID.
func (gm *GraphMem) MemoryID() string {
	return gm.memoryID
}

// UserID returns the current user ID.
func (gm *GraphMem) UserID() string {
	return gm.userID
}

// Helper functions

// calculateConfidence calculates confidence from retrieval scores.
func calculateConfidence(scores map[string]float64) float64 {
	if len(scores) == 0 {
		return 0.0
	}

	var sum float64
	for _, score := range scores {
		sum += score
	}
	return sum / float64(len(scores))
}

// extractSources extracts source information from nodes.
func extractSources(nodes []*MemoryNode) []string {
	sources := make([]string, 0)
	seen := make(map[string]bool)

	for _, node := range nodes {
		source := node.Name
		if !seen[source] {
			sources = append(sources, source)
			seen[source] = true
		}
	}

	return sources
}
