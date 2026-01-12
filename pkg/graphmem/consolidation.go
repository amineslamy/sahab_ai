package graphmem

import (
	"context"
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

// EntityConsolidationPrompt is the prompt template for LLM-based entity consolidation.
const EntityConsolidationPrompt = `You are an expert at identifying when different names refer to the SAME entity.

## ENTITIES (same type: %s)
%s

## TASK
Group entities that refer to the SAME real-world entity. Consider:
- "Dr. Chen", "Alexander Chen", "A. Chen", "Professor Chen" → SAME person
- "The Quantum Pioneer" could be a nickname for someone
- "Tesla, Inc.", "Tesla", "Tesla Motors" → SAME company
- Different descriptions may still be the same entity

## OUTPUT FORMAT
Output groups of entity IDs that should be merged. Each line is one group.
Format: ID1, ID2, ID3, ...

Example:
0, 2, 5
1, 3
4

If entity is unique (no matches), list it alone:
6

List ALL entities, either grouped or alone.

## GROUPS (one per line, entity IDs separated by commas):
`

// MemoryConsolidation consolidates similar memories into stronger, unified representations.
type MemoryConsolidation struct {
	embeddings            EmbeddingProvider
	llm                   LLMProvider
	similarityThreshold   float64
	minOccurrencesToMerge int
	synthesisEnabled      bool
}

// ConsolidationOptions contains options for MemoryConsolidation.
type ConsolidationOptions struct {
	SimilarityThreshold   float64
	MinOccurrencesToMerge int
	SynthesisEnabled      bool
}

// NewMemoryConsolidation creates a new MemoryConsolidation instance.
func NewMemoryConsolidation(embeddings EmbeddingProvider, llm LLMProvider, opts *ConsolidationOptions) *MemoryConsolidation {
	if opts == nil {
		opts = &ConsolidationOptions{
			SimilarityThreshold:   0.85,
			MinOccurrencesToMerge: 2,
			SynthesisEnabled:      true,
		}
	}

	return &MemoryConsolidation{
		embeddings:            embeddings,
		llm:                   llm,
		similarityThreshold:   opts.SimilarityThreshold,
		minOccurrencesToMerge: opts.MinOccurrencesToMerge,
		synthesisEnabled:      opts.SynthesisEnabled,
	}
}

// Consolidate consolidates memories.
func (mc *MemoryConsolidation) Consolidate(memory *Memory) []*EvolutionEvent {
	events := make([]*EvolutionEvent, 0)

	// 1. Find and merge similar entities
	mergeEvents := mc.consolidateEntities(memory)
	events = append(events, mergeEvents...)

	// 2. Strengthen frequently co-occurring edges
	reinforceEvents := mc.reinforceEdges(memory)
	events = append(events, reinforceEvents...)

	// 3. Synthesize new knowledge (optional)
	if mc.synthesisEnabled {
		synthesisEvents := mc.synthesizeKnowledge(memory)
		events = append(events, synthesisEvents...)
	}

	log.Printf("Consolidation complete: %d events", len(events))
	return events
}

// consolidateEntities finds and merges similar entities.
func (mc *MemoryConsolidation) consolidateEntities(memory *Memory) []*EvolutionEvent {
	events := make([]*EvolutionEvent, 0)

	nodes := memory.GetAllNodes()
	if len(nodes) < 2 {
		return events
	}

	// Group nodes by entity type for efficiency
	typeGroups := make(map[string][]*MemoryNode)
	for _, node := range nodes {
		key := strings.ToLower(node.EntityType)
		if key == "" {
			key = "unknown"
		}
		typeGroups[key] = append(typeGroups[key], node)
	}

	// Find merge candidates within each type using LLM - CONCURRENT
	var allMergeGroups []map[string]bool
	var mu sync.Mutex
	var wg sync.WaitGroup

	for entityType, typeNodes := range typeGroups {
		if len(typeNodes) < 2 {
			continue
		}

		wg.Add(1)
		go func(et string, tn []*MemoryNode) {
			defer wg.Done()

			var groups []map[string]bool
			// Use LLM to identify duplicates (if available)
			if mc.llm != nil && len(tn) <= 50 {
				groups = mc.llmFindDuplicates(tn, et)
			} else {
				// Fallback to embedding-based matching for large sets
				groups = mc.embeddingFindDuplicates(tn)
			}

			mu.Lock()
			allMergeGroups = append(allMergeGroups, groups...)
			mu.Unlock()
		}(entityType, typeNodes)
	}

	wg.Wait()

	// Perform merges
	processedNodes := make(map[string]bool)
	for _, group := range allMergeGroups {
		if len(group) < 2 {
			continue
		}

		// Skip if any node already processed
		skip := false
		for nid := range group {
			if processedNodes[nid] {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		groupNodes := make([]*MemoryNode, 0)
		for nid := range group {
			if node := memory.GetNode(nid); node != nil {
				groupNodes = append(groupNodes, node)
			}
		}
		if len(groupNodes) < 2 {
			continue
		}

		mergedNode, affectedEdges := mc.mergeNodes(groupNodes, memory)
		for nid := range group {
			processedNodes[nid] = true
		}

		// Record event
		affectedNodeIDs := make([]string, 0, len(group))
		for nid := range group {
			affectedNodeIDs = append(affectedNodeIDs, nid)
		}

		events = append(events, &EvolutionEvent{
			EvolutionType: EvolutionConsolidation,
			MemoryID:      memory.ID,
			AffectedNodes: affectedNodeIDs,
			AffectedEdges: affectedEdges,
			BeforeState:   map[string]any{"node_count": len(group)},
			AfterState:    map[string]any{"merged_node": mergedNode.ID},
			Reason:        fmt.Sprintf("Merged %d similar entities into '%s'", len(group), mergedNode.Name),
			Timestamp:     time.Now().UTC(),
		})

		nodeNames := make([]string, len(groupNodes))
		for i, n := range groupNodes {
			nodeNames[i] = n.Name
		}
		log.Printf("🔗 Merged entities: %v → '%s'", nodeNames, mergedNode.Name)
	}

	return events
}

// llmFindDuplicates uses LLM to find duplicate entities.
func (mc *MemoryConsolidation) llmFindDuplicates(nodes []*MemoryNode, entityType string) []map[string]bool {
	if mc.llm == nil {
		return nil
	}

	// Build entity list for prompt
	entitiesList := make([]string, len(nodes))
	nodeIDMap := make(map[int]string)
	for i, node := range nodes {
		nodeIDMap[i] = node.ID
		aliasesStr := "none"
		if len(node.Aliases) > 0 {
			aliases := make([]string, 0, len(node.Aliases))
			for a := range node.Aliases {
				aliases = append(aliases, a)
			}
			aliasesStr = strings.Join(aliases, ", ")
		}
		desc := node.Description
		if len(desc) > 100 {
			desc = desc[:100]
		}
		entitiesList[i] = fmt.Sprintf("[%d] %s (aliases: %s) - %s", i, node.Name, aliasesStr, desc)
	}

	prompt := fmt.Sprintf(EntityConsolidationPrompt, entityType, strings.Join(entitiesList, "\n"))

	response, err := mc.llm.Complete(context.Background(), prompt)
	if err != nil {
		log.Printf("LLM consolidation failed: %v, falling back to embeddings", err)
		return mc.embeddingFindDuplicates(nodes)
	}

	// Parse response into groups
	mergeGroups := make([]map[string]bool, 0)
	for _, line := range strings.Split(strings.TrimSpace(response), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse comma-separated IDs
		parts := strings.Split(line, ",")
		ids := make([]int, 0)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			var id int
			if _, err := fmt.Sscanf(p, "%d", &id); err == nil {
				ids = append(ids, id)
			}
		}

		if len(ids) >= 2 {
			// Convert indices to node IDs
			group := make(map[string]bool)
			for _, i := range ids {
				if nid, ok := nodeIDMap[i]; ok {
					group[nid] = true
				}
			}
			if len(group) >= 2 {
				mergeGroups = append(mergeGroups, group)
			}
		}
	}

	log.Printf("LLM found %d entity groups to merge in %s", len(mergeGroups), entityType)
	return mergeGroups
}

// embeddingFindDuplicates finds duplicates using embeddings.
func (mc *MemoryConsolidation) embeddingFindDuplicates(nodes []*MemoryNode) []map[string]bool {
	mergeGroups := make([]map[string]bool, 0)
	processed := make(map[string]bool)

	// Get embeddings
	embeddingsMap := make(map[string][]float32)
	for _, node := range nodes {
		text := node.Description
		if text == "" {
			text = node.Name
		}
		if mc.embeddings != nil {
			emb, err := mc.embeddings.Embed(context.Background(), text)
			if err == nil && len(emb) > 0 {
				embeddingsMap[node.ID] = emb
			}
		}
	}

	// Find similar pairs
	for i, nodeA := range nodes {
		if processed[nodeA.ID] {
			continue
		}

		similarGroup := map[string]bool{nodeA.ID: true}

		for j := i + 1; j < len(nodes); j++ {
			nodeB := nodes[j]
			if processed[nodeB.ID] {
				continue
			}

			if mc.areSimilar(nodeA, nodeB, embeddingsMap) {
				similarGroup[nodeB.ID] = true
			}
		}

		if len(similarGroup) >= 2 {
			mergeGroups = append(mergeGroups, similarGroup)
			for id := range similarGroup {
				processed[id] = true
			}
		}
	}

	return mergeGroups
}

// areSimilar checks if two nodes are similar enough to merge.
func (mc *MemoryConsolidation) areSimilar(nodeA, nodeB *MemoryNode, embeddingsMap map[string][]float32) bool {
	// Name similarity
	nameA := nodeA.CanonicalName
	if nameA == "" {
		nameA = nodeA.Name
	}
	nameB := nodeB.CanonicalName
	if nameB == "" {
		nameB = nodeB.Name
	}

	// CRITICAL: Check for EXACT name match (case-insensitive)
	if strings.EqualFold(strings.TrimSpace(nameA), strings.TrimSpace(nameB)) {
		return true
	}

	// Check if either name matches the other's canonical name
	if strings.EqualFold(strings.TrimSpace(nodeA.Name), strings.TrimSpace(nameB)) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(nodeB.Name), strings.TrimSpace(nameA)) {
		return true
	}

	// Check for alias overlap
	for a := range nodeA.Aliases {
		if nodeB.Aliases[a] {
			return true
		}
	}

	// Check if any alias in A matches any name in B
	aliasesALower := make(map[string]bool)
	for a := range nodeA.Aliases {
		aliasesALower[strings.ToLower(strings.TrimSpace(a))] = true
	}
	aliasesBLower := make(map[string]bool)
	for b := range nodeB.Aliases {
		aliasesBLower[strings.ToLower(strings.TrimSpace(b))] = true
	}
	for a := range aliasesALower {
		if aliasesBLower[a] {
			return true
		}
	}

	// Check if name of one is in aliases of other
	if aliasesBLower[strings.ToLower(strings.TrimSpace(nameA))] {
		return true
	}
	if aliasesALower[strings.ToLower(strings.TrimSpace(nameB))] {
		return true
	}

	// Check name containment
	lowerA := strings.ToLower(nameA)
	lowerB := strings.ToLower(nameB)
	if strings.Contains(lowerA, lowerB) || strings.Contains(lowerB, lowerA) {
		return true
	}

	// Check shared last name with embedding similarity
	if mc.shareLastName(nameA, nameB) {
		embA := embeddingsMap[nodeA.ID]
		embB := embeddingsMap[nodeB.ID]
		if len(embA) > 0 && len(embB) > 0 {
			sim := cosineSimilarity32(embA, embB)
			if sim >= 0.65 {
				return true
			}
		}
	}

	// Token overlap check
	tokensA := extractTokens(nameA)
	tokensB := extractTokens(nameB)
	stopwords := map[string]bool{"the": true, "dr": true, "mr": true, "ms": true, "mrs": true, "prof": true, "professor": true, "inc": true, "corp": true, "llc": true}
	for sw := range stopwords {
		delete(tokensA, sw)
		delete(tokensB, sw)
	}

	if len(tokensA) > 0 && len(tokensB) > 0 {
		overlap := 0
		for t := range tokensA {
			if tokensB[t] {
				overlap++
			}
		}
		minTokens := len(tokensA)
		if len(tokensB) < minTokens {
			minTokens = len(tokensB)
		}
		if minTokens > 0 && float64(overlap)/float64(minTokens) >= 0.5 {
			embA := embeddingsMap[nodeA.ID]
			embB := embeddingsMap[nodeB.ID]
			if len(embA) > 0 && len(embB) > 0 {
				sim := cosineSimilarity32(embA, embB)
				if sim >= 0.60 {
					return true
				}
			}
		}
	}

	// Check embedding similarity
	embA := embeddingsMap[nodeA.ID]
	embB := embeddingsMap[nodeB.ID]
	if len(embA) > 0 && len(embB) > 0 {
		similarity := cosineSimilarity32(embA, embB)
		if similarity >= 0.75 {
			return true
		}
	}

	return false
}

// extractTokens extracts word tokens from a name.
func extractTokens(name string) map[string]bool {
	re := regexp.MustCompile(`\b[a-z]{2,}\b`)
	matches := re.FindAllString(strings.ToLower(name), -1)
	tokens := make(map[string]bool)
	for _, m := range matches {
		tokens[m] = true
	}
	return tokens
}

// shareLastName checks if two names share a last name.
func (mc *MemoryConsolidation) shareLastName(nameA, nameB string) bool {
	getLastName := func(name string) string {
		re := regexp.MustCompile(`\b[A-Za-z]{2,}\b`)
		words := re.FindAllString(name, -1)
		if len(words) == 0 {
			return ""
		}
		skip := map[string]bool{"dr": true, "mr": true, "ms": true, "mrs": true, "prof": true, "professor": true, "jr": true, "sr": true, "phd": true, "md": true, "the": true}
		for i := len(words) - 1; i >= 0; i-- {
			if !skip[strings.ToLower(words[i])] {
				return strings.ToLower(words[i])
			}
		}
		if len(words) > 0 {
			return strings.ToLower(words[len(words)-1])
		}
		return ""
	}

	lastA := getLastName(nameA)
	lastB := getLastName(nameB)

	return lastA != "" && lastB != "" && lastA == lastB && len(lastA) >= 3
}

// mergeNodes merges multiple nodes into one.
func (mc *MemoryConsolidation) mergeNodes(nodes []*MemoryNode, memory *Memory) (*MemoryNode, []string) {
	// Choose the best name (longest/most complete)
	bestNode := nodes[0]
	for _, n := range nodes[1:] {
		if len(n.Name) > len(bestNode.Name) || (len(n.Name) == len(bestNode.Name) && n.AccessCount > bestNode.AccessCount) {
			bestNode = n
		}
	}

	// Collect all aliases
	allAliases := make(map[string]bool)
	allDescriptions := make(map[string]bool)
	totalAccess := 0
	highestImportance := ImportanceEphemeral

	for _, node := range nodes {
		for a := range node.Aliases {
			allAliases[a] = true
		}
		allAliases[node.Name] = true
		if node.Description != "" {
			allDescriptions[node.Description] = true
		}
		totalAccess += node.AccessCount
		if node.Importance > highestImportance {
			highestImportance = node.Importance
		}
	}

	// Choose best description
	bestDesc := bestNode.Description
	for d := range allDescriptions {
		if len(d) > len(bestDesc) {
			bestDesc = d
		}
	}

	// Create merged node
	merged := &MemoryNode{
		ID:            bestNode.ID,
		Name:          bestNode.Name,
		EntityType:    bestNode.EntityType,
		Description:   bestDesc,
		CanonicalName: bestNode.CanonicalName,
		Aliases:       allAliases,
		Embedding:     bestNode.Embedding,
		Properties: map[string]any{
			"merged_from": getNodeIDs(nodes),
			"merge_count": len(nodes),
		},
		Importance:  highestImportance,
		AccessCount: totalAccess,
		UserID:      bestNode.UserID,
		MemoryID:    memory.ID,
		CreatedAt:   bestNode.CreatedAt,
		UpdatedAt:   time.Now().UTC(),
		AccessedAt:  time.Now().UTC(),
	}

	// Copy original properties
	for k, v := range bestNode.Properties {
		if _, exists := merged.Properties[k]; !exists {
			merged.Properties[k] = v
		}
	}

	// Update edges to point to merged node
	affectedEdges := make([]string, 0)
	for _, node := range nodes {
		if node.ID == merged.ID {
			continue
		}

		for edgeID, edge := range memory.Edges {
			updated := false
			newSource := edge.SourceID
			newTarget := edge.TargetID

			if edge.SourceID == node.ID {
				newSource = merged.ID
				updated = true
			}
			if edge.TargetID == node.ID {
				newTarget = merged.ID
				updated = true
			}

			if updated {
				// Create updated edge
				updatedEdge := edge.Clone()
				updatedEdge.SourceID = newSource
				updatedEdge.TargetID = newTarget
				memory.Edges[edgeID] = updatedEdge
				affectedEdges = append(affectedEdges, edgeID)
			}
		}

		// Remove merged node
		delete(memory.Nodes, node.ID)
	}

	// Add/update merged node
	memory.Nodes[merged.ID] = merged

	return merged, affectedEdges
}

func getNodeIDs(nodes []*MemoryNode) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
}

// reinforceEdges strengthens edges that appear multiple times.
func (mc *MemoryConsolidation) reinforceEdges(memory *Memory) []*EvolutionEvent {
	events := make([]*EvolutionEvent, 0)

	// Group edges by (source, target, relation)
	type edgeKey struct {
		Source, Target, Relation string
	}
	edgeGroups := make(map[edgeKey][]*MemoryEdge)

	for _, edge := range memory.Edges {
		key := edgeKey{edge.SourceID, edge.TargetID, edge.RelationType}
		edgeGroups[key] = append(edgeGroups[key], edge)
	}

	// Merge duplicate edges
	for _, edges := range edgeGroups {
		if len(edges) < 2 {
			continue
		}

		// Keep strongest edge, reinforce it
		strongest := edges[0]
		for _, e := range edges[1:] {
			if e.Weight > strongest.Weight || (e.Weight == strongest.Weight && e.Confidence > strongest.Confidence) {
				strongest = e
			}
		}

		// Combine weights and confidence
		var totalWeight float64
		var totalConfidence float64
		for _, e := range edges {
			totalWeight += e.Weight
			totalConfidence += e.Confidence
		}
		avgConfidence := totalConfidence / float64(len(edges))

		// Create reinforced edge
		reinforced := strongest.Clone()
		reinforced.Weight = math.Min(10.0, totalWeight)
		reinforced.Confidence = math.Min(1.0, avgConfidence*1.1)

		// Remove duplicates, keep reinforced
		for _, edge := range edges {
			if edge.ID != reinforced.ID {
				delete(memory.Edges, edge.ID)
			}
		}
		memory.Edges[reinforced.ID] = reinforced

		events = append(events, &EvolutionEvent{
			EvolutionType: EvolutionReinforcement,
			MemoryID:      memory.ID,
			AffectedEdges: []string{reinforced.ID},
			BeforeState:   map[string]any{"edge_count": len(edges)},
			AfterState:    map[string]any{"weight": reinforced.Weight, "confidence": reinforced.Confidence},
			Reason:        fmt.Sprintf("Reinforced edge from %d occurrences", len(edges)),
			Timestamp:     time.Now().UTC(),
		})
	}

	return events
}

// synthesizeKnowledge creates new knowledge by inferring from patterns.
func (mc *MemoryConsolidation) synthesizeKnowledge(memory *Memory) []*EvolutionEvent {
	events := make([]*EvolutionEvent, 0)

	// Build adjacency for outgoing edges
	outgoing := make(map[string][]*MemoryEdge)
	for _, edge := range memory.Edges {
		outgoing[edge.SourceID] = append(outgoing[edge.SourceID], edge)
	}

	newEdges := make([]*MemoryEdge, 0)
	existingPairs := make(map[string]bool)
	for _, e := range memory.Edges {
		existingPairs[e.SourceID+":"+e.TargetID] = true
	}

	for nodeAID := range memory.Nodes {
		edgesA := outgoing[nodeAID]

		for _, edgeAB := range edgesA {
			nodeBID := edgeAB.TargetID
			edgesB := outgoing[nodeBID]

			for _, edgeBC := range edgesB {
				nodeCID := edgeBC.TargetID

				// Skip if A == C
				if nodeAID == nodeCID {
					continue
				}

				// Skip if A->C already exists
				if existingPairs[nodeAID+":"+nodeCID] {
					continue
				}

				// Only infer if both edges are strong
				if edgeAB.Confidence < 0.7 || edgeBC.Confidence < 0.7 {
					continue
				}

				// Create inferred edge
				minWeight := edgeAB.Weight
				if edgeBC.Weight < minWeight {
					minWeight = edgeBC.Weight
				}

				newEdge := &MemoryEdge{
					ID:           GenerateID(),
					SourceID:     nodeAID,
					TargetID:     nodeCID,
					RelationType: "inferred_connection",
					Description:  fmt.Sprintf("Inferred from %s and %s", edgeAB.RelationType, edgeBC.RelationType),
					Weight:       minWeight * 0.5,
					Confidence:   edgeAB.Confidence * edgeBC.Confidence * 0.8,
					Properties: map[string]any{
						"inferred":     true,
						"via_node":     nodeBID,
						"source_edges": []string{edgeAB.ID, edgeBC.ID},
					},
					MemoryID:  memory.ID,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}

				newEdges = append(newEdges, newEdge)
				existingPairs[nodeAID+":"+nodeCID] = true
			}
		}
	}

	// Add synthesized edges (limit to prevent explosion)
	maxEdges := 10
	if len(newEdges) > maxEdges {
		newEdges = newEdges[:maxEdges]
	}

	for _, edge := range newEdges {
		memory.AddEdge(edge)

		events = append(events, &EvolutionEvent{
			EvolutionType: EvolutionSynthesis,
			MemoryID:      memory.ID,
			AffectedEdges: []string{edge.ID},
			AfterState:    map[string]any{"edge_id": edge.ID, "relation": edge.RelationType},
			Reason:        "Synthesized transitive relationship",
			Timestamp:     time.Now().UTC(),
		})
	}

	return events
}
