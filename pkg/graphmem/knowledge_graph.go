package graphmem

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// extractionPrompt is the LLM prompt for knowledge graph extraction.
const extractionPrompt = `You are an EXHAUSTIVE knowledge graph extractor. Extract EVERYTHING from the text.

## EXTRACT
1. ALL ENTITIES - Every person, organization, product, location, concept, event, date, number
2. ALL RELATIONSHIPS - Every connection between entities
3. ALL NUMBERS - Percentages, amounts, counts, measurements
4. ALL DATES - Years, dates, periods, durations

## FORMAT
Entity: ("entity"$$$$<NAME>$$$$<TYPE>$$$$<DESCRIPTION>$$$$<ALIASES>)
Relationship: ("relationship"$$$$<SOURCE>$$$$<TARGET>$$$$<RELATION>$$$$<DESC>$$$$<VALID_FROM>$$$$<VALID_UNTIL>)

## EXAMPLE
Text: "In Q3 2024, Nvidia reported $35.1B revenue. CEO Jensen Huang announced the new B200 GPU."

Output:
("entity"$$$$Nvidia$$$$Company$$$$Semiconductor company, $35.1B Q3 2024 revenue$$$$NVDA, Nvidia Corporation)
("entity"$$$$Jensen Huang$$$$Person$$$$CEO of Nvidia$$$$Jensen, J. Huang)
("entity"$$$$B200$$$$Product$$$$New GPU from Nvidia$$$$B200 GPU, Blackwell B200)
("entity"$$$$Q3 2024$$$$Date$$$$Third quarter 2024$$$$3Q24)
("entity"$$$$$35.1B$$$$Amount$$$$Nvidia Q3 2024 revenue$$$$35.1 billion)
("relationship"$$$$Jensen Huang$$$$Nvidia$$$$is CEO of$$$$Chief Executive Officer$$$$none$$$$present)
("relationship"$$$$Nvidia$$$$$35.1B$$$$reported revenue$$$$Q3 2024 revenue$$$$2024-07$$$$2024-09)
("relationship"$$$$Nvidia$$$$B200$$$$develops$$$$New GPU product$$$$none$$$$present)

## TEXT TO EXTRACT
%s

## OUTPUT (extract everything)
`

// KnowledgeGraphConfig holds configuration for knowledge graph extraction.
type KnowledgeGraphConfig struct {
	ChunkSize           int
	ChunkOverlap        int
	MaxTripletsPerChunk int
	MaxWorkers          int
}

// KnowledgeGraph extracts entities and relationships from text.
type KnowledgeGraph struct {
	llm                 LLMProvider
	embeddings          EmbeddingProvider
	resolver            *EntityResolver
	chunkSize           int
	chunkOverlap        int
	maxTripletsPerChunk int
}

// NewKnowledgeGraph creates a new KnowledgeGraph.
func NewKnowledgeGraph(llmProvider LLMProvider, embProvider EmbeddingProvider, config *KnowledgeGraphConfig) *KnowledgeGraph {
	if config == nil {
		config = &KnowledgeGraphConfig{
			ChunkSize:           2048,
			ChunkOverlap:        200,
			MaxTripletsPerChunk: 40,
			MaxWorkers:          8,
		}
	}

	return &KnowledgeGraph{
		llm:                 llmProvider,
		embeddings:          embProvider,
		resolver:            NewEntityResolver(embProvider),
		chunkSize:           config.ChunkSize,
		chunkOverlap:        config.ChunkOverlap,
		maxTripletsPerChunk: config.MaxTripletsPerChunk,
	}
}

// Extract extracts entities and relationships from content.
func (kg *KnowledgeGraph) Extract(ctx context.Context, content, memoryID, userID string) ([]*MemoryNode, []*MemoryEdge, error) {
	if content == "" {
		return nil, nil, nil
	}

	chunks := kg.splitIntoChunks(content)
	log.Printf("Split content into %d chunks", len(chunks))

	var allNodes []*MemoryNode
	var allEdges []*MemoryEdge

	for i, chunk := range chunks {
		nodes, edges, err := kg.extractFromChunk(ctx, chunk, memoryID, userID)
		if err != nil {
			log.Printf("Warning: Chunk %d extraction failed: %v", i, err)
			continue
		}
		allNodes = append(allNodes, nodes...)
		allEdges = append(allEdges, edges...)
	}

	// Resolve entity duplicates
	resolvedNodes, err := kg.resolver.Resolve(allNodes, memoryID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("entity resolution failed: %w", err)
	}

	// Update edge source/target to use resolved node IDs
	resolvedEdges := kg.resolveEdgeEntities(allEdges, resolvedNodes)

	log.Printf("Extracted %d unique entities, %d relationships", len(resolvedNodes), len(resolvedEdges))
	return resolvedNodes, resolvedEdges, nil
}

func (kg *KnowledgeGraph) splitIntoChunks(content string) []string {
	if len(content) <= kg.chunkSize {
		return []string{content}
	}

	var chunks []string
	sentences := regexp.MustCompile(`[.!?]\s+`).Split(content, -1)

	var currentChunk []string
	currentLen := 0

	for _, sentence := range sentences {
		sentLen := len(sentence)
		if currentLen+sentLen > kg.chunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, strings.Join(currentChunk, " "))

			// Keep overlap
			overlapChunk := []string{}
			overlapLen := 0
			for i := len(currentChunk) - 1; i >= 0 && overlapLen < kg.chunkOverlap; i-- {
				overlapChunk = append([]string{currentChunk[i]}, overlapChunk...)
				overlapLen += len(currentChunk[i])
			}

			currentChunk = overlapChunk
			currentLen = overlapLen
		}

		currentChunk = append(currentChunk, sentence)
		currentLen += sentLen
	}

	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, " "))
	}

	if len(chunks) == 0 {
		return []string{content}
	}

	return chunks
}

func (kg *KnowledgeGraph) extractFromChunk(ctx context.Context, chunk, memoryID, userID string) ([]*MemoryNode, []*MemoryEdge, error) {
	prompt := fmt.Sprintf(extractionPrompt, chunk)

	response, err := kg.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, nil, fmt.Errorf("LLM extraction failed: %w", err)
	}

	entities, relationships := kg.parseResponse(response)

	// Create nodes
	nodes := make([]*MemoryNode, 0, len(entities))
	for _, e := range entities {
		name := e[0]
		entityType := e[1]
		description := e[2]
		aliasesStr := e[3]

		node := NewMemoryNode(name, entityType, userID, memoryID)
		node.Description = description

		// Parse aliases
		for _, alias := range strings.Split(aliasesStr, ",") {
			alias = strings.TrimSpace(alias)
			if alias != "" {
				node.AddAlias(alias)
			}
		}

		// Generate embedding if provider is available
		if kg.embeddings != nil {
			textToEmbed := name
			if description != "" {
				textToEmbed = fmt.Sprintf("%s: %s", name, description)
			}
			emb, err := kg.embeddings.Embed(ctx, textToEmbed)
			if err != nil {
				log.Printf("Warning: Failed to embed %s: %v", name, err)
			} else {
				node.Embedding = emb
			}
		}

		node.Properties["source_chunk"] = chunk[:min(len(chunk), 500)]
		nodes = append(nodes, node)
	}

	// Create edges
	edges := make([]*MemoryEdge, 0, len(relationships))
	for _, r := range relationships {
		source := r[0]
		target := r[1]
		relation := r[2]
		description := r[3]
		validFromStr := r[4]
		validUntilStr := r[5]

		edge := NewMemoryEdge(source, target, relation, memoryID)
		edge.Description = description
		edge.ValidFrom = parseDate(validFromStr)
		edge.ValidUntil = parseDate(validUntilStr)

		edges = append(edges, edge)
	}

	return nodes, edges, nil
}

func (kg *KnowledgeGraph) parseResponse(response string) ([][]string, [][]string) {
	var entities [][]string
	var relationships [][]string

	entityRe := regexp.MustCompile(`\("entity"\$\$\$\$([^$]+)\$\$\$\$([^$]+)\$\$\$\$([^$]*)\$\$\$\$([^)]*)\)`)
	relRe := regexp.MustCompile(`\("relationship"\$\$\$\$([^$]+)\$\$\$\$([^$]+)\$\$\$\$([^$]+)\$\$\$\$([^$]*)\$\$\$\$([^$]*)\$\$\$\$([^)]*)\)`)

	for _, match := range entityRe.FindAllStringSubmatch(response, -1) {
		if len(match) >= 5 {
			entities = append(entities, []string{
				strings.TrimSpace(match[1]),
				strings.TrimSpace(match[2]),
				strings.TrimSpace(match[3]),
				strings.TrimSpace(match[4]),
			})
		}
	}

	for _, match := range relRe.FindAllStringSubmatch(response, -1) {
		if len(match) >= 7 {
			relationships = append(relationships, []string{
				strings.TrimSpace(match[1]),
				strings.TrimSpace(match[2]),
				strings.TrimSpace(match[3]),
				strings.TrimSpace(match[4]),
				strings.TrimSpace(match[5]),
				strings.TrimSpace(match[6]),
			})
		}
	}

	return entities, relationships
}

func (kg *KnowledgeGraph) resolveEdgeEntities(edges []*MemoryEdge, nodes []*MemoryNode) []*MemoryEdge {
	// Build name to ID mapping
	nameToID := make(map[string]string)
	for _, node := range nodes {
		nameToID[strings.ToLower(node.Name)] = node.ID
		for alias := range node.Aliases {
			nameToID[strings.ToLower(alias)] = node.ID
		}
	}

	resolved := make([]*MemoryEdge, 0, len(edges))
	for _, edge := range edges {
		// Try to resolve source and target
		sourceID, sourceFound := nameToID[strings.ToLower(edge.SourceID)]
		targetID, targetFound := nameToID[strings.ToLower(edge.TargetID)]

		if sourceFound && targetFound {
			newEdge := edge.Clone()
			newEdge.SourceID = sourceID
			newEdge.TargetID = targetID
			newEdge.ID = newEdge.GenerateID()
			resolved = append(resolved, newEdge)
		}
	}

	return resolved
}

func parseDate(s string) *time.Time {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "none" || s == "present" {
		return nil
	}

	// Try various formats
	formats := []string{
		"2006-01-02",
		"2006-01",
		"2006",
		"January 2006",
		"Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}

	// Try to extract year
	yearRe := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	if match := yearRe.FindString(s); match != "" {
		if year, err := strconv.Atoi(match); err == nil {
			t := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
			return &t
		}
	}

	return nil
}

