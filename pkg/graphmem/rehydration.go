package graphmem

import (
	"context"
	"log"
	"sort"
	"time"
)

// scoredNode represents a node with its relevance score.
type scoredNode struct {
	Node  *MemoryNode
	Score float64
}

// GraphRehydration rehydrates (restores) memories based on context.
type GraphRehydration struct {
	embeddings           EmbeddingProvider
	rehydrationThreshold float64
	maxRehydrations      int
	strengthBoost        float64
}

// RehydrationOptions contains options for GraphRehydration.
type RehydrationOptions struct {
	RehydrationThreshold float64
	MaxRehydrations      int
	StrengthBoost        float64
}

// RehydrationResult contains statistics about rehydration.
type RehydrationResult struct {
	Rehydrated      int `json:"rehydrated"`
	Restored        int `json:"restored"`
	Strengthened    int `json:"strengthened"`
	EdgesRehydrated int `json:"edges_rehydrated"`
}

// NewGraphRehydration creates a new GraphRehydration instance.
func NewGraphRehydration(embeddings EmbeddingProvider, opts *RehydrationOptions) *GraphRehydration {
	if opts == nil {
		opts = &RehydrationOptions{
			RehydrationThreshold: 0.75,
			MaxRehydrations:      100,
			StrengthBoost:        1.5,
		}
	}

	return &GraphRehydration{
		embeddings:           embeddings,
		rehydrationThreshold: opts.RehydrationThreshold,
		maxRehydrations:      opts.MaxRehydrations,
		strengthBoost:        opts.StrengthBoost,
	}
}

// Rehydrate rehydrates memories based on context.
func (gr *GraphRehydration) Rehydrate(memory *Memory, contextText string, maxNodes int) *RehydrationResult {
	result := &RehydrationResult{}

	if contextText == "" {
		return result
	}

	if maxNodes == 0 {
		maxNodes = 100
	}

	// Get context embedding
	if gr.embeddings == nil {
		return result
	}

	contextEmbedding, err := gr.embeddings.Embed(context.Background(), contextText)
	if err != nil || len(contextEmbedding) == 0 {
		log.Printf("Failed to get context embedding: %v", err)
		return result
	}

	// Score all nodes
	scoredNodes := make([]scoredNode, 0)

	for _, node := range memory.Nodes {
		score := gr.scoreRelevance(node, contextEmbedding)
		if score >= gr.rehydrationThreshold {
			scoredNodes = append(scoredNodes, scoredNode{Node: node, Score: score})
		}
	}

	// Sort by score and limit
	sort.Slice(scoredNodes, func(i, j int) bool {
		return scoredNodes[i].Score > scoredNodes[j].Score
	})
	if len(scoredNodes) > maxNodes {
		scoredNodes = scoredNodes[:maxNodes]
	}

	// Rehydrate relevant nodes
	for _, sn := range scoredNodes {
		node := sn.Node

		switch node.State {
		case StateArchived, StateDecaying:
			// Restore archived/decaying memory
			updatedNode := node.Clone()
			updatedNode.State = StateActive
			updatedNode.AccessedAt = time.Now().UTC()
			updatedNode.AccessCount++
			memory.Nodes[node.ID] = updatedNode
			result.Restored++
			result.Rehydrated++

		case StateActive:
			// Strengthen active memory
			updatedNode := node.Clone()
			updatedNode.AccessedAt = time.Now().UTC()
			updatedNode.AccessCount++
			memory.Nodes[node.ID] = updatedNode
			result.Strengthened++
			result.Rehydrated++
		}

		if result.Rehydrated >= gr.maxRehydrations {
			break
		}
	}

	// Also rehydrate connected edges
	result.EdgesRehydrated = gr.rehydrateEdges(memory, scoredNodes)

	log.Printf(
		"Rehydration complete: %d nodes (%d restored, %d strengthened), %d edges",
		result.Rehydrated, result.Restored, result.Strengthened, result.EdgesRehydrated,
	)

	return result
}

// scoreRelevance scores how relevant a node is to the context.
func (gr *GraphRehydration) scoreRelevance(node *MemoryNode, contextEmbedding []float32) float64 {
	// Get node text
	text := node.Description
	if text == "" {
		text = node.Name
	}

	if gr.embeddings == nil {
		return 0.0
	}
	nodeEmbedding, err := gr.embeddings.Embed(context.Background(), text)
	if err != nil || len(nodeEmbedding) == 0 {
		return 0.0
	}

	// Cosine similarity
	similarity := cosineSimilarity32(contextEmbedding, nodeEmbedding)

	// Boost for archived memories (they need more help)
	if node.State == StateArchived {
		similarity *= 1.1
		if similarity > 1.0 {
			similarity = 1.0
		}
	}

	return float64(similarity)
}

// rehydrateEdges rehydrates edges connected to rehydrated nodes.
func (gr *GraphRehydration) rehydrateEdges(memory *Memory, nodes []scoredNode) int {
	rehydratedNodeIDs := make(map[string]bool)
	for _, sn := range nodes {
		rehydratedNodeIDs[sn.Node.ID] = true
	}

	rehydratedCount := 0

	for edgeID, edge := range memory.Edges {
		if edge.State == StateActive {
			continue
		}

		// Check if connected to rehydrated nodes
		if rehydratedNodeIDs[edge.SourceID] || rehydratedNodeIDs[edge.TargetID] {
			updatedEdge := edge.Clone()
			updatedEdge.State = StateActive
			updatedEdge.AccessedAt = time.Now().UTC()
			updatedEdge.AccessCount++
			memory.Edges[edgeID] = updatedEdge
			rehydratedCount++
		}
	}

	return rehydratedCount
}
