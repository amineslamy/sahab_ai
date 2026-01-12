package graphmem

import (
	"math"
	"strings"
	"sync"
	"time"
)

// Evolution orchestrates all memory evolution operations.
type Evolution struct {
	llm                    LLMProvider
	embeddings             EmbeddingProvider
	consolidationThreshold float64
	decayEnabled           bool
	decayHalfLifeDays      float64
	minEvolutionInterval   time.Duration

	lastEvolution map[string]time.Time
	mu            sync.RWMutex
}

// EvolutionConfig holds configuration for memory evolution.
type EvolutionConfig struct {
	ConsolidationThreshold float64
	DecayEnabled           bool
	DecayHalfLifeDays      float64
	MinEvolutionInterval   time.Duration
	MaxWorkers             int
}

// DefaultEvolutionConfig returns the default configuration.
func DefaultEvolutionConfig() *EvolutionConfig {
	return &EvolutionConfig{
		ConsolidationThreshold: 0.85,
		DecayEnabled:           true,
		DecayHalfLifeDays:      30.0,
		MinEvolutionInterval:   time.Hour,
		MaxWorkers:             10,
	}
}

// NewEvolution creates a new evolution controller.
func NewEvolution(llmProvider LLMProvider, embeddings EmbeddingProvider, config *EvolutionConfig) *Evolution {
	if config == nil {
		config = DefaultEvolutionConfig()
	}

	return &Evolution{
		llm:                    llmProvider,
		embeddings:             embeddings,
		consolidationThreshold: config.ConsolidationThreshold,
		decayEnabled:           config.DecayEnabled,
		decayHalfLifeDays:      config.DecayHalfLifeDays,
		minEvolutionInterval:   config.MinEvolutionInterval,
		lastEvolution:          make(map[string]time.Time),
	}
}

// Evolve evolves the memory.
func (e *Evolution) Evolve(memory *Memory, evolutionTypes []EvolutionType, force bool) []*EvolutionEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if we should evolve
	if !force {
		if lastEvolved, ok := e.lastEvolution[memory.ID]; ok {
			if time.Since(lastEvolved) < e.minEvolutionInterval {
				return nil
			}
		}
	}

	var allEvents []*EvolutionEvent

	// Determine which evolution types to run
	if len(evolutionTypes) == 0 {
		evolutionTypes = []EvolutionType{
			EvolutionConsolidation,
			EvolutionDecay,
			EvolutionReinforcement,
		}
	}

	for _, evolutionType := range evolutionTypes {
		switch evolutionType {
		case EvolutionConsolidation:
			events := e.runConsolidation(memory)
			allEvents = append(allEvents, events...)

		case EvolutionDecay:
			if e.decayEnabled {
				events := e.runDecay(memory)
				allEvents = append(allEvents, events...)
			}

		case EvolutionReinforcement:
			events := e.updateImportanceScores(memory)
			allEvents = append(allEvents, events...)
		}
	}

	// Record evolution time
	e.lastEvolution[memory.ID] = time.Now()

	// Update memory version
	memory.Version++
	memory.UpdatedAt = time.Now().UTC()

	return allEvents
}

// runConsolidation runs memory consolidation.
func (e *Evolution) runConsolidation(memory *Memory) []*EvolutionEvent {
	var events []*EvolutionEvent

	// Group nodes by entity type
	nodesByType := make(map[string][]*MemoryNode)
	for _, node := range memory.GetAllNodes() {
		if node.State == StateActive {
			nodesByType[node.EntityType] = append(nodesByType[node.EntityType], node)
		}
	}

	// Find similar nodes within each type
	for _, nodes := range nodesByType {
		if len(nodes) < 2 {
			continue
		}

		for i := 0; i < len(nodes); i++ {
			for j := i + 1; j < len(nodes); j++ {
				nodeA := nodes[i]
				nodeB := nodes[j]

				if nodeA.State != StateActive || nodeB.State != StateActive {
					continue
				}

				similarity := e.calculateSimilarity(nodeA, nodeB)
				if similarity >= e.consolidationThreshold {
					event := e.mergeNodes(memory, nodeA, nodeB, similarity)
					if event != nil {
						events = append(events, event)
					}
				}
			}
		}
	}

	return events
}

func (e *Evolution) calculateSimilarity(nodeA, nodeB *MemoryNode) float64 {
	if len(nodeA.Embedding) > 0 && len(nodeB.Embedding) > 0 {
		return CosineSimilarity(nodeA.Embedding, nodeB.Embedding)
	}
	return nameSimilarity(nodeA.Name, nodeB.Name)
}

func (e *Evolution) mergeNodes(memory *Memory, nodeA, nodeB *MemoryNode, similarity float64) *EvolutionEvent {
	var keep, remove *MemoryNode
	if nodeA.AccessCount > nodeB.AccessCount ||
		(nodeA.AccessCount == nodeB.AccessCount && nodeA.Importance > nodeB.Importance) {
		keep = nodeA
		remove = nodeB
	} else {
		keep = nodeB
		remove = nodeA
	}

	// Merge aliases
	for alias := range remove.Aliases {
		keep.AddAlias(alias)
	}
	keep.AddAlias(remove.Name)

	// Update description
	if remove.Description != "" && len(remove.Description) > len(keep.Description) {
		keep.Description = remove.Description
	}

	// Update access count
	keep.AccessCount += remove.AccessCount

	// Mark removed node
	remove.State = StateDeleted

	// Update edges
	for _, edge := range memory.GetAllEdges() {
		if edge.SourceID == remove.ID {
			edge.SourceID = keep.ID
		}
		if edge.TargetID == remove.ID {
			edge.TargetID = keep.ID
		}
	}

	event := NewEvolutionEvent(EvolutionConsolidation, memory.ID)
	event.AffectedNodes = []string{keep.ID, remove.ID}
	event.Reason = "Merged similar entities"
	event.BeforeState = map[string]any{
		"kept_node":    keep.Name,
		"removed_node": remove.Name,
		"similarity":   similarity,
	}
	event.AfterState = map[string]any{
		"merged_into": keep.Name,
		"aliases":     keep.GetAliases(),
	}

	return event
}

func (e *Evolution) runDecay(memory *Memory) []*EvolutionEvent {
	var events []*EvolutionEvent
	now := time.Now()

	halfLifeSeconds := e.decayHalfLifeDays * 24 * 60 * 60
	lambda := math.Log(2) / halfLifeSeconds

	for _, node := range memory.GetAllNodes() {
		if node.State != StateActive {
			continue
		}

		if node.Importance == ImportanceCritical {
			continue
		}

		timeSinceAccess := now.Sub(node.AccessedAt).Seconds()
		decayFactor := math.Exp(-lambda * timeSinceAccess)

		currentScore := float64(node.Importance)
		newScore := currentScore * decayFactor
		newImportance := ImportanceFromScore(newScore)

		if newImportance < node.Importance && node.Importance-newImportance >= 2 {
			oldImportance := node.Importance
			node.Importance = newImportance

			if newImportance == ImportanceEphemeral {
				node.State = StateDecaying
			}

			event := NewEvolutionEvent(EvolutionDecay, memory.ID)
			event.AffectedNodes = []string{node.ID}
			event.Reason = "Memory decay due to lack of access"
			event.BeforeState = map[string]any{
				"importance":        oldImportance.String(),
				"days_since_access": timeSinceAccess / (24 * 60 * 60),
			}
			event.AfterState = map[string]any{
				"importance": newImportance.String(),
			}

			events = append(events, event)
		}
	}

	return events
}

func (e *Evolution) updateImportanceScores(memory *Memory) []*EvolutionEvent {
	var events []*EvolutionEvent

	allNodes := memory.GetAllNodes()
	allEdges := memory.GetAllEdges()

	pageRankScores := e.calculatePageRank(allNodes, allEdges)

	for _, node := range allNodes {
		if node.State != StateActive {
			continue
		}

		pageRankScore := pageRankScores[node.ID]
		accessScore := float64(node.AccessCount) / 100.0
		recencyScore := e.calculateRecencyScore(node.AccessedAt)

		combinedScore := 0.3*recencyScore + 0.3*accessScore + 0.2*pageRankScore*10 + 0.2*float64(node.Importance)
		newImportance := ImportanceFromScore(combinedScore)

		if absDiff(int(newImportance), int(node.Importance)) >= 2 {
			oldImportance := node.Importance
			node.Importance = newImportance

			event := NewEvolutionEvent(EvolutionReinforcement, memory.ID)
			event.AffectedNodes = []string{node.ID}
			event.Reason = "Importance score updated"
			event.BeforeState = map[string]any{"importance": oldImportance.String()}
			event.AfterState = map[string]any{"importance": newImportance.String()}

			events = append(events, event)
		}
	}

	return events
}

func (e *Evolution) calculatePageRank(nodes []*MemoryNode, edges []*MemoryEdge) map[string]float64 {
	const dampingFactor = 0.85
	const iterations = 20

	scores := make(map[string]float64)
	outDegree := make(map[string]int)

	n := float64(len(nodes))
	if n == 0 {
		return scores
	}

	for _, node := range nodes {
		scores[node.ID] = 1.0 / n
	}

	for _, edge := range edges {
		outDegree[edge.SourceID]++
	}

	for i := 0; i < iterations; i++ {
		newScores := make(map[string]float64)

		for _, node := range nodes {
			incomingSum := 0.0
			for _, edge := range edges {
				if edge.TargetID == node.ID {
					if out := outDegree[edge.SourceID]; out > 0 {
						incomingSum += scores[edge.SourceID] / float64(out)
					}
				}
			}

			newScores[node.ID] = (1-dampingFactor)/n + dampingFactor*incomingSum
		}

		scores = newScores
	}

	return scores
}

func (e *Evolution) calculateRecencyScore(lastAccess time.Time) float64 {
	daysSinceAccess := time.Since(lastAccess).Hours() / 24
	switch {
	case daysSinceAccess < 1:
		return 10.0
	case daysSinceAccess < 7:
		return 8.0
	case daysSinceAccess < 30:
		return 5.0
	case daysSinceAccess < 90:
		return 3.0
	default:
		return 1.0
	}
}

// GetStats returns evolution statistics.
func (e *Evolution) GetStats(memory *Memory) map[string]any {
	e.mu.RLock()
	lastEvolved := e.lastEvolution[memory.ID]
	e.mu.RUnlock()

	activeNodes := 0
	archivedNodes := 0
	deletedNodes := 0
	for _, node := range memory.GetAllNodes() {
		switch node.State {
		case StateActive:
			activeNodes++
		case StateArchived:
			archivedNodes++
		case StateDeleted:
			deletedNodes++
		}
	}

	importanceDist := make(map[string]int)
	for _, node := range memory.GetAllNodes() {
		importanceDist[node.Importance.String()]++
	}

	return map[string]any{
		"memory_id":               memory.ID,
		"version":                 memory.Version,
		"last_evolved":            lastEvolved.Format(time.RFC3339),
		"total_nodes":             memory.NodeCount(),
		"total_edges":             memory.EdgeCount(),
		"total_clusters":          memory.ClusterCount(),
		"active_nodes":            activeNodes,
		"archived_nodes":          archivedNodes,
		"deleted_nodes":           deletedNodes,
		"importance_distribution": importanceDist,
	}
}

// Helper functions

func nameSimilarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0.0
	}
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	if a == b {
		return 1.0
	}

	lcs := longestCommonSubsequence(a, b)
	return 2.0 * float64(lcs) / float64(len(a)+len(b))
}

func longestCommonSubsequence(a, b string) int {
	m := len(a)
	n := len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	return dp[m][n]
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}
