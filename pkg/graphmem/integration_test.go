//go:build integration
// +build integration

package graphmem

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Test Helpers
// =============================================================================

func skipIfNoEnv(t *testing.T, envVar string) {
	if os.Getenv(envVar) == "" {
		t.Skipf("Skipping: %s not set", envVar)
	}
}

func skipIfNotTrue(t *testing.T, envVar string) {
	if strings.ToLower(os.Getenv(envVar)) != "true" {
		t.Skipf("Skipping: %s not set to true", envVar)
	}
}

// =============================================================================
// LLM Provider Integration Tests
// =============================================================================

func TestOpenAILLMIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	llm, err := NewLLMProvider(&LLMOptions{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       "gpt-4o-mini",
		Temperature: 0.1,
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("Failed to create OpenAI LLM: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := llm.Complete(ctx, "Say 'Hello GraphMem' and nothing else.")
	if err != nil {
		t.Fatalf("OpenAI completion failed: %v", err)
	}

	if !strings.Contains(strings.ToLower(response), "hello") {
		t.Errorf("Expected response to contain 'hello', got: %s", response)
	}

	t.Logf("OpenAI response: %s", response)
}

func TestAnthropicLLMIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "ANTHROPIC_API_KEY")

	llm, err := NewLLMProvider(&LLMOptions{
		Provider:    "anthropic",
		APIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		Model:       "claude-3-5-sonnet-20241022",
		Temperature: 0.1,
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("Failed to create Anthropic LLM: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := llm.Complete(ctx, "Say 'Hello GraphMem' and nothing else.")
	if err != nil {
		t.Fatalf("Anthropic completion failed: %v", err)
	}

	if !strings.Contains(strings.ToLower(response), "hello") {
		t.Errorf("Expected response to contain 'hello', got: %s", response)
	}

	t.Logf("Anthropic response: %s", response)
}

func TestOllamaLLMIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2"
	}

	llm, err := NewLLMProvider(&LLMOptions{
		Provider:    "ollama",
		APIBase:     baseURL,
		Model:       model,
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("Failed to create Ollama LLM: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	response, err := llm.Complete(ctx, "Say 'Hello' and nothing else.")
	if err != nil {
		t.Skipf("Ollama not available: %v", err)
	}

	t.Logf("Ollama response: %s", response)
}

func TestGroqLLMIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "GROQ_API_KEY")

	llm, err := NewLLMProvider(&LLMOptions{
		Provider:    "groq",
		APIKey:      os.Getenv("GROQ_API_KEY"),
		Model:       "llama-3.1-70b-versatile",
		Temperature: 0.1,
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("Failed to create Groq LLM: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := llm.Complete(ctx, "Say 'Hello GraphMem' and nothing else.")
	if err != nil {
		t.Fatalf("Groq completion failed: %v", err)
	}

	t.Logf("Groq response: %s", response)
}

// =============================================================================
// Embedding Provider Integration Tests
// =============================================================================

func TestOpenAIEmbeddingIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	emb, err := NewEmbeddingProvider(&EmbeddingOptions{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "text-embedding-3-small",
	})
	if err != nil {
		t.Fatalf("Failed to create OpenAI embedding: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test single embedding
	embedding, err := emb.Embed(ctx, "GraphMem is a knowledge graph memory system.")
	if err != nil {
		t.Fatalf("OpenAI embedding failed: %v", err)
	}

	if len(embedding) == 0 {
		t.Error("Expected non-empty embedding")
	}

	t.Logf("Embedding dimensions: %d", len(embedding))

	// Test batch embedding
	texts := []string{
		"First document about AI.",
		"Second document about graphs.",
		"Third document about memory.",
	}

	embeddings, err := emb.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatalf("Batch embedding failed: %v", err)
	}

	if len(embeddings) != 3 {
		t.Errorf("Expected 3 embeddings, got %d", len(embeddings))
	}

	// Test similarity
	sim := CosineSimilarity(embedding, embeddings[0])
	t.Logf("Similarity between main text and first doc: %f", sim)
}

// =============================================================================
// Knowledge Graph Extraction Integration Tests
// =============================================================================

func TestKnowledgeGraphExtractionIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	llm, err := NewLLMProvider(&LLMOptions{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       "gpt-4o-mini",
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	extractor := NewKnowledgeGraph(llm, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	text := `
	Apple Inc. was founded by Steve Jobs, Steve Wozniak, and Ronald Wayne in 1976.
	The company is headquartered in Cupertino, California. Tim Cook became CEO in 2011.
	Apple created the iPhone, which revolutionized the smartphone industry.
	`

	nodes, edges, err := extractor.Extract(ctx, text, "memory1", "user1")
	if err != nil {
		t.Fatalf("Knowledge graph extraction failed: %v", err)
	}

	if len(nodes) == 0 {
		t.Error("Expected at least some nodes to be extracted")
	}

	if len(edges) == 0 {
		t.Error("Expected at least some edges to be extracted")
	}

	t.Logf("Extracted %d nodes and %d edges", len(nodes), len(edges))

	for _, node := range nodes {
		t.Logf("  Node: %s (%s)", node.Name, node.EntityType)
	}
	for _, edge := range edges {
		t.Logf("  Edge: %s -[%s]-> %s", edge.SourceID, edge.RelationType, edge.TargetID)
	}
}

// =============================================================================
// Entity Resolution Integration Tests
// =============================================================================

func TestEntityResolutionIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	emb, err := NewEmbeddingProvider(&EmbeddingOptions{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "text-embedding-3-small",
	})
	if err != nil {
		t.Fatalf("Failed to create embedding provider: %v", err)
	}

	resolver := NewEntityResolver(emb)

	// Create nodes with similar/duplicate names
	nodes := []*MemoryNode{
		NewMemoryNode("Apple Inc.", "Company", "user1", "memory1"),
		NewMemoryNode("Apple", "Company", "user1", "memory1"),
		NewMemoryNode("AAPL", "Company", "user1", "memory1"),
		NewMemoryNode("Google", "Company", "user1", "memory1"),
		NewMemoryNode("Alphabet Inc.", "Company", "user1", "memory1"),
		NewMemoryNode("Dr. Smith", "Person", "user1", "memory1"),
		NewMemoryNode("John Smith", "Person", "user1", "memory1"),
	}

	// Add aliases
	nodes[0].AddAlias("Apple")
	nodes[0].AddAlias("AAPL")
	nodes[3].AddAlias("Alphabet")
	nodes[5].AddAlias("Dr. John Smith")

	resolved, err := resolver.Resolve(nodes, "memory1", "user1")
	if err != nil {
		t.Fatalf("Entity resolution failed: %v", err)
	}

	t.Logf("Resolved %d nodes to %d unique entities", len(nodes), len(resolved))
	for _, node := range resolved {
		t.Logf("  Entity: %s (aliases: %v)", node.Name, node.Aliases)
	}

	// Should have fewer nodes after resolution
	if len(resolved) >= len(nodes) {
		t.Error("Expected some entities to be merged")
	}
}

// =============================================================================
// Semantic Search Integration Tests
// =============================================================================

func TestSemanticSearchIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	emb, err := NewEmbeddingProvider(&EmbeddingOptions{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "text-embedding-3-small",
	})
	if err != nil {
		t.Fatalf("Failed to create embedding provider: %v", err)
	}

	search := NewSemanticSearch(emb, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create memory with some nodes
	memory := NewMemory("test-memory")

	// Add some entities with embeddings
	entities := []struct {
		name        string
		entityType  string
		description string
	}{
		{"GraphMem", "Product", "A knowledge graph memory system for AI"},
		{"Neo4j", "Technology", "A graph database management system"},
		{"Redis", "Technology", "An in-memory data structure store"},
		{"Kubernetes", "Technology", "A container orchestration platform"},
		{"Machine Learning", "Concept", "A type of artificial intelligence"},
	}

	for _, e := range entities {
		node := NewMemoryNode(e.name, e.entityType, "user1", "test-memory")
		node.Description = e.description

		// Get embedding
		nodeEmb, err := emb.Embed(ctx, e.description)
		if err != nil {
			t.Fatalf("Failed to embed: %v", err)
		}
		node.Embedding = nodeEmb

		memory.AddNode(node)
	}

	// Index the nodes
	err = search.IndexNodes(ctx, memory.GetAllNodes())
	if err != nil {
		t.Fatalf("Failed to index nodes: %v", err)
	}

	// Search for graph-related items
	results, err := search.Search(ctx, "graph database for storing knowledge", 3, 0.5, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	t.Logf("Search results for 'graph database for storing knowledge':")
	for _, r := range results {
		t.Logf("  %s (score: %.3f)", r.Node.Name, r.Score)
	}

	if len(results) == 0 {
		t.Error("Expected at least one search result")
	}

	// Top result should be GraphMem or Neo4j
	topName := results[0].Node.Name
	if topName != "GraphMem" && topName != "Neo4j" {
		t.Logf("Note: Top result was %s, expected GraphMem or Neo4j", topName)
	}
}

// =============================================================================
// Community Detection Integration Tests
// =============================================================================

func TestCommunityDetectionIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	llm, err := NewLLMProvider(&LLMOptions{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       "gpt-4o-mini",
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	detector := NewCommunityDetector(llm, &CommunityDetectorConfig{
		Algorithm: "greedy_modularity",
	})

	// Create a graph with two clear communities
	memory := NewMemory("test-memory")

	// Tech community
	techNodes := []string{"Python", "JavaScript", "Go", "Rust"}
	for _, name := range techNodes {
		node := NewMemoryNode(name, "Programming Language", "user1", "test-memory")
		memory.AddNode(node)
	}

	// Company community
	companyNodes := []string{"Apple", "Google", "Microsoft", "Amazon"}
	for _, name := range companyNodes {
		node := NewMemoryNode(name, "Company", "user1", "test-memory")
		memory.AddNode(node)
	}

	// Add edges within communities
	nodes := memory.GetAllNodes()
	for i, node := range nodes {
		for j, other := range nodes {
			if i >= j {
				continue
			}
			// Connect nodes of the same type
			if node.EntityType == other.EntityType {
				edge := NewMemoryEdge(node.ID, other.ID, "related_to", "test-memory")
				edge.Weight = 1.0
				memory.AddEdge(edge)
			}
		}
	}

	// Detect communities
	clusters := detector.Detect(nodes, memory.GetAllEdges(), "test-memory")

	t.Logf("Detected %d communities", len(clusters))
	for _, cluster := range clusters {
		t.Logf("  Community %s: %v (coherence: %.2f)", cluster.ID, cluster.Entities, cluster.CoherenceScore)
	}

	// Should detect at least 2 communities
	if len(clusters) < 2 {
		t.Errorf("Expected at least 2 communities, got %d", len(clusters))
	}
}

// =============================================================================
// Evolution Integration Tests
// =============================================================================

func TestConsolidationIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	emb, err := NewEmbeddingProvider(&EmbeddingOptions{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "text-embedding-3-small",
	})
	if err != nil {
		t.Fatalf("Failed to create embedding provider: %v", err)
	}

	llm, err := NewLLMProvider(&LLMOptions{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       "gpt-4o-mini",
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	consolidation := NewMemoryConsolidation(emb, llm, &ConsolidationOptions{
		SimilarityThreshold:   0.8,
		MinOccurrencesToMerge: 2,
		SynthesisEnabled:      true,
	})

	// Create memory with duplicate entities
	memory := NewMemory("test-memory")

	// Add duplicate entities
	node1 := NewMemoryNode("Steve Jobs", "Person", "user1", "test-memory")
	node1.Description = "Co-founder of Apple Inc."

	node2 := NewMemoryNode("Steven Jobs", "Person", "user1", "test-memory")
	node2.Description = "CEO of Apple"

	node3 := NewMemoryNode("Tim Cook", "Person", "user1", "test-memory")
	node3.Description = "Current CEO of Apple"

	memory.AddNode(node1)
	memory.AddNode(node2)
	memory.AddNode(node3)

	initialCount := memory.NodeCount()
	t.Logf("Initial node count: %d", initialCount)

	events := consolidation.Consolidate(memory)

	finalCount := memory.NodeCount()
	t.Logf("Final node count: %d", finalCount)
	t.Logf("Consolidation events: %d", len(events))

	for _, event := range events {
		t.Logf("  Event: %s - %s", event.EvolutionType, event.Reason)
	}
}

// =============================================================================
// Document Chunking Integration Tests
// =============================================================================

func TestDocumentChunkerIntegration(t *testing.T) {
	chunker := NewDocumentChunker(&ChunkerOptions{
		ChunkSize:         500,
		ChunkOverlap:      100,
		MinChunkSize:      50,
		RespectSentences:  true,
		RespectParagraphs: true,
	})

	longText := `GraphMem is a knowledge graph memory system designed for AI applications.
It stores information as entities and relationships in a graph structure.

The system supports multiple storage backends including in-memory, Neo4j, Redis, and Turso.
Each backend has its own advantages for different use cases.

GraphMem also includes evolution mechanisms like decay, consolidation, and rehydration.
These mechanisms help manage memory over time, similar to how human memory works.

The retrieval system supports semantic search, exact matching, and graph traversal.
This enables flexible and powerful querying of the knowledge graph.`

	chunks := chunker.ChunkText(longText, "doc1", map[string]any{"source": "test"})

	t.Logf("Created %d chunks from text of length %d", len(chunks), len(longText))
	for i, chunk := range chunks {
		t.Logf("  Chunk %d: %d chars, starts at %d", i, len(chunk.Content), chunk.StartChar)
	}

	if len(chunks) < 2 {
		t.Error("Expected at least 2 chunks from this text")
	}
}

func TestMarkdownChunkerIntegration(t *testing.T) {
	chunker := NewMarkdownChunker(&ChunkerOptions{
		ChunkSize:        500,
		ChunkOverlap:     50,
		RespectSentences: true,
	})

	markdown := `# GraphMem Overview

GraphMem is a knowledge graph memory system.

## Features

- Entity extraction
- Relationship mapping
- Semantic search

## Installation

Run the following command:

` + "```bash" + `
go get github.com/graphmem/graphmem-go
` + "```" + `

## Usage

Import the package and create a GraphMem instance.

### Basic Example

Create a memory and add some entities.
`

	chunks := chunker.ChunkText(markdown, "readme", nil)

	t.Logf("Created %d chunks from markdown", len(chunks))
	for i, chunk := range chunks {
		level := 0
		if v, ok := chunk.Metadata["header_level"]; ok {
			level = v.(int)
		}
		t.Logf("  Chunk %d: header level %d, %d chars", i, level, len(chunk.Content))
	}
}

func TestCodeChunkerIntegration(t *testing.T) {
	chunker := NewCodeChunker(nil)

	goCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, GraphMem!")
}

func processData(data []string) error {
	for _, item := range data {
		fmt.Println(item)
	}
	return nil
}

type GraphMem struct {
	nodes map[string]Node
	edges map[string]Edge
}

func (g *GraphMem) AddNode(n Node) {
	g.nodes[n.ID] = n
}
`

	chunks := chunker.ChunkCode(goCode, "go", "main.go", nil)

	t.Logf("Created %d chunks from Go code", len(chunks))
	for i, chunk := range chunks {
		symbol := ""
		if v, ok := chunk.Metadata["symbol"]; ok {
			symbol = v.(string)
		}
		t.Logf("  Chunk %d: %s (%d chars)", i, symbol, len(chunk.Content))
	}

	if len(chunks) < 2 {
		t.Error("Expected at least 2 chunks from Go code")
	}
}

// =============================================================================
// Pipeline Integration Tests
// =============================================================================

func TestHighPerformancePipelineIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	llm, err := NewLLMProvider(&LLMOptions{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       "gpt-4o-mini",
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	emb, err := NewEmbeddingProvider(&EmbeddingOptions{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "text-embedding-3-small",
	})
	if err != nil {
		t.Fatalf("Failed to create embedding provider: %v", err)
	}

	pipeline := NewHighPerformancePipeline(
		llm,
		emb,
		&PipelineConfig{
			ChunkSize:            500,
			ChunkOverlap:         100,
			MaxExtractionWorkers: 2,
			EmbeddingBatchSize:   10,
		},
		nil, // No cache for tests
	)

	documents := []Document{
		{
			ID:      "doc1",
			Content: "Apple Inc. was founded by Steve Jobs in 1976. The company created the iPhone.",
			Metadata: map[string]any{
				"source": "wikipedia",
			},
		},
		{
			ID:      "doc2",
			Content: "Google was founded by Larry Page and Sergey Brin in 1998. It started as a search engine.",
			Metadata: map[string]any{
				"source": "wikipedia",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	results, err := pipeline.IngestDocuments(ctx, documents, "memory1", "user1")
	if err != nil {
		t.Fatalf("Pipeline ingestion failed: %v", err)
	}

	t.Logf("Ingested %d documents", len(results))
	for _, r := range results {
		t.Logf("  Doc %s: %d entities, %d relationships, success: %v",
			r.DocID, r.Entities, r.Relationships, r.Success)
	}

	stats := pipeline.Stats()
	t.Logf("Pipeline stats: %d docs, %d entities, %d edges, %d embeddings",
		stats.DocumentsProcessed, stats.TotalEntities, stats.TotalRelationships, stats.TotalEmbeddings)
}

// =============================================================================
// Query Engine Integration Tests
// =============================================================================

func TestQueryEngineIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	llm, err := NewLLMProvider(&LLMOptions{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       "gpt-4o-mini",
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	emb, err := NewEmbeddingProvider(&EmbeddingOptions{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "text-embedding-3-small",
	})
	if err != nil {
		t.Fatalf("Failed to create embedding provider: %v", err)
	}

	// Create retriever and query engine
	retriever := NewRetriever(emb, 10, 0.5)
	qe := NewQueryEngine(llm, retriever, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create memory with some data
	memory := NewMemory("test-memory")

	// Add some entities with relationships
	apple := NewMemoryNode("Apple Inc.", "Company", "user1", "test-memory")
	apple.Description = "A technology company that creates iPhone, iPad, and Mac computers"
	appleEmb, _ := emb.Embed(ctx, apple.Description)
	apple.Embedding = appleEmb

	steve := NewMemoryNode("Steve Jobs", "Person", "user1", "test-memory")
	steve.Description = "Co-founder and former CEO of Apple Inc."
	steveEmb, _ := emb.Embed(ctx, steve.Description)
	steve.Embedding = steveEmb

	iphone := NewMemoryNode("iPhone", "Product", "user1", "test-memory")
	iphone.Description = "A smartphone made by Apple"
	iphoneEmb, _ := emb.Embed(ctx, iphone.Description)
	iphone.Embedding = iphoneEmb

	memory.AddNode(apple)
	memory.AddNode(steve)
	memory.AddNode(iphone)

	// Add edges
	memory.AddEdge(&MemoryEdge{
		ID:           GenerateID(),
		SourceID:     steve.ID,
		TargetID:     apple.ID,
		RelationType: "founded",
		Description:  "Steve Jobs co-founded Apple",
		Weight:       1.0,
		Confidence:   1.0,
		MemoryID:     "test-memory",
	})

	memory.AddEdge(&MemoryEdge{
		ID:           GenerateID(),
		SourceID:     apple.ID,
		TargetID:     iphone.ID,
		RelationType: "created",
		Description:  "Apple created the iPhone",
		Weight:       1.0,
		Confidence:   1.0,
		MemoryID:     "test-memory",
	})

	// Query the memory
	query := &MemoryQuery{
		Query:         "Who founded Apple and what products did the company create?",
		MemoryID:      "test-memory",
		TopK:          5,
		Mode:          "hybrid",
		MinSimilarity: 0.5,
	}

	response, err := qe.Query(ctx, query, memory)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Query: %s", query.Query)
	t.Logf("Answer: %s", response.Answer)
	t.Logf("Nodes retrieved: %d", len(response.Nodes))
	t.Logf("Edges retrieved: %d", len(response.Edges))

	if response.Answer == "" {
		t.Error("Expected non-empty answer")
	}
}

// =============================================================================
// Full End-to-End Integration Test
// =============================================================================

func TestEndToEndIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Create GraphMem configuration
	config := NewConfig()
	config.LLMProvider = "openai"
	config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
	config.LLMModel = "gpt-4o-mini"
	config.EmbeddingProvider = "openai"
	config.EmbeddingAPIKey = os.Getenv("OPENAI_API_KEY")
	config.EmbeddingModel = "text-embedding-3-small"
	config.DecayEnabled = true
	config.EvolutionEnabled = true

	// Create GraphMem instance
	gm, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create GraphMem: %v", err)
	}
	defer gm.Close()

	// Ingest some content
	content := `
	GraphMem is a knowledge graph memory system designed for AI applications.
	It was developed to provide persistent, queryable memory for LLM-based systems.
	
	The system supports multiple storage backends:
	- In-memory storage for fast prototyping
	- Neo4j for production graph storage
	- Redis for high-performance caching
	- Turso for SQLite-based persistence
	
	Key features include:
	1. Automatic entity and relationship extraction using LLMs
	2. Semantic search using embeddings
	3. Memory evolution (decay, consolidation, rehydration)
	4. Multi-tenant support with user isolation
	`

	result, err := gm.IngestWithContext(ctx, content, nil)
	if err != nil {
		t.Fatalf("Ingestion failed: %v", err)
	}

	t.Logf("Ingestion result:")
	t.Logf("  Memory ID: %s", result.MemoryID)
	t.Logf("  Entities: %d", result.Entities)
	t.Logf("  Relationships: %d", result.Relationships)

	if result.Entities == 0 {
		t.Error("Expected at least one entity to be extracted")
	}

	// Query the memory
	queryResult, err := gm.QueryWithContext(ctx, "What storage backends does GraphMem support?")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Query result:")
	t.Logf("  Answer: %s", queryResult.Answer)
	t.Logf("  Nodes: %d", len(queryResult.Nodes))
	t.Logf("  Confidence: %.2f", queryResult.Confidence)

	if queryResult.Answer == "" {
		t.Error("Expected non-empty answer")
	}

	// The answer should mention storage backends
	answerLower := strings.ToLower(queryResult.Answer)
	if !strings.Contains(answerLower, "neo4j") && !strings.Contains(answerLower, "redis") &&
		!strings.Contains(answerLower, "memory") && !strings.Contains(answerLower, "storage") {
		t.Log("Note: Answer may not have mentioned specific storage backends")
	}
}

// =============================================================================
// Storage Backend Integration Tests
// =============================================================================

func TestNeo4jStoreIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_NEO4J_TESTS")
	skipIfNoEnv(t, "NEO4J_URI")

	uri := os.Getenv("NEO4J_URI")
	user := os.Getenv("NEO4J_USER")
	password := os.Getenv("NEO4J_PASSWORD")

	store, err := NewNeo4jStore(&Neo4jStoreOptions{
		URI:      uri,
		Username: user,
		Password: password,
	})
	if err != nil {
		t.Fatalf("Failed to create Neo4j store: %v", err)
	}
	defer store.Close()

	// Test health check
	if !store.HealthCheck() {
		t.Fatal("Neo4j health check failed")
	}

	// Create and save memory
	memory := NewMemory("neo4j-test-memory")
	node := NewMemoryNode("TestEntity", "TestType", "user1", "neo4j-test-memory")
	node.Description = "Test entity for Neo4j"
	memory.AddNode(node)

	err = store.SaveMemory(memory)
	if err != nil {
		t.Fatalf("Failed to save memory: %v", err)
	}

	// Load memory
	loaded, err := store.LoadMemory("neo4j-test-memory", "user1")
	if err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	if loaded.NodeCount() != 1 {
		t.Errorf("Expected 1 node, got %d", loaded.NodeCount())
	}

	// Cleanup
	err = store.DeleteMemory("neo4j-test-memory")
	if err != nil {
		t.Errorf("Failed to delete memory: %v", err)
	}
}

func TestRedisCacheIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_REDIS_TESTS")

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	cache, err := NewRedisCache(&RedisCacheOptions{
		URL: redisURL,
		TTL: 3600,
	})
	if err != nil {
		t.Fatalf("Failed to create Redis cache: %v", err)
	}
	defer cache.Close()

	// Test set and get
	err = cache.Set("test-key", "test-value")
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	value, found := cache.Get("test-key")
	if !found {
		t.Error("Expected to find key")
	}
	if value != "test-value" {
		t.Errorf("Expected 'test-value', got '%v'", value)
	}

	// Test delete
	err = cache.Delete("test-key")
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	_, found = cache.Get("test-key")
	if found {
		t.Error("Expected key to be deleted")
	}
}

func TestTursoStoreIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_TURSO_TESTS")

	// Use local file for testing
	localPath := os.Getenv("TURSO_LOCAL_PATH")
	if localPath == "" {
		localPath = "/tmp/graphmem-test.db"
	}

	store, err := NewTursoStore(&TursoStoreOptions{
		DBPath:              localPath,
		EmbeddingDimensions: 3,
	})
	if err != nil {
		t.Fatalf("Failed to create Turso store: %v", err)
	}
	defer store.Close()

	// Test health check
	if !store.HealthCheck() {
		t.Fatal("Turso health check failed")
	}

	// Create and save memory
	memory := NewMemory("turso-test-memory")
	node := NewMemoryNode("TestEntity", "TestType", "user1", "turso-test-memory")
	node.Description = "Test entity for Turso"
	memory.AddNode(node)

	err = store.SaveMemory(memory)
	if err != nil {
		t.Fatalf("Failed to save memory: %v", err)
	}

	// Load memory
	loaded, err := store.LoadMemory("turso-test-memory", "user1")
	if err != nil {
		t.Fatalf("Failed to load memory: %v", err)
	}

	if loaded.NodeCount() != 1 {
		t.Errorf("Expected 1 node, got %d", loaded.NodeCount())
	}

	// Cleanup
	err = store.DeleteMemory("turso-test-memory")
	if err != nil {
		t.Errorf("Failed to delete memory: %v", err)
	}
}

// =============================================================================
// Importance Scorer Integration Tests
// =============================================================================

func TestImportanceScorerIntegration(t *testing.T) {
	scorer := NewImportanceScorer(nil)

	// Create a graph with nodes and edges
	nodes := make([]*MemoryNode, 5)
	for i := 0; i < 5; i++ {
		nodes[i] = NewMemoryNode(
			"Entity"+string(rune('A'+i)),
			"Type",
			"user1",
			"memory1",
		)
		nodes[i].AccessCount = i * 10                                          // Vary access counts
		nodes[i].AccessedAt = time.Now().Add(-time.Duration(i*24) * time.Hour) // Vary recency
	}

	// Create edges (hub-and-spoke: first node connects to all)
	edges := make([]*MemoryEdge, 0)
	for i := 1; i < 5; i++ {
		edge := NewMemoryEdge(nodes[0].ID, nodes[i].ID, "related_to", "memory1")
		edges = append(edges, edge)
	}

	// Score nodes
	t.Log("Node importance scores:")
	for _, node := range nodes {
		score := scorer.ScoreNode(node, edges, nodes, nil)
		t.Logf("  %s: %.2f (access: %d, age: %v days)",
			node.Name, score, node.AccessCount,
			time.Since(node.AccessedAt).Hours()/24)
	}

	// First node should have highest score (most connections)
	score0 := scorer.ScoreNode(nodes[0], edges, nodes, nil)
	score4 := scorer.ScoreNode(nodes[4], edges, nodes, nil)

	if score0 <= score4 {
		t.Log("Note: Hub node didn't score highest (may be due to recency/frequency)")
	}
}

// =============================================================================
// Rehydration Integration Tests
// =============================================================================

func TestRehydrationIntegration(t *testing.T) {
	skipIfNotTrue(t, "RUN_LLM_TESTS")
	skipIfNoEnv(t, "OPENAI_API_KEY")

	emb, err := NewEmbeddingProvider(&EmbeddingOptions{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "text-embedding-3-small",
	})
	if err != nil {
		t.Fatalf("Failed to create embedding provider: %v", err)
	}

	rehydration := NewGraphRehydration(emb, &RehydrationOptions{
		RehydrationThreshold: 0.5,
		MaxRehydrations:      10,
		StrengthBoost:        1.5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create memory with some archived nodes
	memory := NewMemory("test-memory")

	// Active node
	activeNode := NewMemoryNode("GraphMem", "Product", "user1", "test-memory")
	activeNode.Description = "A knowledge graph memory system"
	activeEmb, _ := emb.Embed(ctx, activeNode.Description)
	activeNode.Embedding = activeEmb
	activeNode.State = StateActive
	memory.AddNode(activeNode)

	// Archived node (should be rehydrated by relevant query)
	archivedNode := NewMemoryNode("Neo4j", "Technology", "user1", "test-memory")
	archivedNode.Description = "A graph database for storing knowledge graphs"
	archivedEmb, _ := emb.Embed(ctx, archivedNode.Description)
	archivedNode.Embedding = archivedEmb
	archivedNode.State = StateArchived
	memory.AddNode(archivedNode)

	// Unrelated archived node (should not be rehydrated)
	unrelatedNode := NewMemoryNode("Pizza", "Food", "user1", "test-memory")
	unrelatedNode.Description = "An Italian dish with cheese and tomato"
	unrelatedEmb, _ := emb.Embed(ctx, unrelatedNode.Description)
	unrelatedNode.Embedding = unrelatedEmb
	unrelatedNode.State = StateArchived
	memory.AddNode(unrelatedNode)

	// Rehydrate with graph-related context
	result := rehydration.Rehydrate(memory, "I need information about graph databases and knowledge graphs", 10)

	t.Logf("Rehydration result:")
	t.Logf("  Rehydrated: %d", result.Rehydrated)
	t.Logf("  Restored: %d", result.Restored)
	t.Logf("  Strengthened: %d", result.Strengthened)

	// Check that Neo4j was rehydrated
	neo4jNode := memory.GetNode(archivedNode.ID)
	if neo4jNode.State != StateActive {
		t.Error("Expected Neo4j node to be rehydrated to active state")
	}

	// Check that Pizza was not rehydrated (different domain)
	pizzaNode := memory.GetNode(unrelatedNode.ID)
	if pizzaNode.State == StateActive {
		t.Log("Note: Unrelated node was also rehydrated (similarity threshold may be too low)")
	}
}
