package graphmem

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

// SemanticSearch provides semantic search over memory nodes.
type SemanticSearch struct {
	embeddings     EmbeddingProvider
	cache          Cache
	topK           int
	minSimilarity  float64
	neo4jStore     *Neo4jStore
	memoryID       string
	userID         string
	useNeo4jVector bool

	mu         sync.RWMutex
	index      map[string][]float32
	nodeLookup map[string]*MemoryNode
}

// SemanticSearchConfig contains configuration for SemanticSearch.
type SemanticSearchConfig struct {
	TopK          int
	MinSimilarity float64
	MemoryID      string
	UserID        string
}

// NewSemanticSearch creates a new semantic search engine.
func NewSemanticSearch(embeddings EmbeddingProvider, cache Cache, config *SemanticSearchConfig) *SemanticSearch {
	if config == nil {
		config = &SemanticSearchConfig{}
	}
	if config.TopK <= 0 {
		config.TopK = 10
	}
	if config.MinSimilarity <= 0 {
		config.MinSimilarity = 0.5
	}
	if config.UserID == "" {
		config.UserID = "default"
	}

	return &SemanticSearch{
		embeddings:    embeddings,
		cache:         cache,
		topK:          config.TopK,
		minSimilarity: config.MinSimilarity,
		memoryID:      config.MemoryID,
		userID:        config.UserID,
		index:         make(map[string][]float32),
		nodeLookup:    make(map[string]*MemoryNode),
	}
}

// IndexNodes indexes nodes for search.
func (ss *SemanticSearch) IndexNodes(ctx context.Context, nodes []*MemoryNode) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	for _, node := range nodes {
		text := node.Description
		if text == "" {
			text = node.Name
		}

		embedding, err := ss.embeddings.Embed(ctx, text)
		if err != nil {
			log.Printf("Failed to index node %s: %v", node.ID, err)
			continue
		}

		if len(embedding) > 0 {
			ss.index[node.ID] = embedding
			ss.nodeLookup[node.ID] = node
		}
	}

	log.Printf("Indexed %d nodes", len(ss.index))
	return nil
}

// Search searches for similar nodes.
func (ss *SemanticSearch) Search(ctx context.Context, query string, topK int, minSimilarity float64, filters map[string]any) ([]*NodeWithScore, error) {
	if topK <= 0 {
		topK = ss.topK
	}
	if minSimilarity <= 0 {
		minSimilarity = ss.minSimilarity
	}

	// Generate query hash for caching
	filterJSON, _ := json.Marshal(filters)
	queryHash := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%d:%f:%s", query, topK, minSimilarity, string(filterJSON)))))

	// Check cache first
	if ss.cache != nil && ss.memoryID != "" {
		if cached, ok := ss.cache.Get(fmt.Sprintf("search:%s:%s:%s", ss.userID, ss.memoryID, queryHash)); ok {
			if results, ok := cached.([]*NodeWithScore); ok {
				log.Printf("Cache hit for search query: %s", query[:min(50, len(query))])
				return results, nil
			}
		}
	}

	// Get query embedding
	queryEmbedding, err := ss.embeddings.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	if len(queryEmbedding) == 0 {
		return nil, nil
	}

	// Use Neo4j vector search if available
	if ss.useNeo4jVector && ss.neo4jStore != nil && ss.memoryID != "" {
		results, err := ss.neo4jStore.VectorSearch(ctx, ss.memoryID, queryEmbedding, topK, minSimilarity, ss.userID)
		if err == nil {
			nodeResults := make([]*NodeWithScore, len(results))
			for i, r := range results {
				nodeResults[i] = &NodeWithScore{Node: r.Node, Score: r.Score}
			}

			// Apply filters
			if filters != nil {
				nodeResults = ss.applyFilters(nodeResults, filters)
			}

			// Cache results
			ss.cacheResults(queryHash, nodeResults)

			log.Printf("Neo4j vector search returned %d results", len(nodeResults))
			return nodeResults, nil
		}
		log.Printf("Neo4j vector search failed, falling back to in-memory: %v", err)
	}

	// Fallback: In-memory search with hybrid alias matching
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if len(ss.index) == 0 {
		return nil, nil
	}

	// Pre-process query for alias matching
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)
	queryWordSet := make(map[string]bool)
	for _, w := range queryWords {
		queryWordSet[w] = true
	}

	results := make([]*NodeWithScore, 0)

	for nodeID, nodeVector := range ss.index {
		similarity := ss.cosineSimilarity(queryEmbedding, nodeVector)

		// Check for alias matches (hybrid search)
		aliasBoost := 0.0
		node := ss.nodeLookup[nodeID]
		if node != nil && len(node.Aliases) > 0 {
			for alias := range node.Aliases {
				aliasLower := strings.ToLower(alias)
				// Exact alias match in query
				if strings.Contains(queryLower, aliasLower) {
					aliasBoost = max(aliasBoost, 0.3)
					log.Printf("Alias match: '%s' in query '%s'", alias, query[:min(50, len(query))])
					break
				}
				// Partial word match
				aliasWords := strings.Fields(aliasLower)
				for _, aw := range aliasWords {
					if _, ok := queryWordSet[aw]; ok {
						aliasBoost = max(aliasBoost, 0.15)
						break
					}
				}
			}

			// Check canonical name
			if node.CanonicalName != "" && strings.Contains(queryLower, strings.ToLower(node.CanonicalName)) {
				aliasBoost = max(aliasBoost, 0.25)
			}

			// Check node name
			if strings.Contains(queryLower, strings.ToLower(node.Name)) {
				aliasBoost = max(aliasBoost, 0.25)
			}
		}

		// Adjust effective similarity with alias boost
		effectiveSimilarity := min(1.0, similarity+aliasBoost)

		if effectiveSimilarity >= minSimilarity && node != nil {
			// Skip nodes that have decayed too much (EPHEMERAL = 0)
			if node.Importance == ImportanceEphemeral {
				continue
			}

			// Weight similarity by importance
			importanceWeight := float64(node.Importance.Value()) / 10.0

			// Recency boost
			recencyBoost := 0.0
			if !node.AccessedAt.IsZero() {
				hoursSinceAccess := time.Since(node.AccessedAt).Hours()
				if hoursSinceAccess < 24 {
					recencyBoost = 0.1 * (1 - hoursSinceAccess/24)
				}
			}

			// Access count boost
			accessBoost := min(0.1, float64(node.AccessCount)*0.01)

			// Combined score: 50% similarity + 25% importance + 15% alias + 5% recency + 5% access
			combinedScore := 0.50*similarity + 0.25*importanceWeight + 0.15*(aliasBoost*3.33) + recencyBoost + accessBoost

			results = append(results, &NodeWithScore{
				Node:  node,
				Score: combinedScore,
			})
		}
	}

	// Apply filters
	if filters != nil {
		results = ss.applyFilters(results, filters)
	}

	// Sort by combined score and limit
	ss.sortByScore(results)
	if len(results) > topK {
		results = results[:topK]
	}

	// Cache results
	ss.cacheResults(queryHash, results)

	return results, nil
}

// NodeWithScore represents a node with its similarity score.
type NodeWithScore struct {
	Node  *MemoryNode
	Score float64
}

// EnableNeo4jVector enables Neo4j vector search.
func (ss *SemanticSearch) EnableNeo4jVector(store *Neo4jStore, memoryID string) bool {
	ss.neo4jStore = store
	ss.memoryID = memoryID

	ctx := context.Background()
	if store.EnsureVectorIndex(ctx, memoryID) {
		ss.useNeo4jVector = store.HasVectorSupport()
		if ss.useNeo4jVector {
			log.Printf("Neo4j vector search enabled")
		}
		return ss.useNeo4jVector
	}

	return false
}

// UsingNeo4jVector returns whether Neo4j vector search is being used.
func (ss *SemanticSearch) UsingNeo4jVector() bool {
	return ss.useNeo4jVector
}

// FindSimilarEntities finds entity names similar to query.
func (ss *SemanticSearch) FindSimilarEntities(ctx context.Context, query string, entityNames []string, topK int) ([]string, error) {
	if len(entityNames) == 0 {
		return nil, nil
	}

	queryEmbedding, err := ss.embeddings.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	entityEmbeddings, err := ss.embeddings.EmbedBatch(ctx, entityNames)
	if err != nil {
		return nil, err
	}

	if len(queryEmbedding) == 0 || len(entityEmbeddings) == 0 {
		return nil, nil
	}

	type scored struct {
		name  string
		score float64
	}

	scoredEntities := make([]scored, 0, len(entityNames))
	for i, name := range entityNames {
		if i < len(entityEmbeddings) {
			sim := ss.cosineSimilarity(queryEmbedding, entityEmbeddings[i])
			scoredEntities = append(scoredEntities, scored{name: name, score: sim})
		}
	}

	// Sort by score
	for i := 0; i < len(scoredEntities); i++ {
		for j := i + 1; j < len(scoredEntities); j++ {
			if scoredEntities[j].score > scoredEntities[i].score {
				scoredEntities[i], scoredEntities[j] = scoredEntities[j], scoredEntities[i]
			}
		}
	}

	// Take top K
	if len(scoredEntities) > topK {
		scoredEntities = scoredEntities[:topK]
	}

	result := make([]string, len(scoredEntities))
	for i, s := range scoredEntities {
		result[i] = s.name
	}

	return result, nil
}

// cosineSimilarity calculates cosine similarity between two vectors.
func (ss *SemanticSearch) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// applyFilters applies filters to search results.
func (ss *SemanticSearch) applyFilters(results []*NodeWithScore, filters map[string]any) []*NodeWithScore {
	filtered := make([]*NodeWithScore, 0, len(results))

	for _, r := range results {
		if ss.matchesFilters(r.Node, filters) {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

// matchesFilters checks if a node matches filters.
func (ss *SemanticSearch) matchesFilters(node *MemoryNode, filters map[string]any) bool {
	for key, value := range filters {
		switch key {
		case "entity_type":
			if strVal, ok := value.(string); ok {
				if !strings.EqualFold(node.EntityType, strVal) {
					return false
				}
			}
		case "min_importance":
			if minVal, ok := value.(int); ok {
				if node.Importance.Value() < minVal {
					return false
				}
			}
		case "state":
			if strVal, ok := value.(string); ok {
				if node.State.String() != strVal {
					return false
				}
			}
		default:
			if node.Properties != nil {
				if propVal, ok := node.Properties[key]; ok {
					if propVal != value {
						return false
					}
				}
			}
		}
	}
	return true
}

// sortByScore sorts results by score in descending order.
func (ss *SemanticSearch) sortByScore(results []*NodeWithScore) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// cacheResults caches search results.
func (ss *SemanticSearch) cacheResults(queryHash string, results []*NodeWithScore) {
	if ss.cache != nil && ss.memoryID != "" && len(results) > 0 {
		key := fmt.Sprintf("search:%s:%s:%s", ss.userID, ss.memoryID, queryHash)
		if err := ss.cache.Set(key, results); err != nil {
			log.Printf("Failed to cache search results: %v", err)
		}
	}
}

// ClearIndex clears the search index.
func (ss *SemanticSearch) ClearIndex() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.index = make(map[string][]float32)
	ss.nodeLookup = make(map[string]*MemoryNode)
}
