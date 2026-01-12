package graphmem

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// RetrievalResult holds retrieval results.
type RetrievalResult struct {
	Nodes    []*MemoryNode
	Edges    []*MemoryEdge
	Clusters []*MemoryCluster
	Scores   map[string]float64
	Context  string
}

// Retriever retrieves relevant memories.
type Retriever struct {
	embeddings    EmbeddingProvider
	topK          int
	minSimilarity float64
}

// NewRetriever creates a new Retriever.
func NewRetriever(embeddings EmbeddingProvider, topK int, minSimilarity float64) *Retriever {
	if topK <= 0 {
		topK = 10
	}
	if minSimilarity <= 0 {
		minSimilarity = 0.5
	}

	return &Retriever{
		embeddings:    embeddings,
		topK:          topK,
		minSimilarity: minSimilarity,
	}
}

// Retrieve retrieves relevant memories for a query.
func (r *Retriever) Retrieve(ctx context.Context, query *MemoryQuery, memory *Memory) (*RetrievalResult, error) {
	if query == nil || memory == nil {
		return &RetrievalResult{Scores: make(map[string]float64)}, nil
	}

	result := &RetrievalResult{
		Scores: make(map[string]float64),
	}

	// Get query embedding
	if r.embeddings == nil {
		return nil, fmt.Errorf("embedding provider not configured")
	}
	queryEmb, err := r.embeddings.Embed(ctx, query.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Score nodes
	type scoredNode struct {
		node  *MemoryNode
		score float64
	}
	scoredNodes := make([]scoredNode, 0)

	for _, node := range memory.GetAllNodes() {
		if node.State != StateActive {
			continue
		}

		// Check importance filter
		if node.Importance < query.MinImportance {
			continue
		}

		score := r.scoreNode(node, query.Query, queryEmb)
		if score >= r.minSimilarity {
			scoredNodes = append(scoredNodes, scoredNode{node: node, score: score})
			result.Scores[node.ID] = score
		}
	}

	// Sort by score
	sort.Slice(scoredNodes, func(i, j int) bool {
		return scoredNodes[i].score > scoredNodes[j].score
	})

	// Take top K
	topK := query.TopK
	if topK <= 0 {
		topK = r.topK
	}
	if len(scoredNodes) > topK {
		scoredNodes = scoredNodes[:topK]
	}

	// Collect nodes
	nodeIDs := make(map[string]bool)
	for _, sn := range scoredNodes {
		result.Nodes = append(result.Nodes, sn.node)
		nodeIDs[sn.node.ID] = true
	}

	// Find relevant edges
	for _, edge := range memory.GetAllEdges() {
		if edge.State != StateActive {
			continue
		}

		// Include edge if both source and target are in results
		if nodeIDs[edge.SourceID] && nodeIDs[edge.TargetID] {
			result.Edges = append(result.Edges, edge)
		}
	}

	// Build context
	if query.IncludeContext {
		result.Context = r.buildContext(result.Nodes, result.Edges)
	}

	return result, nil
}

func (r *Retriever) scoreNode(node *MemoryNode, query string, queryEmb []float32) float64 {
	queryLower := strings.ToLower(query)
	nameLower := strings.ToLower(node.Name)

	// Exact name match
	if strings.Contains(queryLower, nameLower) || strings.Contains(nameLower, queryLower) {
		return 1.0
	}

	// Alias match
	for alias := range node.Aliases {
		aliasLower := strings.ToLower(alias)
		if strings.Contains(queryLower, aliasLower) || strings.Contains(aliasLower, queryLower) {
			return 0.95
		}
	}

	// Embedding similarity
	if len(node.Embedding) > 0 && len(queryEmb) > 0 {
		return CosineSimilarity(queryEmb, node.Embedding)
	}

	// Token overlap
	queryTokens := retrieverTokenize(queryLower)
	nameTokens := retrieverTokenize(nameLower)

	if len(queryTokens) > 0 && len(nameTokens) > 0 {
		overlap := 0
		for token := range queryTokens {
			if nameTokens[token] {
				overlap++
			}
		}
		if overlap > 0 {
			return 0.5 + 0.3*float64(overlap)/float64(len(queryTokens))
		}
	}

	return 0.0
}

func (r *Retriever) buildContext(nodes []*MemoryNode, edges []*MemoryEdge) string {
	var sb strings.Builder

	// Entities
	if len(nodes) > 0 {
		sb.WriteString("Entities:\n")
		for _, node := range nodes {
			sb.WriteString(fmt.Sprintf("- %s (%s)", node.Name, node.EntityType))
			if node.Description != "" {
				sb.WriteString(": " + node.Description)
			}
			sb.WriteString("\n")
		}
	}

	// Relationships
	if len(edges) > 0 {
		sb.WriteString("\nRelationships:\n")
		for _, edge := range edges {
			sb.WriteString(fmt.Sprintf("- %s %s %s", edge.SourceID, edge.RelationType, edge.TargetID))
			if edge.Description != "" {
				sb.WriteString(": " + edge.Description)
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func retrieverTokenize(s string) map[string]bool {
	tokens := make(map[string]bool)
	words := strings.Fields(s)
	for _, word := range words {
		cleaned := strings.Trim(word, ".,!?;:'\"")
		if len(cleaned) > 2 {
			tokens[strings.ToLower(cleaned)] = true
		}
	}
	return tokens
}

