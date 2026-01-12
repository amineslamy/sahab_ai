package graphmem

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// QueryEngine orchestrates memory querying.
type QueryEngine struct {
	llm               LLMProvider
	retriever         *Retriever
	communityDetector *CommunityDetector
	maxWorkers        int
}

// QueryEngineConfig contains configuration for QueryEngine.
type QueryEngineConfig struct {
	MaxWorkers int
}

// NewQueryEngine creates a new query engine.
func NewQueryEngine(llm LLMProvider, retriever *Retriever, communityDetector *CommunityDetector, config *QueryEngineConfig) *QueryEngine {
	if config == nil {
		config = &QueryEngineConfig{}
	}
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 4
	}

	return &QueryEngine{
		llm:               llm,
		retriever:         retriever,
		communityDetector: communityDetector,
		maxWorkers:        config.MaxWorkers,
	}
}

// Query executes a query against memory.
func (qe *QueryEngine) Query(ctx context.Context, query *MemoryQuery, memory *Memory) (*MemoryResponse, error) {
	startTime := time.Now()

	// Retrieve relevant context
	retrievalResult, err := qe.retriever.Retrieve(ctx, query, memory)
	if err != nil {
		return nil, err
	}

	nodes := retrievalResult.Nodes
	edges := retrievalResult.Edges
	clusters := retrievalResult.Clusters
	context := retrievalResult.Context

	if len(nodes) == 0 && len(clusters) == 0 {
		return &MemoryResponse{
			Query:      query.Query,
			Answer:     "No relevant information found in memory.",
			Confidence: 0.0,
			LatencyMS:  float64(time.Since(startTime).Milliseconds()),
		}, nil
	}

	// Prioritize direct entity answers
	var answer string
	var confidence float64

	// First try: Direct answer from retrieved entities
	if len(nodes) > 0 && context != "" {
		entityContext := qe.buildEntityContext(nodes, edges)
		if entityContext != "" {
			answer, confidence = qe.generateDirectAnswer(ctx, query.Query, entityContext)
			if confidence >= 0.5 {
				log.Printf("Using direct entity answer (confidence=%f)", confidence)
			}
		}
	}

	// Second try: Community answers (for broader queries)
	if answer == "" || confidence < 0.5 {
		if len(clusters) > 0 {
			allNodes := memory.GetAllNodes()
			allEdges := memory.GetAllEdges()

			communityAnswers := qe.queryCommunities(ctx, query.Query, clusters, allNodes, allEdges)

			if len(communityAnswers) > 0 {
				commAnswer, commConfidence := qe.aggregateAnswers(ctx, query.Query, communityAnswers)
				if commConfidence > confidence {
					answer = commAnswer
					confidence = commConfidence
				}
			}
		}
	}

	// Fallback: General context answer
	if answer == "" {
		answer, confidence = qe.generateDirectAnswer(ctx, query.Query, context)
	}

	latencyMS := float64(time.Since(startTime).Milliseconds())

	return &MemoryResponse{
		Query:      query.Query,
		Answer:     answer,
		Confidence: confidence,
		Nodes:      nodes,
		Edges:      edges,
		Clusters:   clusters,
		Context:    context,
		Sources:    extractNodeNames(nodes),
		LatencyMS:  latencyMS,
		Timestamp:  time.Now().UTC(),
	}, nil
}

// queryCommunities queries each relevant community for answers.
func (qe *QueryEngine) queryCommunities(ctx context.Context, query string, clusters []*MemoryCluster, nodes []*MemoryNode, edges []*MemoryEdge) []communityAnswer {
	if len(clusters) == 0 {
		return nil
	}

	answers := make([]communityAnswer, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use semaphore for max workers
	sem := make(chan struct{}, qe.maxWorkers)

	for _, cluster := range clusters {
		wg.Add(1)
		go func(c *MemoryCluster) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := qe.querySingleCommunity(ctx, query, c, nodes, edges)
			if result != nil {
				mu.Lock()
				answers = append(answers, *result)
				mu.Unlock()
			}
		}(cluster)
	}

	wg.Wait()
	return answers
}

type communityAnswer struct {
	ClusterID  string
	Answer     string
	Confidence int
}

// querySingleCommunity queries a single community for an answer.
func (qe *QueryEngine) querySingleCommunity(ctx context.Context, query string, cluster *MemoryCluster, nodes []*MemoryNode, edges []*MemoryEdge) *communityAnswer {
	// Get entities in this cluster
	clusterEntitySet := make(map[string]struct{})
	for _, e := range cluster.Entities {
		clusterEntitySet[e] = struct{}{}
	}

	clusterNodes := make([]*MemoryNode, 0)
	clusterNodeIDs := make(map[string]struct{})
	for _, n := range nodes {
		if _, ok := clusterEntitySet[n.Name]; ok {
			clusterNodes = append(clusterNodes, n)
			clusterNodeIDs[n.ID] = struct{}{}
		}
	}

	// Get edges involving cluster nodes
	clusterEdges := make([]*MemoryEdge, 0)
	connectedNodeIDs := make(map[string]struct{})
	for _, e := range edges {
		_, sourceIn := clusterNodeIDs[e.SourceID]
		_, targetIn := clusterNodeIDs[e.TargetID]
		if sourceIn || targetIn {
			clusterEdges = append(clusterEdges, e)
			connectedNodeIDs[e.SourceID] = struct{}{}
			connectedNodeIDs[e.TargetID] = struct{}{}
		}
	}

	// Add connected nodes that aren't already in cluster
	allRelevantNodes := make([]*MemoryNode, len(clusterNodes))
	copy(allRelevantNodes, clusterNodes)
	for _, n := range nodes {
		if _, connected := connectedNodeIDs[n.ID]; connected {
			if _, inCluster := clusterNodeIDs[n.ID]; !inCluster {
				allRelevantNodes = append(allRelevantNodes, n)
			}
		}
	}

	entityContext := qe.formatEntities(allRelevantNodes)
	relContext := qe.formatRelationships(clusterEdges)
	sourceTextContext := qe.extractSourceChunks(allRelevantNodes)

	prompt := fmt.Sprintf(`You are answering questions using knowledge from a memory system.

COMMUNITY SUMMARY:
%s

ORIGINAL SOURCE TEXT (contains full details from ingested documents):
%s

ENTITIES IN THIS COMMUNITY:
%s

RELATIONSHIPS:
%s

QUESTION: %s

CRITICAL INSTRUCTIONS:
1. **PRIORITIZE SOURCE TEXT**: The original source text contains the most complete information.
   Use it to find specific details, numbers, dates, and facts.
2. Consider ALL entities and relationships shown above
3. If multiple facts are relevant to the question, include them ALL
4. Cross-reference the community summary with the source text and entities
5. Provide a comprehensive answer that covers all relevant information
6. Include specific numbers, dates, and details from the source text

Respond in JSON format:
{"answer": "your comprehensive answer with specific details", "confidence": 0-10}`, cluster.Summary, sourceTextContext, entityContext, relContext, query)

	response, err := qe.llm.Complete(ctx, prompt)
	if err != nil {
		log.Printf("LLM query failed: %v", err)
		return nil
	}

	parsed := qe.parseAnswerResponse(response)
	if parsed != nil {
		parsed.ClusterID = cluster.ID
		return parsed
	}

	return nil
}

// extractSourceChunks extracts source chunks from nodes for context.
func (qe *QueryEngine) extractSourceChunks(nodes []*MemoryNode) string {
	allChunks := make([]string, 0)
	seen := make(map[string]struct{})

	for i, node := range nodes {
		if i >= 10 {
			break
		}
		if node.Properties != nil {
			if chunks, ok := node.Properties["source_chunks"].([]any); ok {
				for _, chunk := range chunks {
					if s, ok := chunk.(string); ok && s != "" {
						if _, exists := seen[s]; !exists {
							seen[s] = struct{}{}
							allChunks = append(allChunks, s)
						}
					}
				}
			}
			if chunk, ok := node.Properties["source_chunk"].(string); ok && chunk != "" {
				if _, exists := seen[chunk]; !exists {
					seen[chunk] = struct{}{}
					allChunks = append(allChunks, chunk)
				}
			}
		}
	}

	if len(allChunks) == 0 {
		return "No source text available"
	}

	var sb strings.Builder
	for i, chunk := range allChunks {
		if i >= 5 {
			break
		}
		truncated := chunk
		if len(truncated) > 1500 {
			truncated = truncated[:1500]
		}
		sb.WriteString(fmt.Sprintf("[Source %d]: %s\n\n", i+1, truncated))
	}

	return sb.String()
}

// aggregateAnswers aggregates answers from multiple communities.
func (qe *QueryEngine) aggregateAnswers(ctx context.Context, query string, answers []communityAnswer) (string, float64) {
	if len(answers) == 0 {
		return "No answer found.", 0.0
	}

	// Sort by confidence
	for i := 0; i < len(answers); i++ {
		for j := i + 1; j < len(answers); j++ {
			if answers[j].Confidence > answers[i].Confidence {
				answers[i], answers[j] = answers[j], answers[i]
			}
		}
	}

	// If single high-confidence answer, return it
	if len(answers) == 1 || answers[0].Confidence >= 9 {
		return answers[0].Answer, float64(answers[0].Confidence) / 10.0
	}

	// Aggregate multiple answers
	var sb strings.Builder
	for i := 0; i < len(answers) && i < 5; i++ {
		sb.WriteString(fmt.Sprintf("Answer %d (confidence %d/10): %s\n", i+1, answers[i].Confidence, answers[i].Answer))
	}

	prompt := fmt.Sprintf(`Synthesize these answers from different knowledge communities into a single comprehensive response.

ANSWERS FROM DIFFERENT COMMUNITIES:
%s

QUESTION: %s

INSTRUCTIONS:
1. Combine information from ALL answers, not just the highest confidence one
2. If answers mention different entities or facts, include ALL of them
3. Remove duplicates but preserve all unique information
4. Present a unified, comprehensive answer

Synthesized Answer:`, sb.String(), query)

	finalAnswer, err := qe.llm.Complete(ctx, prompt)
	if err != nil {
		log.Printf("Answer aggregation failed: %v", err)
		return answers[0].Answer, float64(answers[0].Confidence) / 10.0
	}

	// Average confidence of top 3
	var totalConf int
	count := min(3, len(answers))
	for i := 0; i < count; i++ {
		totalConf += answers[i].Confidence
	}
	avgConfidence := float64(totalConf) / float64(count) / 10.0

	return strings.TrimSpace(finalAnswer), avgConfidence
}

// buildEntityContext builds comprehensive context from retrieved entities.
func (qe *QueryEngine) buildEntityContext(nodes []*MemoryNode, edges []*MemoryEdge) string {
	if len(nodes) == 0 {
		return ""
	}

	var sb strings.Builder

	// Section 1: Original source text
	allSourceChunks := make([]string, 0)
	seen := make(map[string]struct{})

	for i, node := range nodes {
		if i >= 15 {
			break
		}
		if node.Properties != nil {
			if chunks, ok := node.Properties["source_chunks"].([]any); ok {
				for _, chunk := range chunks {
					if s, ok := chunk.(string); ok && s != "" {
						if _, exists := seen[s]; !exists {
							seen[s] = struct{}{}
							allSourceChunks = append(allSourceChunks, s)
						}
					}
				}
			}
			if chunk, ok := node.Properties["source_chunk"].(string); ok && chunk != "" {
				if _, exists := seen[chunk]; !exists {
					seen[chunk] = struct{}{}
					allSourceChunks = append(allSourceChunks, chunk)
				}
			}
		}
	}

	if len(allSourceChunks) > 0 {
		sb.WriteString("## ORIGINAL SOURCE TEXT (contains full details)\n")
		for i, chunk := range allSourceChunks {
			if i >= 10 {
				break
			}
			truncated := chunk
			if len(truncated) > 2000 {
				truncated = truncated[:2000]
			}
			sb.WriteString(fmt.Sprintf("\n[Source %d]:\n%s\n", i+1, truncated))
		}
	}

	// Section 2: Extracted entities
	sb.WriteString("\n\n## EXTRACTED ENTITIES\n")
	for i, node := range nodes {
		if i >= 20 {
			break
		}
		aliases := ""
		if len(node.Aliases) > 0 {
			aliasSlice := make([]string, 0, 8)
			for a := range node.Aliases {
				if a != node.Name {
					aliasSlice = append(aliasSlice, a)
					if len(aliasSlice) >= 8 {
						break
					}
				}
			}
			if len(aliasSlice) > 0 {
				aliases = fmt.Sprintf(" [Also known as: %s]", strings.Join(aliasSlice, ", "))
			}
		}

		desc := node.Description
		if desc == "" {
			desc = "No description"
		}

		sb.WriteString(fmt.Sprintf("\n• %s (%s)%s\n", node.Name, node.EntityType, aliases))
		sb.WriteString(fmt.Sprintf("  Description: %s\n", desc))

		if node.Properties != nil {
			if count, ok := node.Properties["occurrence_count"].(int); ok && count > 1 {
				sb.WriteString(fmt.Sprintf("  [Mentioned %d times across documents]\n", count))
			}
		}
	}

	// Section 3: Relationships
	if len(edges) > 0 {
		sb.WriteString("\n\n## RELATIONSHIPS\n")
		nodeIDs := make(map[string]struct{})
		for _, n := range nodes {
			nodeIDs[n.ID] = struct{}{}
		}

		for i, edge := range edges {
			if i >= 40 {
				break
			}

			_, sourceMatch := nodeIDs[edge.SourceID]
			_, targetMatch := nodeIDs[edge.TargetID]

			if sourceMatch || targetMatch {
				temporal := ""
				if edge.ValidFrom != nil {
					fromStr := edge.ValidFrom.Format("2006")
					untilStr := "present"
					if edge.ValidUntil != nil {
						untilStr = edge.ValidUntil.Format("2006")
					}
					temporal = fmt.Sprintf(" [valid: %s → %s]", fromStr, untilStr)
				}

				sb.WriteString(fmt.Sprintf("• %s --[%s]--> %s%s\n", edge.SourceID, edge.RelationType, edge.TargetID, temporal))
				if edge.Description != "" {
					sb.WriteString(fmt.Sprintf("  Detail: %s\n", edge.Description))
				}
			}
		}
	}

	return sb.String()
}

// generateDirectAnswer generates answer directly from context.
func (qe *QueryEngine) generateDirectAnswer(ctx context.Context, query, context string) (string, float64) {
	prompt := fmt.Sprintf(`You are answering questions based on a knowledge graph memory system.

CONTEXT (contains entities, relationships, temporal info, and topic summaries):
%s

QUESTION: %s

CRITICAL INSTRUCTIONS:
1. **ALIASES**: Entities may have multiple names (e.g., "Alexander Chen" = "Dr. Chen" = "The Quantum Pioneer"). 
   If the question uses any alias, find the matching entity by ANY of its names.
   
2. **TEMPORAL VALIDITY**: Relationships have time periods (valid_from → valid_until).
   - If asked "WHO IS current X" → find relationship with valid_until = present/None
   - If asked "WHO WAS X in YEAR" → find relationship where YEAR is between valid_from and valid_until
   - "Present" or no valid_until means the relationship is CURRENT
   
3. **EXHAUSTIVE SEARCH**: Check ALL entities and relationships, not just the first match.

4. **SYNTHESIZE**: Combine information from multiple sources if relevant.

5. **BE SPECIFIC**: If you find temporal info, include it (e.g., "X was CEO from 2015 to 2018").

Answer:`, context, query)

	answer, err := qe.llm.Complete(ctx, prompt)
	if err != nil {
		log.Printf("Direct answer generation failed: %v", err)
		return "Unable to generate answer.", 0.0
	}

	return strings.TrimSpace(answer), 0.7
}

// formatEntities formats entities for prompt.
func (qe *QueryEngine) formatEntities(nodes []*MemoryNode) string {
	var sb strings.Builder
	for i, node := range nodes {
		if i >= 15 {
			break
		}
		desc := node.Description
		if desc == "" {
			desc = "No description"
		}
		if len(desc) > 200 {
			desc = desc[:200]
		}

		aliases := ""
		if len(node.Aliases) > 0 {
			aliasSlice := make([]string, 0, 5)
			for a := range node.Aliases {
				if a != node.Name {
					aliasSlice = append(aliasSlice, a)
					if len(aliasSlice) >= 5 {
						break
					}
				}
			}
			if len(aliasSlice) > 0 {
				aliases = fmt.Sprintf(" [also known as: %s]", strings.Join(aliasSlice, ", "))
			}
		}

		sb.WriteString(fmt.Sprintf("- %s (%s)%s: %s\n", node.Name, node.EntityType, aliases, desc))
	}

	if sb.Len() == 0 {
		return "No entity details available."
	}
	return sb.String()
}

// formatRelationships formats relationships for prompt.
func (qe *QueryEngine) formatRelationships(edges []*MemoryEdge) string {
	var sb strings.Builder
	for i, edge := range edges {
		if i >= 15 {
			break
		}

		temporal := ""
		if edge.ValidFrom != nil {
			fromStr := edge.ValidFrom.Format("2006")
			untilStr := "present"
			if edge.ValidUntil != nil {
				untilStr = edge.ValidUntil.Format("2006")
			}
			temporal = fmt.Sprintf(" [%s → %s]", fromStr, untilStr)
		}

		sb.WriteString(fmt.Sprintf("- %s --[%s]--> %s%s\n", edge.SourceID, edge.RelationType, edge.TargetID, temporal))
	}

	if sb.Len() == 0 {
		return "No relationships available."
	}
	return sb.String()
}

// parseAnswerResponse parses LLM answer response.
func (qe *QueryEngine) parseAnswerResponse(response string) *communityAnswer {
	// Try to parse as JSON
	var parsed struct {
		Answer     string `json:"answer"`
		Confidence int    `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err == nil {
		return &communityAnswer{
			Answer:     parsed.Answer,
			Confidence: parsed.Confidence,
		}
	}

	// Try to extract JSON from response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start != -1 && end != -1 && end > start {
		if err := json.Unmarshal([]byte(response[start:end+1]), &parsed); err == nil {
			return &communityAnswer{
				Answer:     parsed.Answer,
				Confidence: parsed.Confidence,
			}
		}
	}

	// Fallback: use response as answer
	return &communityAnswer{
		Answer:     strings.TrimSpace(response),
		Confidence: 5,
	}
}

// extractNodeNames extracts names from nodes.
func extractNodeNames(nodes []*MemoryNode) []string {
	names := make([]string, 0, len(nodes))
	seen := make(map[string]struct{})

	for _, node := range nodes {
		if _, exists := seen[node.Name]; !exists {
			names = append(names, node.Name)
			seen[node.Name] = struct{}{}
		}
	}

	return names
}
