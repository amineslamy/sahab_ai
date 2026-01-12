package graphmem

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
)

// ContextWindow represents a constructed context window for LLM input.
type ContextWindow struct {
	Content     string           // The constructed context text
	TokensUsed  int              // Estimated tokens used
	TokenLimit  int              // Maximum token limit
	Entities    []*MemoryNode    // Included entities
	Edges       []*MemoryEdge    // Included relationships
	Communities []*MemoryCluster // Included communities
	Documents   []*DocumentChunk // Included document chunks
	Query       string           // Original query
	Priority    string           // Priority mode used
	Truncated   bool             // Whether content was truncated
}

// ContextPriority defines how to prioritize content in the context window.
type ContextPriority string

const (
	PriorityBalanced      ContextPriority = "balanced"
	PriorityEntities      ContextPriority = "entities"
	PriorityRelationships ContextPriority = "relationships"
	PriorityCommunities   ContextPriority = "communities"
	PriorityDocuments     ContextPriority = "documents"
)

// ContextEngineConfig holds configuration for the context engine.
type ContextEngineConfig struct {
	TokenLimit     int     // Maximum tokens in context window
	TokensPerChar  float64 // Approximate tokens per character
	ChunkSize      int     // Default chunk size
	ChunkOverlap   int     // Default chunk overlap
}

// ContextEngine provides super context engineering for optimal LLM context construction.
type ContextEngine struct {
	llm            LLMProvider
	embeddings     EmbeddingProvider
	multimodal     *MultiModalProcessor
	chunker        *DocumentChunker
	tokenLimit     int
	tokensPerChar  float64
}

// NewContextEngine creates a new context engine.
func NewContextEngine(llm LLMProvider, embeddings EmbeddingProvider, config *ContextEngineConfig) *ContextEngine {
	if config == nil {
		config = &ContextEngineConfig{
			TokenLimit:    8000,
			TokensPerChar: 0.25,
			ChunkSize:     1000,
			ChunkOverlap:  200,
		}
	}

	return &ContextEngine{
		llm:           llm,
		embeddings:    embeddings,
		multimodal:    NewMultiModalProcessor(llm, &MultiModalConfig{ChunkSize: config.ChunkSize, ChunkOverlap: config.ChunkOverlap}),
		chunker:       &DocumentChunker{ChunkSize: config.ChunkSize, ChunkOverlap: config.ChunkOverlap},
		tokenLimit:    config.TokenLimit,
		tokensPerChar: config.TokensPerChar,
	}
}

// BuildContext builds an optimal context window from the given components.
func (ce *ContextEngine) BuildContext(
	query string,
	entities []*MemoryNode,
	edges []*MemoryEdge,
	communities []*MemoryCluster,
	documents []*DocumentChunk,
	priority ContextPriority,
	includeQuery bool,
) *ContextWindow {
	if priority == "" {
		priority = PriorityBalanced
	}

	// Get priority order
	priorityOrder := ce.getPriorityOrder(priority)

	// Calculate available budget
	budget := ce.tokenLimit
	if includeQuery && query != "" {
		budget -= ce.estimateTokens(fmt.Sprintf("Query: %s\n\n", query))
	}

	// Count items
	counts := map[string]int{
		"entities":      len(entities),
		"relationships": len(edges),
		"communities":   len(communities),
		"documents":     len(documents),
	}

	// Allocate budget
	allocations := ce.allocateBudget(budget, priorityOrder, counts)

	// Build sections
	var sections []struct {
		name    string
		content string
	}
	usedTokens := 0
	truncated := false

	includedEntities := []*MemoryNode{}
	includedEdges := []*MemoryEdge{}
	includedCommunities := []*MemoryCluster{}
	includedDocuments := []*DocumentChunk{}

	for _, component := range priorityOrder {
		switch component {
		case "communities":
			if len(communities) > 0 {
				text, tokens, items := ce.buildCommunitiesSection(communities, allocations["communities"])
				if text != "" {
					sections = append(sections, struct{ name, content string }{"Topic Summaries", text})
					usedTokens += tokens
					includedCommunities = items
					if len(items) < len(communities) {
						truncated = true
					}
				}
			}

		case "entities":
			if len(entities) > 0 {
				text, tokens, items := ce.buildEntitiesSection(entities, allocations["entities"])
				if text != "" {
					sections = append(sections, struct{ name, content string }{"Relevant Entities", text})
					usedTokens += tokens
					includedEntities = items
					if len(items) < len(entities) {
						truncated = true
					}
				}
			}

		case "relationships":
			if len(edges) > 0 {
				text, tokens, items := ce.buildRelationshipsSection(edges, allocations["relationships"])
				if text != "" {
					sections = append(sections, struct{ name, content string }{"Relationships", text})
					usedTokens += tokens
					includedEdges = items
					if len(items) < len(edges) {
						truncated = true
					}
				}
			}

		case "documents":
			if len(documents) > 0 {
				text, tokens, items := ce.buildDocumentsSection(documents, allocations["documents"])
				if text != "" {
					sections = append(sections, struct{ name, content string }{"Supporting Documents", text})
					usedTokens += tokens
					includedDocuments = items
					if len(items) < len(documents) {
						truncated = true
					}
				}
			}
		}
	}

	// Construct final context
	var contextParts []string
	if includeQuery && query != "" {
		contextParts = append(contextParts, fmt.Sprintf("Query: %s", query))
	}

	for _, section := range sections {
		contextParts = append(contextParts, fmt.Sprintf("\n## %s\n%s", section.name, section.content))
	}

	content := strings.Join(contextParts, "\n")
	finalTokens := ce.estimateTokens(content)

	return &ContextWindow{
		Content:     content,
		TokensUsed:  finalTokens,
		TokenLimit:  ce.tokenLimit,
		Entities:    includedEntities,
		Edges:       includedEdges,
		Communities: includedCommunities,
		Documents:   includedDocuments,
		Query:       query,
		Priority:    string(priority),
		Truncated:   truncated,
	}
}

// getPriorityOrder returns the component priority order.
func (ce *ContextEngine) getPriorityOrder(priority ContextPriority) []string {
	switch priority {
	case PriorityCommunities:
		return []string{"communities", "entities", "relationships", "documents"}
	case PriorityEntities:
		return []string{"entities", "relationships", "communities", "documents"}
	case PriorityRelationships:
		return []string{"relationships", "entities", "communities", "documents"}
	case PriorityDocuments:
		return []string{"documents", "entities", "relationships", "communities"}
	default: // balanced
		return []string{"communities", "entities", "relationships", "documents"}
	}
}

// allocateBudget allocates token budget to components.
func (ce *ContextEngine) allocateBudget(budget int, priorityOrder []string, counts map[string]int) map[string]int {
	allocations := make(map[string]int)

	// Priority weights
	weights := map[string]float64{}
	if len(priorityOrder) >= 4 {
		weights[priorityOrder[0]] = 0.35
		weights[priorityOrder[1]] = 0.30
		weights[priorityOrder[2]] = 0.20
		weights[priorityOrder[3]] = 0.15
	}

	for _, component := range priorityOrder {
		if counts[component] == 0 {
			allocations[component] = 0
		} else {
			allocations[component] = int(float64(budget) * weights[component])
		}
	}

	return allocations
}

// estimateTokens estimates token count for text.
func (ce *ContextEngine) estimateTokens(text string) int {
	return int(float64(len(text)) * ce.tokensPerChar)
}

// buildCommunitiesSection builds the communities section.
func (ce *ContextEngine) buildCommunitiesSection(communities []*MemoryCluster, budget int) (string, int, []*MemoryCluster) {
	// Sort by importance
	sorted := make([]*MemoryCluster, len(communities))
	copy(sorted, communities)
	sort.Slice(sorted, func(i, j int) bool {
		return float64(sorted[i].Importance) > float64(sorted[j].Importance)
	})

	var lines []string
	usedTokens := 0
	var included []*MemoryCluster

	for _, community := range sorted {
		line := fmt.Sprintf("• %s", community.Summary)
		lineTokens := ce.estimateTokens(line + "\n")

		if usedTokens+lineTokens > budget {
			break
		}

		lines = append(lines, line)
		usedTokens += lineTokens
		included = append(included, community)
	}

	return strings.Join(lines, "\n"), usedTokens, included
}

// buildEntitiesSection builds the entities section.
func (ce *ContextEngine) buildEntitiesSection(entities []*MemoryNode, budget int) (string, int, []*MemoryNode) {
	// Sort by importance
	sorted := make([]*MemoryNode, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		return float64(sorted[i].Importance) > float64(sorted[j].Importance)
	})

	var lines []string
	usedTokens := 0
	var included []*MemoryNode

	for _, entity := range sorted {
		desc := entity.Description
		if desc == "" {
			desc = entity.Name
		}
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}

		line := fmt.Sprintf("• %s (%s): %s", entity.Name, entity.EntityType, desc)
		lineTokens := ce.estimateTokens(line + "\n")

		if usedTokens+lineTokens > budget {
			break
		}

		lines = append(lines, line)
		usedTokens += lineTokens
		included = append(included, entity)
	}

	return strings.Join(lines, "\n"), usedTokens, included
}

// buildRelationshipsSection builds the relationships section.
func (ce *ContextEngine) buildRelationshipsSection(edges []*MemoryEdge, budget int) (string, int, []*MemoryEdge) {
	// Sort by weight * confidence
	sorted := make([]*MemoryEdge, len(edges))
	copy(sorted, edges)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Weight*sorted[i].Confidence > sorted[j].Weight*sorted[j].Confidence
	})

	var lines []string
	usedTokens := 0
	var included []*MemoryEdge

	for _, rel := range sorted {
		line := fmt.Sprintf("• %s --[%s]--> %s", rel.SourceID, rel.RelationType, rel.TargetID)
		if rel.Description != "" {
			desc := rel.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			line += fmt.Sprintf(": %s", desc)
		}
		lineTokens := ce.estimateTokens(line + "\n")

		if usedTokens+lineTokens > budget {
			break
		}

		lines = append(lines, line)
		usedTokens += lineTokens
		included = append(included, rel)
	}

	return strings.Join(lines, "\n"), usedTokens, included
}

// buildDocumentsSection builds the documents section.
func (ce *ContextEngine) buildDocumentsSection(documents []*DocumentChunk, budget int) (string, int, []*DocumentChunk) {
	var parts []string
	usedTokens := 0
	var included []*DocumentChunk

	for _, doc := range documents {
		content := doc.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		lineTokens := ce.estimateTokens(content + "\n\n")

		if usedTokens+lineTokens > budget {
			break
		}

		parts = append(parts, content)
		usedTokens += lineTokens
		included = append(included, doc)
	}

	return strings.Join(parts, "\n\n"), usedTokens, included
}

// IngestDocument ingests a document for memory.
func (ce *ContextEngine) IngestDocument(ctx context.Context, content string, modality Modality, sourceURI string, metadata map[string]any) (*ProcessedDocument, error) {
	input := &MultiModalInput{
		Text:      content,
		Modality:  modality,
		SourceURI: sourceURI,
		Metadata:  metadata,
	}

	return ce.multimodal.Process(ctx, input)
}

// IngestFile ingests a file for memory.
func (ce *ContextEngine) IngestFile(ctx context.Context, filePath string) (*ProcessedDocument, error) {
	return ce.multimodal.ProcessFile(ctx, filePath)
}

// IngestURL ingests a URL for memory.
func (ce *ContextEngine) IngestURL(ctx context.Context, url string) (*ProcessedDocument, error) {
	return ce.multimodal.ProcessURL(ctx, url)
}

// SummarizeContext summarizes a context window.
func (ce *ContextEngine) SummarizeContext(ctx context.Context, contextWindow *ContextWindow, maxLength int) (string, error) {
	if maxLength <= 0 {
		maxLength = 500
	}

	if ce.llm == nil {
		// Extractive summary
		lines := strings.Split(contextWindow.Content, "\n")
		var summaryLines []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				summaryLines = append(summaryLines, line)
				if len(summaryLines) >= 5 {
					break
				}
			}
		}
		summary := strings.Join(summaryLines, " ")
		if len(summary) > maxLength {
			summary = summary[:maxLength]
		}
		return summary, nil
	}

	prompt := fmt.Sprintf(`Summarize the following context in %d characters or less:

%s

Summary:`, maxLength, contextWindow.Content)

	summary, err := ce.llm.Complete(ctx, prompt)
	if err != nil {
		log.Printf("Context summarization failed: %v", err)
		// Fallback to truncation
		if len(contextWindow.Content) > maxLength {
			return contextWindow.Content[:maxLength], nil
		}
		return contextWindow.Content, nil
	}

	if len(summary) > maxLength {
		summary = summary[:maxLength]
	}

	return summary, nil
}

// ExtractFromURL extracts content from a URL.
func (ce *ContextEngine) ExtractFromURL(url string) (string, error) {
	doc, err := ce.multimodal.ProcessURL(context.Background(), url)
	if err != nil {
		return "", err
	}
	return doc.RawText, nil
}

// GetTokenLimit returns the token limit.
func (ce *ContextEngine) GetTokenLimit() int {
	return ce.tokenLimit
}

// SetTokenLimit sets the token limit.
func (ce *ContextEngine) SetTokenLimit(limit int) {
	ce.tokenLimit = limit
}

