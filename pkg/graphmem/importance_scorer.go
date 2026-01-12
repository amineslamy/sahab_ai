package graphmem

import (
	"log"
	"math"
	"sync"
	"time"
)

// ImportanceScorer calculates importance scores for memory elements.
// Uses the formula: ρ(e) = w1·f1(e) + w2·f2(e) + w3·f3(e) + w4·f4(e)
type ImportanceScorer struct {
	recencyWeight   float64
	frequencyWeight float64
	pagerankWeight  float64
	userWeight      float64
	pagerankDamping float64

	// PageRank cache
	pagerankCache     map[string]float64
	pagerankCacheTime time.Time
	pagerankCacheTTL  float64
	mu                sync.RWMutex
}

// ImportanceScorerOptions contains options for ImportanceScorer.
type ImportanceScorerOptions struct {
	RecencyWeight   float64
	FrequencyWeight float64
	PagerankWeight  float64
	UserWeight      float64
	PagerankDamping float64
}

// NewImportanceScorer creates a new ImportanceScorer.
func NewImportanceScorer(opts *ImportanceScorerOptions) *ImportanceScorer {
	if opts == nil {
		opts = &ImportanceScorerOptions{
			RecencyWeight:   0.3,
			FrequencyWeight: 0.3,
			PagerankWeight:  0.2,
			UserWeight:      0.2,
			PagerankDamping: 0.85,
		}
	}

	return &ImportanceScorer{
		recencyWeight:    opts.RecencyWeight,
		frequencyWeight:  opts.FrequencyWeight,
		pagerankWeight:   opts.PagerankWeight,
		userWeight:       opts.UserWeight,
		pagerankDamping:  opts.PagerankDamping,
		pagerankCache:    make(map[string]float64),
		pagerankCacheTTL: 300.0, // 5 minutes
	}
}

// ScoreNode calculates importance score for a node using the paper's formula.
func (is *ImportanceScorer) ScoreNode(node *MemoryNode, allEdges []*MemoryEdge, allNodes []*MemoryNode, currentTime *time.Time) float64 {
	now := time.Now().UTC()
	if currentTime != nil {
		now = *currentTime
	}

	// f1: Temporal recency score (0-1)
	f1Recency := is.recencyScore(node.AccessedAt, now)

	// f2: Access frequency score (0-1)
	f2Frequency := is.frequencyScore(node.AccessCount)

	// f3: PageRank centrality score (0-1)
	f3Pagerank := is.pagerankScore(node.ID, allNodes, allEdges)

	// f4: User/explicit importance (0-1)
	f4User := float64(node.Importance) / 10.0

	// Weighted combination
	score := is.recencyWeight*f1Recency +
		is.frequencyWeight*f2Frequency +
		is.pagerankWeight*f3Pagerank +
		is.userWeight*f4User

	// Scale to 0-10
	return math.Min(10.0, math.Max(0.0, score*10))
}

// pagerankScore calculates PageRank centrality for a node.
func (is *ImportanceScorer) pagerankScore(nodeID string, allNodes []*MemoryNode, allEdges []*MemoryEdge) float64 {
	// Check cache validity
	is.mu.RLock()
	now := time.Now().UTC()
	if len(is.pagerankCache) > 0 &&
		!is.pagerankCacheTime.IsZero() &&
		now.Sub(is.pagerankCacheTime).Seconds() < is.pagerankCacheTTL {
		if score, ok := is.pagerankCache[nodeID]; ok {
			is.mu.RUnlock()
			return score
		}
	}
	is.mu.RUnlock()

	// Need to recompute PageRank
	// Simple implementation of PageRank algorithm
	if len(allNodes) == 0 {
		return 0.0
	}

	// Build graph
	nodeIDs := make(map[string]bool)
	for _, node := range allNodes {
		nodeIDs[node.ID] = true
	}

	// Outgoing edges for each node
	outgoing := make(map[string][]*MemoryEdge)
	for _, edge := range allEdges {
		if nodeIDs[edge.SourceID] && nodeIDs[edge.TargetID] {
			outgoing[edge.SourceID] = append(outgoing[edge.SourceID], edge)
		}
	}

	// Initialize PageRank scores
	n := float64(len(allNodes))
	scores := make(map[string]float64)
	for _, node := range allNodes {
		scores[node.ID] = 1.0 / n
	}

	// Iterate PageRank
	maxIterations := 100
	damping := is.pagerankDamping

	for iter := 0; iter < maxIterations; iter++ {
		newScores := make(map[string]float64)
		for _, node := range allNodes {
			newScores[node.ID] = (1 - damping) / n
		}

		for _, node := range allNodes {
			edges := outgoing[node.ID]
			if len(edges) == 0 {
				// Dangling node - distribute evenly
				contribution := damping * scores[node.ID] / n
				for _, n := range allNodes {
					newScores[n.ID] += contribution
				}
			} else {
				// Distribute based on edge weights
				totalWeight := 0.0
				for _, e := range edges {
					totalWeight += e.Weight * e.Confidence
				}
				if totalWeight == 0 {
					totalWeight = float64(len(edges))
				}

				for _, e := range edges {
					weight := e.Weight * e.Confidence
					if weight == 0 {
						weight = 1.0
					}
					contribution := damping * scores[node.ID] * (weight / totalWeight)
					newScores[e.TargetID] += contribution
				}
			}
		}

		scores = newScores
	}

	// Normalize to [0, 1] range
	maxPR := 0.0
	for _, score := range scores {
		if score > maxPR {
			maxPR = score
		}
	}
	if maxPR > 0 {
		for id := range scores {
			scores[id] /= maxPR
		}
	}

	// Update cache
	is.mu.Lock()
	is.pagerankCache = scores
	is.pagerankCacheTime = now
	is.mu.Unlock()

	return scores[nodeID]
}

// ScoreEdge calculates importance score for an edge.
func (is *ImportanceScorer) ScoreEdge(edge *MemoryEdge, sourceNode, targetNode *MemoryNode, currentTime *time.Time) float64 {
	now := time.Now().UTC()
	if currentTime != nil {
		now = *currentTime
	}

	// Base score from edge properties
	recency := is.recencyScore(edge.AccessedAt, now)
	frequency := is.frequencyScore(edge.AccessCount)

	// Edge strength factors
	weightFactor := math.Min(1.0, edge.Weight/5.0)
	confidenceFactor := edge.Confidence

	// Node importance affects edge importance
	nodeFactor := 0.5
	if sourceNode != nil && targetNode != nil {
		nodeFactor = (float64(sourceNode.Importance)/10.0 + float64(targetNode.Importance)/10.0) / 2
	}

	// Combine factors
	score := 0.25*recency +
		0.2*frequency +
		0.2*weightFactor +
		0.15*confidenceFactor +
		0.2*nodeFactor

	return math.Min(10.0, math.Max(0.0, score*10))
}

// UpdateImportance updates and returns new importance level for a node.
func (is *ImportanceScorer) UpdateImportance(node *MemoryNode, allEdges []*MemoryEdge, allNodes []*MemoryNode) MemoryImportance {
	score := is.ScoreNode(node, allEdges, allNodes, nil)
	return ImportanceFromScore(score)
}

// recencyScore calculates recency score using exponential decay.
func (is *ImportanceScorer) recencyScore(accessedAt time.Time, currentTime time.Time) float64 {
	halfLifeDays := 30.0
	age := currentTime.Sub(accessedAt)
	ageDays := age.Hours() / 24.0

	// Exponential decay
	decay := math.Exp(-0.693 * ageDays / halfLifeDays)
	return decay
}

// frequencyScore calculates frequency score with diminishing returns.
func (is *ImportanceScorer) frequencyScore(accessCount int) float64 {
	if accessCount <= 0 {
		return 0.0
	}

	saturationPoint := 100
	// Logarithmic scaling with saturation
	return math.Min(1.0, math.Log(1.0+float64(accessCount))/math.Log(1.0+float64(saturationPoint)))
}

// ConnectivityScore calculates connectivity score based on edge count.
// This is used as a fallback when PageRank cannot be computed.
func (is *ImportanceScorer) ConnectivityScore(nodeID string, edges []*MemoryEdge) float64 {
	maxConnections := 50
	connectionCount := 0
	for _, e := range edges {
		if e.SourceID == nodeID || e.TargetID == nodeID {
			connectionCount++
		}
	}

	if connectionCount <= 0 {
		return 0.0
	}

	// Logarithmic scaling
	return math.Min(1.0, math.Log(1.0+float64(connectionCount))/math.Log(1.0+float64(maxConnections)))
}

// InvalidateCache clears the PageRank cache.
func (is *ImportanceScorer) InvalidateCache() {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.pagerankCache = make(map[string]float64)
	is.pagerankCacheTime = time.Time{}
	log.Printf("PageRank cache invalidated")
}
