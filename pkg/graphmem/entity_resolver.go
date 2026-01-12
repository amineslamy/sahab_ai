package graphmem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
)

// stopwords are common words to ignore in entity names.
var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "and": true,
	"inc": true, "ltd": true, "llc": true, "corp": true, "co": true,
	"company": true, "limited": true, "corporation": true,
}

// EntityCandidate represents a candidate entity for resolution.
type EntityCandidate struct {
	Name        string
	EntityType  string
	Description string
	Tokens      map[string]bool
	Embedding   []float32
	Aliases     map[string]bool
	Occurrences int
}

// EntityResolver resolves and merges duplicate entities.
type EntityResolver struct {
	embeddings          EmbeddingProvider
	similarityThreshold float64
	tokenThreshold      float64

	mu          sync.RWMutex
	entityIndex map[string]*EntityCandidate // key -> candidate
	aliasLookup map[string]string           // alias -> key
}

// NewEntityResolver creates a new EntityResolver.
func NewEntityResolver(embeddings EmbeddingProvider) *EntityResolver {
	return &EntityResolver{
		embeddings:          embeddings,
		similarityThreshold: 0.85,
		tokenThreshold:      0.7,
		entityIndex:         make(map[string]*EntityCandidate),
		aliasLookup:         make(map[string]string),
	}
}

// Resolve deduplicates entities.
func (r *EntityResolver) Resolve(nodes []*MemoryNode, memoryID, userID string) ([]*MemoryNode, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	resolved := make([]*MemoryNode, 0)
	resolvedIDs := make(map[string]bool)

	for _, node := range nodes {
		candidate, key := r.findMatch(node)

		if candidate != nil {
			// Merge with existing
			r.mergeCandidate(candidate, node)
			r.entityIndex[key] = candidate

			resolvedNode := r.toMemoryNode(candidate, key, userID, memoryID)
			if !resolvedIDs[resolvedNode.ID] {
				resolved = append(resolved, resolvedNode)
				resolvedIDs[resolvedNode.ID] = true
			}
		} else {
			// New entity
			candidate := r.createCandidate(node)
			key := r.generateKey(node.Name)
			r.entityIndex[key] = candidate
			r.registerAlias(node.Name, key)

			for alias := range node.Aliases {
				r.registerAlias(alias, key)
			}

			resolvedNode := r.toMemoryNode(candidate, key, userID, memoryID)
			if !resolvedIDs[resolvedNode.ID] {
				resolved = append(resolved, resolvedNode)
				resolvedIDs[resolvedNode.ID] = true
			}
		}
	}

	log.Printf("Resolved %d nodes to %d unique entities", len(nodes), len(resolved))
	return resolved, nil
}

func (r *EntityResolver) findMatch(node *MemoryNode) (*EntityCandidate, string) {
	cleanedName := cleanName(node.Name)
	tokens := tokenize(cleanedName)

	// Check direct alias
	if key, found := r.aliasLookup[strings.ToLower(cleanedName)]; found {
		if candidate, found := r.entityIndex[key]; found {
			return candidate, key
		}
	}

	// Check token-based and embedding similarity
	var bestMatch *EntityCandidate
	var bestKey string
	bestScore := 0.0

	for key, candidate := range r.entityIndex {
		// Skip different entity types
		if node.EntityType != "" && candidate.EntityType != "" &&
			!strings.EqualFold(node.EntityType, candidate.EntityType) {
			continue
		}

		// Token similarity
		tokenScore := tokenSimilarity(tokens, candidate.Tokens)

		// Embedding similarity
		embScore := 0.0
		if len(node.Embedding) > 0 && len(candidate.Embedding) > 0 {
			embScore = CosineSimilarity(node.Embedding, candidate.Embedding)
		}

		// Combined score
		score := tokenScore
		if embScore > 0 {
			score = max(score, 0.4*tokenScore+0.6*embScore)
		}

		if score > bestScore && (tokenScore >= r.tokenThreshold || embScore >= r.similarityThreshold) {
			bestScore = score
			bestMatch = candidate
			bestKey = key
		}
	}

	return bestMatch, bestKey
}

func (r *EntityResolver) createCandidate(node *MemoryNode) *EntityCandidate {
	cleanedName := cleanName(node.Name)

	aliases := make(map[string]bool)
	aliases[node.Name] = true
	aliases[cleanedName] = true
	for alias := range node.Aliases {
		aliases[alias] = true
	}

	tokens := tokenize(cleanedName)
	for alias := range aliases {
		for token := range tokenize(alias) {
			tokens[token] = true
		}
	}

	return &EntityCandidate{
		Name:        node.Name,
		EntityType:  node.EntityType,
		Description: node.Description,
		Tokens:      tokens,
		Embedding:   node.Embedding,
		Aliases:     aliases,
		Occurrences: 1,
	}
}

func (r *EntityResolver) mergeCandidate(candidate *EntityCandidate, node *MemoryNode) {
	// Add aliases
	candidate.Aliases[node.Name] = true
	candidate.Aliases[cleanName(node.Name)] = true
	for alias := range node.Aliases {
		candidate.Aliases[alias] = true
	}

	// Update tokens
	for alias := range candidate.Aliases {
		for token := range tokenize(alias) {
			candidate.Tokens[token] = true
		}
	}

	// Update description if longer
	if len(node.Description) > len(candidate.Description) {
		candidate.Description = node.Description
	}

	// Blend embeddings
	if len(node.Embedding) > 0 && len(candidate.Embedding) > 0 {
		weight := float32(candidate.Occurrences) / float32(candidate.Occurrences+1)
		for i := range candidate.Embedding {
			candidate.Embedding[i] = candidate.Embedding[i]*weight + node.Embedding[i]*(1-weight)
		}
	} else if len(node.Embedding) > 0 {
		candidate.Embedding = node.Embedding
	}

	// Choose better display name
	if len(strings.Fields(node.Name)) > len(strings.Fields(candidate.Name)) {
		candidate.Name = node.Name
	}

	candidate.Occurrences++

	// Register new aliases
	key := r.generateKey(candidate.Name)
	r.registerAlias(node.Name, key)
}

func (r *EntityResolver) toMemoryNode(candidate *EntityCandidate, id, userID, memoryID string) *MemoryNode {
	node := NewMemoryNode(candidate.Name, candidate.EntityType, userID, memoryID)
	node.ID = id
	node.Description = candidate.Description
	node.CanonicalName = candidate.Name
	node.Embedding = candidate.Embedding

	for alias := range candidate.Aliases {
		node.AddAlias(alias)
	}

	node.Properties["occurrence_count"] = candidate.Occurrences

	return node
}

func (r *EntityResolver) generateKey(name string) string {
	cleaned := strings.ToLower(cleanName(name))
	hash := sha256.Sum256([]byte(cleaned))
	return hex.EncodeToString(hash[:16])
}

func (r *EntityResolver) registerAlias(alias, key string) {
	if alias == "" {
		return
	}
	lowered := strings.ToLower(alias)
	r.aliasLookup[lowered] = key
}

// Clear resets the resolver state.
func (r *EntityResolver) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entityIndex = make(map[string]*EntityCandidate)
	r.aliasLookup = make(map[string]string)
}

// GetEmbedding gets embedding for text (used for entity resolution).
func (r *EntityResolver) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	if r.embeddings == nil {
		return nil, fmt.Errorf("embedding provider not configured")
	}
	return r.embeddings.Embed(ctx, text)
}

// Helper functions

func cleanName(name string) string {
	if name == "" {
		return ""
	}
	re := regexp.MustCompile(`\s+`)
	cleaned := re.ReplaceAllString(name, " ")
	cleaned = strings.Trim(cleaned, ` "'`)
	return cleaned
}

func tokenize(name string) map[string]bool {
	if name == "" {
		return nil
	}
	tokens := make(map[string]bool)
	re := regexp.MustCompile(`[a-z0-9]+`)
	for _, match := range re.FindAllString(strings.ToLower(name), -1) {
		if !stopwords[match] {
			tokens[match] = true
		}
	}
	return tokens
}

func tokenSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	intersection := 0
	for token := range a {
		if b[token] {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}
