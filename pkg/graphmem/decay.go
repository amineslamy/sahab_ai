package graphmem

import (
	"log"
	"math"
	"time"
)

// MemoryDecay handles memory decay (forgetting) over time.
type MemoryDecay struct {
	llm                 LLMProvider
	halfLifeDays        float64
	minImportanceToKeep MemoryImportance
	archiveThreshold    float64
	deleteThreshold     float64
	importanceScorer    *ImportanceScorer
}

// DecayOptions contains options for MemoryDecay.
type DecayOptions struct {
	HalfLifeDays        float64
	MinImportanceToKeep MemoryImportance
	ArchiveThreshold    float64
	DeleteThreshold     float64
}

// NewMemoryDecay creates a new MemoryDecay instance.
func NewMemoryDecay(llm LLMProvider, opts *DecayOptions) *MemoryDecay {
	if opts == nil {
		opts = &DecayOptions{
			HalfLifeDays:        30.0,
			MinImportanceToKeep: ImportanceVeryLow,
			ArchiveThreshold:    0.2,
			DeleteThreshold:     0.05,
		}
	}

	return &MemoryDecay{
		llm:                 llm,
		halfLifeDays:        opts.HalfLifeDays,
		minImportanceToKeep: opts.MinImportanceToKeep,
		archiveThreshold:    opts.ArchiveThreshold,
		deleteThreshold:     opts.DeleteThreshold,
		importanceScorer:    NewImportanceScorer(nil),
	}
}

// ApplyDecay applies decay to all memory elements.
func (md *MemoryDecay) ApplyDecay(memory *Memory, currentTime *time.Time) []*EvolutionEvent {
	now := time.Now().UTC()
	if currentTime != nil {
		now = *currentTime
	}

	events := make([]*EvolutionEvent, 0)

	// Decay nodes
	nodeEvents := md.decayNodes(memory, now)
	events = append(events, nodeEvents...)

	// Decay edges
	edgeEvents := md.decayEdges(memory, now)
	events = append(events, edgeEvents...)

	log.Printf("Applied decay: %d elements affected", len(events))
	return events
}

// decayNodes decays nodes based on age and importance.
func (md *MemoryDecay) decayNodes(memory *Memory, currentTime time.Time) []*EvolutionEvent {
	events := make([]*EvolutionEvent, 0)
	nodesToArchive := make([]string, 0)
	nodesToDelete := make([]string, 0)

	allEdges := memory.GetAllEdges()
	allNodes := memory.GetAllNodes()

	for nodeID, node := range memory.Nodes {
		// Skip if already archived or deleted
		if node.State == StateArchived || node.State == StateDeleted {
			continue
		}

		// Critical memories never decay
		if node.Importance == ImportanceCritical {
			continue
		}

		// Calculate decay factor
		strength := md.calculateStrength(node, currentTime)

		// Determine action
		if strength <= md.deleteThreshold {
			nodesToDelete = append(nodesToDelete, nodeID)
		} else if strength <= md.archiveThreshold {
			nodesToArchive = append(nodesToArchive, nodeID)
		} else {
			// Update importance based on decay
			newImportance := md.importanceScorer.UpdateImportance(node, allEdges, allNodes)

			if newImportance != node.Importance {
				updatedNode := node.Clone()
				updatedNode.Importance = newImportance
				memory.Nodes[nodeID] = updatedNode
			}
		}
	}

	// Archive nodes
	for _, nodeID := range nodesToArchive {
		node := memory.Nodes[nodeID]
		if node.Importance >= md.minImportanceToKeep {
			continue // Don't archive if importance is high enough
		}

		beforeState := map[string]any{"state": string(node.State)}
		archivedNode := node.Clone()
		archivedNode.State = StateArchived
		memory.Nodes[nodeID] = archivedNode

		events = append(events, &EvolutionEvent{
			EvolutionType: EvolutionDecay,
			MemoryID:      memory.ID,
			AffectedNodes: []string{nodeID},
			BeforeState:   beforeState,
			AfterState:    map[string]any{"state": "ARCHIVED"},
			Reason:        "Memory strength below archive threshold",
			Timestamp:     time.Now().UTC(),
		})
	}

	// Delete nodes (soft delete)
	for _, nodeID := range nodesToDelete {
		node := memory.Nodes[nodeID]
		if node.Importance >= md.minImportanceToKeep {
			continue
		}

		beforeState := map[string]any{"state": string(node.State)}
		deletedNode := node.Clone()
		deletedNode.State = StateDeleted
		memory.Nodes[nodeID] = deletedNode

		events = append(events, &EvolutionEvent{
			EvolutionType: EvolutionPruning,
			MemoryID:      memory.ID,
			AffectedNodes: []string{nodeID},
			BeforeState:   beforeState,
			AfterState:    map[string]any{"state": "DELETED"},
			Reason:        "Memory strength below delete threshold",
			Timestamp:     time.Now().UTC(),
		})
	}

	return events
}

// decayEdges decays edges based on age and strength.
func (md *MemoryDecay) decayEdges(memory *Memory, currentTime time.Time) []*EvolutionEvent {
	events := make([]*EvolutionEvent, 0)
	edgesToWeaken := make([]struct {
		ID       string
		Strength float64
	}, 0)
	edgesToDelete := make([]string, 0)

	for edgeID, edge := range memory.Edges {
		if edge.State == StateArchived || edge.State == StateDeleted {
			continue
		}

		if edge.Importance == ImportanceCritical {
			continue
		}

		// Check if source or target is deleted
		sourceDeleted := false
		targetDeleted := false
		if source, ok := memory.Nodes[edge.SourceID]; ok {
			sourceDeleted = source.State == StateDeleted
		}
		if target, ok := memory.Nodes[edge.TargetID]; ok {
			targetDeleted = target.State == StateDeleted
		}

		if sourceDeleted || targetDeleted {
			edgesToDelete = append(edgesToDelete, edgeID)
			continue
		}

		// Calculate decay
		strength := md.calculateEdgeStrength(edge, currentTime)

		if strength <= md.deleteThreshold {
			edgesToDelete = append(edgesToDelete, edgeID)
		} else if strength < 0.5 {
			edgesToWeaken = append(edgesToWeaken, struct {
				ID       string
				Strength float64
			}{edgeID, strength})
		}
	}

	// Weaken edges
	for _, item := range edgesToWeaken {
		edge := memory.Edges[item.ID]
		oldWeight := edge.Weight
		newWeight := oldWeight * item.Strength

		weakenedEdge := edge.Clone()
		weakenedEdge.Weight = newWeight
		memory.Edges[item.ID] = weakenedEdge

		events = append(events, &EvolutionEvent{
			EvolutionType: EvolutionDecay,
			MemoryID:      memory.ID,
			AffectedEdges: []string{item.ID},
			BeforeState:   map[string]any{"weight": oldWeight},
			AfterState:    map[string]any{"weight": newWeight},
			Reason:        "Edge decay over time",
			Timestamp:     time.Now().UTC(),
		})
	}

	// Delete edges
	for _, edgeID := range edgesToDelete {
		edge := memory.Edges[edgeID]
		beforeState := map[string]any{"state": string(edge.State)}

		deletedEdge := edge.Clone()
		deletedEdge.State = StateDeleted
		memory.Edges[edgeID] = deletedEdge

		events = append(events, &EvolutionEvent{
			EvolutionType: EvolutionPruning,
			MemoryID:      memory.ID,
			AffectedEdges: []string{edgeID},
			BeforeState:   beforeState,
			AfterState:    map[string]any{"state": "DELETED"},
			Reason:        "Edge strength below threshold or connected to deleted node",
			Timestamp:     time.Now().UTC(),
		})
	}

	return events
}

// calculateStrength calculates current strength of a memory node.
func (md *MemoryDecay) calculateStrength(node *MemoryNode, currentTime time.Time) float64 {
	// Time since last access
	age := currentTime.Sub(node.AccessedAt)
	ageDays := age.Hours() / 24.0

	// Importance modifier (higher importance = slower decay)
	importanceFactor := 0.5 + (float64(node.Importance) / 20.0) // 0.5 to 1.0

	// Access count modifier (more access = slower decay)
	accessFactor := math.Min(1.0, 0.5+math.Log(1.0+float64(node.AccessCount))/10.0)

	// Effective half-life
	effectiveHalfLife := md.halfLifeDays * importanceFactor * accessFactor

	// Exponential decay
	strength := math.Exp(-0.693 * ageDays / effectiveHalfLife)

	return strength
}

// calculateEdgeStrength calculates current strength of an edge.
func (md *MemoryDecay) calculateEdgeStrength(edge *MemoryEdge, currentTime time.Time) float64 {
	age := currentTime.Sub(edge.AccessedAt)
	ageDays := age.Hours() / 24.0

	// Edge weight and confidence affect decay
	weightFactor := math.Min(1.0, edge.Weight/5.0)
	confidenceFactor := edge.Confidence

	// Effective half-life
	effectiveHalfLife := md.halfLifeDays * weightFactor * confidenceFactor
	if effectiveHalfLife < 1.0 {
		effectiveHalfLife = 1.0
	}

	// Exponential decay
	strength := math.Exp(-0.693 * ageDays / effectiveHalfLife)

	return strength
}
