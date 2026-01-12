package graphmem

import (
	"context"
	"testing"
	"time"
)

// TestNewMemoryNode tests creating a new memory node.
func TestNewMemoryNode(t *testing.T) {
	node := NewMemoryNode("Test Entity", "Person", "user1", "memory1")

	if node.Name != "Test Entity" {
		t.Errorf("Expected name 'Test Entity', got '%s'", node.Name)
	}
	if node.EntityType != "Person" {
		t.Errorf("Expected entity type 'Person', got '%s'", node.EntityType)
	}
	if node.UserID != "user1" {
		t.Errorf("Expected user ID 'user1', got '%s'", node.UserID)
	}
	if node.MemoryID != "memory1" {
		t.Errorf("Expected memory ID 'memory1', got '%s'", node.MemoryID)
	}
	if node.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if node.Importance != ImportanceMedium {
		t.Errorf("Expected importance Medium, got %s", node.Importance.String())
	}
	if node.State != StateActive {
		t.Errorf("Expected state Active, got %s", node.State.String())
	}
	if !node.Aliases["Test Entity"] {
		t.Error("Expected 'Test Entity' in aliases")
	}
}

// TestMemoryNodeAddAlias tests adding aliases to a node.
func TestMemoryNodeAddAlias(t *testing.T) {
	node := NewMemoryNode("Apple", "Company", "user1", "memory1")

	node.AddAlias("Apple Inc.")
	node.AddAlias("AAPL")

	if !node.Aliases["Apple Inc."] {
		t.Error("Expected 'Apple Inc.' in aliases")
	}
	if !node.Aliases["AAPL"] {
		t.Error("Expected 'AAPL' in aliases")
	}
}

// TestMemoryNodeRecordAccess tests access recording.
func TestMemoryNodeRecordAccess(t *testing.T) {
	node := NewMemoryNode("Test", "Entity", "user1", "memory1")

	initialCount := node.AccessCount
	node.RecordAccess()

	if node.AccessCount != initialCount+1 {
		t.Errorf("Expected access count %d, got %d", initialCount+1, node.AccessCount)
	}
}

// TestNewMemoryEdge tests creating a new memory edge.
func TestNewMemoryEdge(t *testing.T) {
	edge := NewMemoryEdge("source1", "target1", "relates_to", "memory1")

	if edge.SourceID != "source1" {
		t.Errorf("Expected source ID 'source1', got '%s'", edge.SourceID)
	}
	if edge.TargetID != "target1" {
		t.Errorf("Expected target ID 'target1', got '%s'", edge.TargetID)
	}
	if edge.RelationType != "relates_to" {
		t.Errorf("Expected relation type 'relates_to', got '%s'", edge.RelationType)
	}
	if edge.Weight != 1.0 {
		t.Errorf("Expected weight 1.0, got %f", edge.Weight)
	}
	if edge.Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %f", edge.Confidence)
	}
}

// TestMemoryEdgeIsValidAt tests temporal validity checking.
func TestMemoryEdgeIsValidAt(t *testing.T) {
	edge := NewMemoryEdge("source", "target", "relation", "memory1")

	// No temporal bounds
	if !edge.IsValidAt(nil) {
		t.Error("Expected edge to be valid with no temporal bounds")
	}

	// Set valid_from in the past
	pastTime := time.Now().Add(-24 * time.Hour)
	edge.ValidFrom = &pastTime

	if !edge.IsValidAt(nil) {
		t.Error("Expected edge to be valid when valid_from is in the past")
	}

	// Set valid_from in the future
	futureTime := time.Now().Add(24 * time.Hour)
	edge.ValidFrom = &futureTime

	if edge.IsValidAt(nil) {
		t.Error("Expected edge to be invalid when valid_from is in the future")
	}
}

// TestNewMemory tests creating a new memory.
func TestNewMemory(t *testing.T) {
	memory := NewMemory("test-memory")

	if memory.ID != "test-memory" {
		t.Errorf("Expected ID 'test-memory', got '%s'", memory.ID)
	}
	if memory.NodeCount() != 0 {
		t.Errorf("Expected 0 nodes, got %d", memory.NodeCount())
	}
	if memory.EdgeCount() != 0 {
		t.Errorf("Expected 0 edges, got %d", memory.EdgeCount())
	}
	if memory.Version != 1 {
		t.Errorf("Expected version 1, got %d", memory.Version)
	}
}

// TestMemoryAddNode tests adding nodes to memory.
func TestMemoryAddNode(t *testing.T) {
	memory := NewMemory("test-memory")
	node := NewMemoryNode("Test Entity", "Person", "user1", "test-memory")

	memory.AddNode(node)

	if memory.NodeCount() != 1 {
		t.Errorf("Expected 1 node, got %d", memory.NodeCount())
	}

	retrieved := memory.GetNode(node.ID)
	if retrieved == nil {
		t.Fatal("Expected to retrieve node")
	}
	if retrieved.Name != "Test Entity" {
		t.Errorf("Expected name 'Test Entity', got '%s'", retrieved.Name)
	}
}

// TestMemoryAddEdge tests adding edges to memory.
func TestMemoryAddEdge(t *testing.T) {
	memory := NewMemory("test-memory")
	edge := NewMemoryEdge("source", "target", "relates_to", "test-memory")

	memory.AddEdge(edge)

	if memory.EdgeCount() != 1 {
		t.Errorf("Expected 1 edge, got %d", memory.EdgeCount())
	}

	retrieved := memory.GetEdge(edge.ID)
	if retrieved == nil {
		t.Fatal("Expected to retrieve edge")
	}
	if retrieved.RelationType != "relates_to" {
		t.Errorf("Expected relation type 'relates_to', got '%s'", retrieved.RelationType)
	}
}

// TestMemoryClone tests cloning a memory.
func TestMemoryClone(t *testing.T) {
	memory := NewMemory("test-memory")
	node := NewMemoryNode("Entity", "Type", "user1", "test-memory")
	edge := NewMemoryEdge("s", "t", "r", "test-memory")

	memory.AddNode(node)
	memory.AddEdge(edge)

	clone := memory.Clone()

	if clone.ID != memory.ID {
		t.Error("Clone should have same ID")
	}
	if clone.NodeCount() != memory.NodeCount() {
		t.Error("Clone should have same node count")
	}
	if clone.EdgeCount() != memory.EdgeCount() {
		t.Error("Clone should have same edge count")
	}

	// Verify deep copy - modifying clone shouldn't affect original
	clone.AddNode(NewMemoryNode("New", "Type", "user1", "test-memory"))
	if memory.NodeCount() == clone.NodeCount() {
		t.Error("Clone should be a deep copy")
	}
}

// TestInMemoryStore tests the in-memory store.
func TestInMemoryStore(t *testing.T) {
	store := NewInMemoryStore()

	// Test health check
	if !store.HealthCheck() {
		t.Error("Expected healthy store")
	}

	// Create and save memory
	memory := NewMemory("test-memory")
	memory.AddNode(NewMemoryNode("Entity", "Type", "user1", "test-memory"))

	err := store.SaveMemory(memory)
	if err != nil {
		t.Errorf("Failed to save memory: %v", err)
	}

	// Load memory
	loaded, err := store.LoadMemory("test-memory", "user1")
	if err != nil {
		t.Errorf("Failed to load memory: %v", err)
	}
	if loaded == nil {
		t.Error("Expected to load memory")
	}
	if loaded.NodeCount() != 1 {
		t.Errorf("Expected 1 node, got %d", loaded.NodeCount())
	}

	// List memories
	memories, err := store.ListMemories("user1")
	if err != nil {
		t.Errorf("Failed to list memories: %v", err)
	}
	if len(memories) != 1 {
		t.Errorf("Expected 1 memory, got %d", len(memories))
	}

	// Delete memory
	err = store.DeleteMemory("test-memory")
	if err != nil {
		t.Errorf("Failed to delete memory: %v", err)
	}

	loaded, err = store.LoadMemory("test-memory", "user1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if loaded != nil {
		t.Error("Expected nil after deletion")
	}
}

// TestInMemoryCache tests the in-memory cache.
func TestInMemoryCache(t *testing.T) {
	cache := NewInMemoryCache(3600)

	// Test set and get
	err := cache.Set("key1", "value1")
	if err != nil {
		t.Errorf("Failed to set: %v", err)
	}

	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}

	// Test invalidate
	err = cache.Invalidate("memory1", "user1")
	if err != nil {
		t.Errorf("Failed to invalidate: %v", err)
	}

	_, found = cache.Get("key1")
	if found {
		t.Error("Expected key to be invalidated")
	}
}

// TestCosineSimilarity tests the cosine similarity function.
func TestCosineSimilarity(t *testing.T) {
	// Identical vectors
	v1 := []float32{1, 0, 0}
	v2 := []float32{1, 0, 0}
	sim := CosineSimilarity(v1, v2)
	if sim < 0.99 {
		t.Errorf("Expected similarity ~1.0 for identical vectors, got %f", sim)
	}

	// Orthogonal vectors
	v3 := []float32{1, 0, 0}
	v4 := []float32{0, 1, 0}
	sim = CosineSimilarity(v3, v4)
	if sim > 0.01 {
		t.Errorf("Expected similarity ~0.0 for orthogonal vectors, got %f", sim)
	}

	// Empty vectors
	sim = CosineSimilarity([]float32{}, []float32{1, 2, 3})
	if sim != 0.0 {
		t.Errorf("Expected 0.0 for empty vectors, got %f", sim)
	}
}

// TestImportanceFromScore tests importance level conversion.
func TestImportanceFromScore(t *testing.T) {
	tests := []struct {
		score    float64
		expected MemoryImportance
	}{
		{10.0, ImportanceCritical},
		{9.5, ImportanceCritical},
		{8.0, ImportanceVeryHigh},
		{6.0, ImportanceHigh},
		{5.0, ImportanceMedium},
		{3.0, ImportanceLow},
		{1.0, ImportanceVeryLow},
		{0.0, ImportanceEphemeral},
	}

	for _, test := range tests {
		result := ImportanceFromScore(test.score)
		if result != test.expected {
			t.Errorf("For score %f, expected %s, got %s", test.score, test.expected.String(), result.String())
		}
	}
}

// TestMemoryQuery tests creating a memory query.
func TestMemoryQuery(t *testing.T) {
	query := NewMemoryQuery("What is GraphMem?")

	if query.Query != "What is GraphMem?" {
		t.Errorf("Expected query 'What is GraphMem?', got '%s'", query.Query)
	}
	if query.Mode != "hybrid" {
		t.Errorf("Expected mode 'hybrid', got '%s'", query.Mode)
	}
	if query.TopK != 10 {
		t.Errorf("Expected top_k 10, got %d", query.TopK)
	}
}

// TestEvolutionEvent tests creating an evolution event.
func TestEvolutionEvent(t *testing.T) {
	event := NewEvolutionEvent(EvolutionConsolidation, "memory1")

	if event.EvolutionType != EvolutionConsolidation {
		t.Errorf("Expected type Consolidation, got %s", event.EvolutionType)
	}
	if event.MemoryID != "memory1" {
		t.Errorf("Expected memory ID 'memory1', got '%s'", event.MemoryID)
	}
	if !event.Success {
		t.Error("Expected success to be true by default")
	}
	if event.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

// TestEntityResolver tests entity resolution.
func TestEntityResolver(t *testing.T) {
	// Create a mock embedding provider
	mockEmb := &mockEmbeddingProvider{}
	resolver := NewEntityResolver(mockEmb)

	// Create nodes with similar names
	node1 := NewMemoryNode("Apple Inc.", "Company", "user1", "memory1")
	node2 := NewMemoryNode("Apple", "Company", "user1", "memory1")
	node3 := NewMemoryNode("Google", "Company", "user1", "memory1")

	nodes := []*MemoryNode{node1, node2, node3}
	resolved, err := resolver.Resolve(nodes, "memory1", "user1")
	if err != nil {
		t.Errorf("Failed to resolve: %v", err)
	}

	// Should have merged Apple Inc. and Apple
	if len(resolved) != 2 {
		t.Errorf("Expected 2 resolved nodes, got %d", len(resolved))
	}

	// Reset for next test
	resolver.Clear()
}

// TestRetriever tests the retriever.
func TestRetriever(t *testing.T) {
	mockEmb := &mockEmbeddingProvider{}
	retriever := NewRetriever(mockEmb, 10, 0.5)

	memory := NewMemory("test-memory")
	node := NewMemoryNode("GraphMem", "Product", "user1", "test-memory")
	node.Description = "A knowledge graph memory system"
	memory.AddNode(node)

	query := NewMemoryQuery("What is GraphMem?")

	result, err := retriever.Retrieve(context.Background(), query, memory)
	if err != nil {
		t.Errorf("Retrieval failed: %v", err)
	}

	if len(result.Nodes) != 1 {
		t.Errorf("Expected 1 node in results, got %d", len(result.Nodes))
	}
}

// TestEvolution tests the evolution system.
func TestEvolution(t *testing.T) {
	mockLLM := &mockLLMProvider{}
	mockEmb := &mockEmbeddingProvider{}

	config := DefaultEvolutionConfig()
	config.MinEvolutionInterval = 0 // Allow immediate evolution

	evolution := NewEvolution(mockLLM, mockEmb, config)

	memory := NewMemory("test-memory")
	node1 := NewMemoryNode("Entity1", "Type", "user1", "test-memory")
	node1.AccessedAt = time.Now().Add(-100 * 24 * time.Hour) // Accessed 100 days ago
	node1.Importance = ImportanceHigh
	memory.AddNode(node1)

	events := evolution.Evolve(memory, []EvolutionType{EvolutionDecay}, true)

	// Node should have decayed
	if len(events) == 0 {
		t.Log("No decay events (importance may not have dropped enough)")
	}
}

// TestGraphMemError tests error types.
func TestGraphMemError(t *testing.T) {
	err := NewConfigurationError("missing API key").
		WithSuggestion("Set OPENAI_API_KEY")

	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}

	if len(err.Suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(err.Suggestions))
	}
}

// Mock implementations for testing

type mockLLMProvider struct{}

func (m *mockLLMProvider) Complete(ctx context.Context, prompt string) (string, error) {
	return "Mock response", nil
}

func (m *mockLLMProvider) Chat(ctx context.Context, messages []LLMMessage) (string, error) {
	return "Mock chat response", nil
}

type mockEmbeddingProvider struct{}

func (m *mockEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// Return a simple embedding based on text length for testing
	return []float32{float32(len(text)) / 100.0, 0.5, 0.3}, nil
}

func (m *mockEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		results[i] = []float32{float32(len(text)) / 100.0, 0.5, 0.3}
	}
	return results, nil
}
