package graphmem

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
)

// CommunityDetector detects communities in knowledge graphs.
type CommunityDetector struct {
	llm            LLMProvider
	maxClusterSize int
	minClusterSize int
	algorithm      string // "greedy_modularity", "louvain", "label_propagation"
}

// CommunityDetectorConfig contains configuration for CommunityDetector.
type CommunityDetectorConfig struct {
	MaxClusterSize int
	MinClusterSize int
	Algorithm      string
}

// NewCommunityDetector creates a new community detector.
func NewCommunityDetector(llm LLMProvider, config *CommunityDetectorConfig) *CommunityDetector {
	if config == nil {
		config = &CommunityDetectorConfig{}
	}
	if config.MaxClusterSize <= 0 {
		config.MaxClusterSize = 100
	}
	if config.MinClusterSize <= 0 {
		config.MinClusterSize = 2
	}
	if config.Algorithm == "" {
		config.Algorithm = "greedy_modularity"
	}

	return &CommunityDetector{
		llm:            llm,
		maxClusterSize: config.MaxClusterSize,
		minClusterSize: config.MinClusterSize,
		algorithm:      config.Algorithm,
	}
}

// Detect detects communities in the graph.
func (cd *CommunityDetector) Detect(nodes []*MemoryNode, edges []*MemoryEdge, memoryID string) []*MemoryCluster {
	if len(nodes) == 0 {
		return nil
	}

	// Build graph structure
	nodeByID := make(map[string]*MemoryNode)
	for _, n := range nodes {
		nodeByID[n.ID] = n
	}

	// Build adjacency list
	adjacency := make(map[string]map[string]float64)
	for _, node := range nodes {
		adjacency[node.ID] = make(map[string]float64)
	}

	for _, edge := range edges {
		if _, ok := nodeByID[edge.SourceID]; !ok {
			continue
		}
		if _, ok := nodeByID[edge.TargetID]; !ok {
			continue
		}
		adjacency[edge.SourceID][edge.TargetID] = edge.Weight
		adjacency[edge.TargetID][edge.SourceID] = edge.Weight // Undirected
	}

	// Detect communities using the selected algorithm
	communities := cd.detectCommunities(adjacency, nodes)

	if len(communities) == 0 {
		return nil
	}

	// Build clusters with summaries
	clusters := make([]*MemoryCluster, 0)

	for i, community := range communities {
		if len(community) < cd.minClusterSize {
			continue
		}

		// Get nodes in this community
		communityNodes := make([]*MemoryNode, 0)
		for nodeID := range community {
			if node, ok := nodeByID[nodeID]; ok {
				communityNodes = append(communityNodes, node)
			}
		}

		if len(communityNodes) == 0 {
			continue
		}

		// Get edges within community
		communityEdges := make([]*MemoryEdge, 0)
		for _, e := range edges {
			_, sourceIn := community[e.SourceID]
			_, targetIn := community[e.TargetID]
			if sourceIn && targetIn {
				communityEdges = append(communityEdges, e)
			}
		}

		// Generate summary
		summary := cd.generateSummary(communityNodes, communityEdges)

		// Calculate metrics
		coherence := cd.calculateCoherence(adjacency, community)
		density := cd.calculateDensity(adjacency, community)

		// Determine importance from nodes
		importance := ImportanceMedium
		for _, n := range communityNodes {
			if n.Importance > importance {
				importance = n.Importance
			}
		}

		// Collect entity names
		entities := make([]string, len(communityNodes))
		for j, n := range communityNodes {
			entities[j] = n.Name
		}

		// Collect edge IDs
		edgeIDs := make([]string, len(communityEdges))
		for j, e := range communityEdges {
			edgeIDs[j] = e.ID
		}

		cluster := &MemoryCluster{
			ID:             fmt.Sprintf("%d", i),
			Summary:        summary,
			Entities:       entities,
			Edges:          edgeIDs,
			Importance:     importance,
			CoherenceScore: coherence,
			Density:        density,
			MemoryID:       memoryID,
			Metadata: map[string]any{
				"algorithm":  cd.algorithm,
				"node_count": len(communityNodes),
				"edge_count": len(communityEdges),
			},
		}
		clusters = append(clusters, cluster)
	}

	log.Printf("Detected %d communities", len(clusters))
	return clusters
}

// detectCommunities runs the community detection algorithm.
func (cd *CommunityDetector) detectCommunities(adjacency map[string]map[string]float64, nodes []*MemoryNode) []map[string]bool {
	if len(nodes) == 0 {
		return nil
	}

	if len(nodes) == 1 {
		// Single node = single community
		community := make(map[string]bool)
		community[nodes[0].ID] = true
		return []map[string]bool{community}
	}

	// Check if graph has edges
	hasEdges := false
	for _, neighbors := range adjacency {
		if len(neighbors) > 0 {
			hasEdges = true
			break
		}
	}

	if !hasEdges {
		// No edges - each node is its own community
		communities := make([]map[string]bool, len(nodes))
		for i, node := range nodes {
			community := make(map[string]bool)
			community[node.ID] = true
			communities[i] = community
		}
		return communities
	}

	// Use greedy modularity algorithm (simplified implementation)
	switch cd.algorithm {
	case "greedy_modularity", "louvain":
		return cd.greedyModularity(adjacency, nodes)
	case "label_propagation":
		return cd.labelPropagation(adjacency, nodes)
	default:
		return cd.greedyModularity(adjacency, nodes)
	}
}

// greedyModularity implements a simplified greedy modularity algorithm.
func (cd *CommunityDetector) greedyModularity(adjacency map[string]map[string]float64, nodes []*MemoryNode) []map[string]bool {
	// Initialize each node in its own community
	nodeToComm := make(map[string]int)
	commToNodes := make(map[int]map[string]bool)

	for i, node := range nodes {
		nodeToComm[node.ID] = i
		commToNodes[i] = map[string]bool{node.ID: true}
	}

	// Calculate total edge weight
	var totalWeight float64
	for _, neighbors := range adjacency {
		for _, w := range neighbors {
			totalWeight += w
		}
	}
	totalWeight /= 2 // Undirected

	if totalWeight == 0 {
		totalWeight = 1
	}

	// Iteratively merge communities that maximize modularity gain
	improved := true
	maxIterations := 100
	iteration := 0

	for improved && iteration < maxIterations {
		improved = false
		iteration++

		for _, node := range nodes {
			currentComm := nodeToComm[node.ID]

			// Find the best community to move to
			bestComm := currentComm
			bestGain := 0.0

			// Check moving to neighbor communities
			neighborComms := make(map[int]struct{})
			for neighborID := range adjacency[node.ID] {
				neighborComms[nodeToComm[neighborID]] = struct{}{}
			}

			for targetComm := range neighborComms {
				if targetComm == currentComm {
					continue
				}

				gain := cd.modularityGain(node.ID, currentComm, targetComm, nodeToComm, adjacency, totalWeight)
				if gain > bestGain {
					bestGain = gain
					bestComm = targetComm
				}
			}

			// Move to best community if there's a gain
			if bestComm != currentComm && bestGain > 0 {
				// Remove from current community
				delete(commToNodes[currentComm], node.ID)
				if len(commToNodes[currentComm]) == 0 {
					delete(commToNodes, currentComm)
				}

				// Add to new community
				if commToNodes[bestComm] == nil {
					commToNodes[bestComm] = make(map[string]bool)
				}
				commToNodes[bestComm][node.ID] = true
				nodeToComm[node.ID] = bestComm

				improved = true
			}
		}
	}

	// Convert to result format
	communities := make([]map[string]bool, 0, len(commToNodes))
	for _, nodes := range commToNodes {
		if len(nodes) > 0 {
			community := make(map[string]bool)
			for nodeID := range nodes {
				community[nodeID] = true
			}
			communities = append(communities, community)
		}
	}

	// Split large communities
	result := make([]map[string]bool, 0)
	for _, community := range communities {
		if len(community) > cd.maxClusterSize {
			// Split into smaller chunks
			nodeIDs := make([]string, 0, len(community))
			for id := range community {
				nodeIDs = append(nodeIDs, id)
			}

			for i := 0; i < len(nodeIDs); i += cd.maxClusterSize {
				end := i + cd.maxClusterSize
				if end > len(nodeIDs) {
					end = len(nodeIDs)
				}
				subCommunity := make(map[string]bool)
				for _, id := range nodeIDs[i:end] {
					subCommunity[id] = true
				}
				result = append(result, subCommunity)
			}
		} else {
			result = append(result, community)
		}
	}

	return result
}

// modularityGain calculates the modularity gain of moving a node to a new community.
func (cd *CommunityDetector) modularityGain(nodeID string, fromComm, toComm int, nodeToComm map[string]int, adjacency map[string]map[string]float64, totalWeight float64) float64 {
	// Simplified modularity gain calculation
	var inDegree, outDegree float64

	for neighborID, weight := range adjacency[nodeID] {
		if nodeToComm[neighborID] == toComm {
			inDegree += weight
		}
		outDegree += weight
	}

	// Calculate community internal weight
	var toCommInternal float64
	for nID, comm := range nodeToComm {
		if comm == toComm {
			for _, weight := range adjacency[nID] {
				toCommInternal += weight
			}
		}
	}
	toCommInternal /= 2

	gain := (inDegree - (outDegree * toCommInternal / totalWeight)) / totalWeight
	return gain
}

// labelPropagation implements a simple label propagation algorithm.
func (cd *CommunityDetector) labelPropagation(adjacency map[string]map[string]float64, nodes []*MemoryNode) []map[string]bool {
	// Initialize labels
	labels := make(map[string]string)
	for _, node := range nodes {
		labels[node.ID] = node.ID
	}

	// Iterate until convergence
	maxIterations := 100
	for iter := 0; iter < maxIterations; iter++ {
		changed := false

		// Shuffle nodes for random order
		shuffled := make([]*MemoryNode, len(nodes))
		copy(shuffled, nodes)

		for _, node := range shuffled {
			// Count neighbor labels
			labelCounts := make(map[string]float64)
			for neighborID, weight := range adjacency[node.ID] {
				labelCounts[labels[neighborID]] += weight
			}

			if len(labelCounts) == 0 {
				continue
			}

			// Find most common label
			var maxLabel string
			var maxCount float64
			for label, count := range labelCounts {
				if count > maxCount {
					maxCount = count
					maxLabel = label
				}
			}

			if maxLabel != "" && maxLabel != labels[node.ID] {
				labels[node.ID] = maxLabel
				changed = true
			}
		}

		if !changed {
			break
		}
	}

	// Group nodes by label
	labelToNodes := make(map[string]map[string]bool)
	for nodeID, label := range labels {
		if labelToNodes[label] == nil {
			labelToNodes[label] = make(map[string]bool)
		}
		labelToNodes[label][nodeID] = true
	}

	// Convert to result
	communities := make([]map[string]bool, 0, len(labelToNodes))
	for _, nodes := range labelToNodes {
		communities = append(communities, nodes)
	}

	return communities
}

// generateSummary generates an exhaustive summary for a community.
func (cd *CommunityDetector) generateSummary(nodes []*MemoryNode, edges []*MemoryEdge) string {
	if len(nodes) == 0 {
		return "Empty community"
	}

	// Collect entity information
	var entityDescriptions []string
	for i, node := range nodes {
		if i >= 15 {
			break
		}
		aliases := ""
		if len(node.Aliases) > 0 {
			aliasSlice := make([]string, 0, 3)
			for a := range node.Aliases {
				if len(aliasSlice) >= 3 {
					break
				}
				aliasSlice = append(aliasSlice, a)
			}
			if len(aliasSlice) > 0 {
				aliases = fmt.Sprintf(" [aliases: %s]", strings.Join(aliasSlice, ", "))
			}
		}
		desc := node.Description
		if len(desc) > 100 {
			desc = desc[:100]
		}
		if desc == "" {
			desc = "no description"
		}
		entityDescriptions = append(entityDescriptions, fmt.Sprintf("• %s (%s)%s: %s", node.Name, node.EntityType, aliases, desc))
	}

	// Collect relationship information with temporal data
	var relationships []string
	var temporalInfo []string
	nodeByID := make(map[string]*MemoryNode)
	for _, n := range nodes {
		nodeByID[n.ID] = n
	}

	for i, edge := range edges {
		if i >= 20 {
			break
		}
		sourceNode := nodeByID[edge.SourceID]
		targetNode := nodeByID[edge.TargetID]

		if sourceNode == nil || targetNode == nil {
			continue
		}

		relStr := fmt.Sprintf("• %s → %s → %s", sourceNode.Name, edge.RelationType, targetNode.Name)

		if edge.ValidFrom != nil {
			fromStr := edge.ValidFrom.Format("2006")
			untilStr := "present"
			if edge.ValidUntil != nil {
				untilStr = edge.ValidUntil.Format("2006")
			}
			relStr += fmt.Sprintf(" [valid: %s to %s]", fromStr, untilStr)
			temporalInfo = append(temporalInfo, fmt.Sprintf("%s-%s: %s %s %s", fromStr, untilStr, sourceNode.Name, edge.RelationType, targetNode.Name))
		}

		relationships = append(relationships, relStr)
	}

	if len(relationships) == 0 && len(entityDescriptions) == 0 {
		entityNames := make([]string, 0)
		for i, n := range nodes {
			if i >= 10 {
				break
			}
			entityNames = append(entityNames, n.Name)
		}
		return fmt.Sprintf("Group of related entities: %s", strings.Join(entityNames, ", "))
	}

	entityText := strings.Join(entityDescriptions, "\n")
	if entityText == "" {
		entityText = "No entity details"
	}
	relationshipText := strings.Join(relationships, "\n")
	if relationshipText == "" {
		relationshipText = "No relationships"
	}
	temporalText := strings.Join(temporalInfo, "\n")
	if temporalText == "" {
		temporalText = "No temporal information"
	}

	prompt := fmt.Sprintf(`You are creating an EXHAUSTIVE knowledge base summary for a community of related entities.

ENTITIES IN THIS COMMUNITY:
%s

RELATIONSHIPS:
%s

TEMPORAL TIMELINE:
%s

Create a comprehensive summary with the following sections:

## ABOUT THIS COMMUNITY
- Main theme/topic (one sentence)
- Key entities and their roles
- How they are connected

## KEY FACTS (directly answerable questions)
- List 3-5 specific facts that can be answered from this community
- Example: "Alexander Chen is the founder of Quantum AI Labs"

## TEMPORAL SCOPE
- When is this information valid?
- Are there relationships that have ended vs ongoing?
- If asking about "current" X, which facts are still active?

## NOT ABOUT (for noise filtering)
- What topics/entities are SIMILAR but NOT in this community?
- What questions would this community NOT be able to answer?
- Help distinguish this from other communities

## CONFIDENCE NOTES
- What is well-established vs uncertain?
- Are there conflicting facts?

Write the summary:`, entityText, relationshipText, temporalText)

	if cd.llm != nil {
		ctx := context.Background()
		response, err := cd.llm.Complete(ctx, prompt)
		if err == nil {
			return strings.TrimSpace(response)
		}
		log.Printf("Exhaustive summary generation failed: %v", err)
	}

	// Fallback
	entityNames := make([]string, 0)
	for i, n := range nodes {
		if i >= 5 {
			break
		}
		entityNames = append(entityNames, n.Name)
	}
	return fmt.Sprintf("Community about: %s. Contains %d entities and %d relationships.", strings.Join(entityNames, ", "), len(nodes), len(edges))
}

// calculateCoherence calculates how well-connected the community is internally.
func (cd *CommunityDetector) calculateCoherence(adjacency map[string]map[string]float64, community map[string]bool) float64 {
	if len(community) <= 1 {
		return 1.0
	}

	// Count actual edges within community
	var actualEdges float64
	for nodeID := range community {
		for neighborID := range adjacency[nodeID] {
			if _, ok := community[neighborID]; ok {
				actualEdges++
			}
		}
	}
	actualEdges /= 2 // Undirected

	// Max possible edges
	n := float64(len(community))
	maxEdges := n * (n - 1) / 2

	if maxEdges == 0 {
		return 1.0
	}

	coherence := actualEdges / maxEdges
	if coherence > 1.0 {
		coherence = 1.0
	}
	return coherence
}

// calculateDensity calculates edge density within the community.
func (cd *CommunityDetector) calculateDensity(adjacency map[string]map[string]float64, community map[string]bool) float64 {
	if len(community) <= 1 {
		return 1.0
	}

	// Same as coherence for simple graphs
	return cd.calculateCoherence(adjacency, community)
}

// GetTopCommunities returns the top N communities by importance.
func (cd *CommunityDetector) GetTopCommunities(clusters []*MemoryCluster, n int) []*MemoryCluster {
	if len(clusters) == 0 {
		return nil
	}

	// Sort by importance
	sorted := make([]*MemoryCluster, len(clusters))
	copy(sorted, clusters)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Importance > sorted[j].Importance
	})

	if n >= len(sorted) {
		return sorted
	}

	return sorted[:n]
}
