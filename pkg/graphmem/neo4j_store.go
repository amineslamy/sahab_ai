package graphmem

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jStore implements Store for Neo4j persistent storage.
type Neo4jStore struct {
	uri                 string
	username            string
	password            string
	database            string
	maxRetries          int
	retryDelay          time.Duration
	embeddingDimensions int
	useVectorIndex      bool
	vectorIndexCreated  bool
	vectorIndexName     string

	driver neo4j.DriverWithContext
}

// Neo4jStoreOptions contains options for Neo4jStore.
type Neo4jStoreOptions struct {
	URI                 string
	Username            string
	Password            string
	Database            string
	MaxRetries          int
	RetryDelay          time.Duration
	EmbeddingDimensions int
	UseVectorIndex      bool
}

// NewNeo4jStore creates a new Neo4j store.
func NewNeo4jStore(opts *Neo4jStoreOptions) (*Neo4jStore, error) {
	if opts == nil {
		return nil, fmt.Errorf("options required")
	}

	if opts.Database == "" {
		opts.Database = "neo4j"
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 3
	}
	if opts.RetryDelay == 0 {
		opts.RetryDelay = 5 * time.Second
	}
	if opts.EmbeddingDimensions <= 0 {
		opts.EmbeddingDimensions = 1536
	}

	store := &Neo4jStore{
		uri:                 opts.URI,
		username:            opts.Username,
		password:            opts.Password,
		database:            opts.Database,
		maxRetries:          opts.MaxRetries,
		retryDelay:          opts.RetryDelay,
		embeddingDimensions: opts.EmbeddingDimensions,
		useVectorIndex:      opts.UseVectorIndex,
	}

	// Create driver
	driver, err := neo4j.NewDriverWithContext(
		opts.URI,
		neo4j.BasicAuth(opts.Username, opts.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	store.driver = driver
	return store, nil
}

// executeQuery executes a query with retry logic.
func (s *Neo4jStore) executeQuery(ctx context.Context, query string, params map[string]any, write bool) ([]map[string]any, error) {
	if params == nil {
		params = make(map[string]any)
	}

	var results []map[string]any
	var lastErr error

	for attempt := 0; attempt < s.maxRetries; attempt++ {
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: s.database,
			AccessMode:   neo4j.AccessModeRead,
		})
		if write {
			session = s.driver.NewSession(ctx, neo4j.SessionConfig{
				DatabaseName: s.database,
				AccessMode:   neo4j.AccessModeWrite,
			})
		}

		var err error
		if write {
			results, err = neo4j.ExecuteWrite(ctx, session, func(tx neo4j.ManagedTransaction) ([]map[string]any, error) {
				result, err := tx.Run(ctx, query, params)
				if err != nil {
					return nil, err
				}
				return collectResults(ctx, result)
			})
		} else {
			results, err = neo4j.ExecuteRead(ctx, session, func(tx neo4j.ManagedTransaction) ([]map[string]any, error) {
				result, err := tx.Run(ctx, query, params)
				if err != nil {
					return nil, err
				}
				return collectResults(ctx, result)
			})
		}

		session.Close(ctx)

		if err == nil {
			return results, nil
		}

		lastErr = err
		log.Printf("Neo4j query failed (attempt %d): %v", attempt+1, err)

		if attempt < s.maxRetries-1 {
			time.Sleep(s.retryDelay)
		}
	}

	return nil, NewStorageError(fmt.Sprintf("Neo4j query failed after %d attempts", s.maxRetries)).
		WithStorageType("neo4j").
		WithOperation("query").
		WithCause(lastErr)
}

func collectResults(ctx context.Context, result neo4j.ResultWithContext) ([]map[string]any, error) {
	var records []map[string]any
	for result.Next(ctx) {
		record := result.Record()
		recordMap := make(map[string]any)
		keys := record.Keys
		for _, key := range keys {
			recordMap[key], _ = record.Get(key)
		}
		records = append(records, recordMap)
	}
	return records, result.Err()
}

// SaveMemory saves a memory to Neo4j.
func (s *Neo4jStore) SaveMemory(memory *Memory) error {
	ctx := context.Background()

	// Save memory metadata
	_, err := s.executeQuery(ctx, `
		MERGE (m:Memory {id: $id})
		SET m.name = $name,
			m.description = $description,
			m.importance = $importance,
			m.state = $state,
			m.version = $version,
			m.created_at = $created_at,
			m.updated_at = $updated_at
	`, map[string]any{
		"id":          memory.ID,
		"name":        memory.Name,
		"description": memory.Description,
		"importance":  memory.Importance.Value(),
		"state":       memory.State.String(),
		"version":     memory.Version,
		"created_at":  memory.CreatedAt.Format(time.RFC3339),
		"updated_at":  time.Now().UTC().Format(time.RFC3339),
	}, true)
	if err != nil {
		return err
	}

	// Save nodes in batches
	if err := s.saveNodesBatch(ctx, memory.ID, memory.GetAllNodes()); err != nil {
		return err
	}

	// Save edges in batches
	if err := s.saveEdgesBatch(ctx, memory.ID, memory.GetAllEdges()); err != nil {
		return err
	}

	// Save clusters
	if err := s.saveClusters(ctx, memory.ID, memory.GetAllClusters()); err != nil {
		return err
	}

	log.Printf("Saved memory %s: %d nodes, %d edges", memory.ID, len(memory.Nodes), len(memory.Edges))
	return nil
}

func (s *Neo4jStore) saveNodesBatch(ctx context.Context, memoryID string, nodes []*MemoryNode) error {
	const batchSize = 500

	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}
		batch := nodes[i:end]

		nodeData := make([]map[string]any, len(batch))
		for j, n := range batch {
			properties, _ := json.Marshal(n.Properties)
			aliases := make([]string, 0, len(n.Aliases))
			for alias := range n.Aliases {
				aliases = append(aliases, alias)
			}

			userID := n.UserID
			if userID == "" {
				userID = "default"
			}

			nodeData[j] = map[string]any{
				"id":             n.ID,
				"name":           n.Name,
				"entity_type":    n.EntityType,
				"description":    n.Description,
				"canonical_name": n.CanonicalName,
				"aliases":        aliases,
				"properties":     string(properties),
				"importance":     n.Importance.Value(),
				"state":          n.State.String(),
				"access_count":   n.AccessCount,
				"created_at":     n.CreatedAt.Format(time.RFC3339),
				"updated_at":     n.UpdatedAt.Format(time.RFC3339),
				"accessed_at":    n.AccessedAt.Format(time.RFC3339),
				"embedding":      n.Embedding,
				"user_id":        userID,
			}
		}

		_, err := s.executeQuery(ctx, `
			UNWIND $nodes AS node
			MERGE (n:Entity {id: node.id, user_id: node.user_id, memory_id: $memory_id})
			SET n.name = node.name,
				n.entity_type = node.entity_type,
				n.description = node.description,
				n.canonical_name = node.canonical_name,
				n.aliases = node.aliases,
				n.properties = node.properties,
				n.importance = node.importance,
				n.state = node.state,
				n.access_count = node.access_count,
				n.created_at = node.created_at,
				n.updated_at = node.updated_at,
				n.accessed_at = node.accessed_at,
				n.embedding = node.embedding
		`, map[string]any{
			"memory_id": memoryID,
			"nodes":     nodeData,
		}, true)
		if err != nil {
			return err
		}
	}

	// Ensure vector index exists after saving nodes with embeddings
	if s.useVectorIndex {
		s.EnsureVectorIndex(ctx, memoryID)
	}

	return nil
}

func (s *Neo4jStore) saveEdgesBatch(ctx context.Context, memoryID string, edges []*MemoryEdge) error {
	const batchSize = 500

	for i := 0; i < len(edges); i += batchSize {
		end := i + batchSize
		if end > len(edges) {
			end = len(edges)
		}
		batch := edges[i:end]

		edgeData := make([]map[string]any, len(batch))
		for j, e := range batch {
			properties, _ := json.Marshal(e.Properties)

			var validFrom, validUntil *string
			if e.ValidFrom != nil && !e.ValidFrom.IsZero() {
				v := e.ValidFrom.Format(time.RFC3339)
				validFrom = &v
			}
			if e.ValidUntil != nil && !e.ValidUntil.IsZero() {
				v := e.ValidUntil.Format(time.RFC3339)
				validUntil = &v
			}

			edgeData[j] = map[string]any{
				"id":            e.ID,
				"source_id":     e.SourceID,
				"target_id":     e.TargetID,
				"relation_type": e.RelationType,
				"description":   e.Description,
				"weight":        e.Weight,
				"confidence":    e.Confidence,
				"properties":    string(properties),
				"importance":    e.Importance.Value(),
				"state":         e.State.String(),
				"valid_from":    validFrom,
				"valid_until":   validUntil,
			}
		}

		_, err := s.executeQuery(ctx, `
			UNWIND $edges AS edge
			MATCH (s:Entity {id: edge.source_id, memory_id: $memory_id})
			MATCH (t:Entity {id: edge.target_id, memory_id: $memory_id})
			MERGE (s)-[r:RELATED {id: edge.id}]->(t)
			SET r.relation_type = edge.relation_type,
				r.description = edge.description,
				r.weight = edge.weight,
				r.confidence = edge.confidence,
				r.properties = edge.properties,
				r.importance = edge.importance,
				r.state = edge.state,
				r.memory_id = $memory_id,
				r.valid_from = edge.valid_from,
				r.valid_until = edge.valid_until
		`, map[string]any{
			"memory_id": memoryID,
			"edges":     edgeData,
		}, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Neo4jStore) saveClusters(ctx context.Context, memoryID string, clusters []*MemoryCluster) error {
	for _, cluster := range clusters {
		_, err := s.executeQuery(ctx, `
			MERGE (c:Community {id: $id, memory_id: $memory_id})
			SET c.summary = $summary,
				c.entities = $entities,
				c.importance = $importance,
				c.coherence_score = $coherence_score,
				c.density = $density,
				c.updated_at = $updated_at
		`, map[string]any{
			"memory_id":       memoryID,
			"id":              cluster.ID,
			"summary":         cluster.Summary,
			"entities":        cluster.Entities,
			"importance":      cluster.Importance.Value(),
			"coherence_score": cluster.CoherenceScore,
			"density":         cluster.Density,
			"updated_at":      time.Now().UTC().Format(time.RFC3339),
		}, true)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadMemory loads a memory from Neo4j.
func (s *Neo4jStore) LoadMemory(memoryID, userID string) (*Memory, error) {
	ctx := context.Background()

	// Load memory metadata
	result, err := s.executeQuery(ctx, `
		MATCH (m:Memory {id: $id})
		RETURN m
	`, map[string]any{"id": memoryID}, false)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	// Create memory object
	memory := NewMemory(memoryID)

	// Load nodes
	nodes, err := s.loadNodes(ctx, memoryID, userID)
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		memory.Nodes[node.ID] = node
	}

	// Load edges
	edges, err := s.loadEdges(ctx, memoryID, nil)
	if err != nil {
		return nil, err
	}
	for _, edge := range edges {
		memory.Edges[edge.ID] = edge
	}

	// Load clusters
	clusters, err := s.loadClusters(ctx, memoryID)
	if err != nil {
		return nil, err
	}
	for _, cluster := range clusters {
		memory.Clusters[cluster.ID] = cluster
	}

	log.Printf("Loaded memory %s for user %s: %d nodes", memoryID, userID, len(memory.Nodes))
	return memory, nil
}

func (s *Neo4jStore) loadNodes(ctx context.Context, memoryID, userID string) ([]*MemoryNode, error) {
	query := `
		MATCH (n:Entity {user_id: $user_id, memory_id: $memory_id})
		RETURN n
	`
	if userID == "" {
		userID = "default"
	}

	result, err := s.executeQuery(ctx, query, map[string]any{
		"user_id":   userID,
		"memory_id": memoryID,
	}, false)
	if err != nil {
		return nil, err
	}

	nodes := make([]*MemoryNode, 0, len(result))
	for _, record := range result {
		nodeVal, ok := record["n"]
		if !ok {
			continue
		}

		n, ok := nodeVal.(neo4j.Node)
		if !ok {
			continue
		}

		props := n.Props

		var properties map[string]any
		if propsStr, ok := props["properties"].(string); ok {
			json.Unmarshal([]byte(propsStr), &properties)
		}

		aliases := make(map[string]bool)
		if aliasSlice, ok := props["aliases"].([]any); ok {
			for _, a := range aliasSlice {
				if str, ok := a.(string); ok {
					aliases[str] = true
				}
			}
		}

		var embedding []float32
		if embSlice, ok := props["embedding"].([]any); ok {
			embedding = make([]float32, len(embSlice))
			for i, v := range embSlice {
				if f, ok := v.(float64); ok {
					embedding[i] = float32(f)
				}
			}
		}

		importance := ImportanceMedium
		if impVal, ok := props["importance"].(int64); ok {
			importance = MemoryImportance(impVal)
		}

		state := StateActive
		if stateStr, ok := props["state"].(string); ok {
			switch stateStr {
			case "ARCHIVED":
				state = StateArchived
			case "DELETED":
				state = StateDeleted
			}
		}

		node := &MemoryNode{
			ID:            getString(props, "id"),
			Name:          getString(props, "name"),
			EntityType:    getString(props, "entity_type"),
			Description:   getString(props, "description"),
			CanonicalName: getString(props, "canonical_name"),
			Aliases:       aliases,
			Properties:    properties,
			Embedding:     embedding,
			Importance:    importance,
			State:         state,
			AccessCount:   getInt(props, "access_count"),
			MemoryID:      memoryID,
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (s *Neo4jStore) loadEdges(ctx context.Context, memoryID string, validAt *time.Time) ([]*MemoryEdge, error) {
	result, err := s.executeQuery(ctx, `
		MATCH (s:Entity {memory_id: $memory_id})-[r:RELATED {memory_id: $memory_id}]->(t:Entity {memory_id: $memory_id})
		RETURN r, s.id AS source_id, t.id AS target_id
	`, map[string]any{"memory_id": memoryID}, false)
	if err != nil {
		return nil, err
	}

	edges := make([]*MemoryEdge, 0, len(result))
	for _, record := range result {
		relVal, ok := record["r"]
		if !ok {
			continue
		}

		r, ok := relVal.(neo4j.Relationship)
		if !ok {
			continue
		}

		props := r.Props

		var properties map[string]any
		if propsStr, ok := props["properties"].(string); ok {
			json.Unmarshal([]byte(propsStr), &properties)
		}

		var validFrom, validUntil *time.Time
		if vf, ok := props["valid_from"].(string); ok && vf != "" {
			t, _ := time.Parse(time.RFC3339, vf)
			validFrom = &t
		}
		if vu, ok := props["valid_until"].(string); ok && vu != "" {
			t, _ := time.Parse(time.RFC3339, vu)
			validUntil = &t
		}

		importance := ImportanceMedium
		if impVal, ok := props["importance"].(int64); ok {
			importance = MemoryImportance(impVal)
		}

		state := StateActive
		if stateStr, ok := props["state"].(string); ok {
			switch stateStr {
			case "ARCHIVED":
				state = StateArchived
			case "DELETED":
				state = StateDeleted
			}
		}

		edge := &MemoryEdge{
			ID:           getString(props, "id"),
			SourceID:     getString(record, "source_id"),
			TargetID:     getString(record, "target_id"),
			RelationType: getString(props, "relation_type"),
			Description:  getString(props, "description"),
			Weight:       getFloat(props, "weight", 1.0),
			Confidence:   getFloat(props, "confidence", 1.0),
			Properties:   properties,
			Importance:   importance,
			State:        state,
			MemoryID:     memoryID,
			ValidFrom:    validFrom,
			ValidUntil:   validUntil,
		}

		// Apply temporal filter if specified
		if validAt != nil {
			if !edge.IsValidAt(validAt) {
				continue
			}
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

func (s *Neo4jStore) loadClusters(ctx context.Context, memoryID string) ([]*MemoryCluster, error) {
	result, err := s.executeQuery(ctx, `
		MATCH (c:Community {memory_id: $memory_id})
		RETURN c
	`, map[string]any{"memory_id": memoryID}, false)
	if err != nil {
		return nil, err
	}

	clusters := make([]*MemoryCluster, 0, len(result))
	for _, record := range result {
		cVal, ok := record["c"]
		if !ok {
			continue
		}

		c, ok := cVal.(neo4j.Node)
		if !ok {
			continue
		}

		props := c.Props

		entities := make([]string, 0)
		if entSlice, ok := props["entities"].([]any); ok {
			for _, e := range entSlice {
				if str, ok := e.(string); ok {
					entities = append(entities, str)
				}
			}
		}

		importance := ImportanceMedium
		if impVal, ok := props["importance"].(int64); ok {
			importance = MemoryImportance(impVal)
		}

		cluster := &MemoryCluster{
			ID:             getString(props, "id"),
			Summary:        getString(props, "summary"),
			Entities:       entities,
			Importance:     importance,
			CoherenceScore: getFloat(props, "coherence_score", 1.0),
			Density:        getFloat(props, "density", 1.0),
			MemoryID:       memoryID,
		}

		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// QueryEdgesAtTime queries edges that were valid at a specific point in time.
func (s *Neo4jStore) QueryEdgesAtTime(memoryID string, queryTime time.Time, sourceID, targetID, relationType *string) ([]*MemoryEdge, error) {
	ctx := context.Background()
	allEdges, err := s.loadEdges(ctx, memoryID, &queryTime)
	if err != nil {
		return nil, err
	}

	// Apply additional filters
	filtered := make([]*MemoryEdge, 0)
	for _, e := range allEdges {
		if sourceID != nil && e.SourceID != *sourceID {
			continue
		}
		if targetID != nil && e.TargetID != *targetID {
			continue
		}
		if relationType != nil && !strings.EqualFold(e.RelationType, *relationType) {
			continue
		}
		filtered = append(filtered, e)
	}

	return filtered, nil
}

// SupersedeRelationship marks a relationship as superseded.
func (s *Neo4jStore) SupersedeRelationship(memoryID, edgeID string, endTime *time.Time) (bool, error) {
	ctx := context.Background()

	if endTime == nil {
		now := time.Now().UTC()
		endTime = &now
	}

	result, err := s.executeQuery(ctx, `
		MATCH ()-[r:RELATED {id: $edge_id, memory_id: $memory_id}]->()
		SET r.valid_until = $end_time,
			r.state = 'ARCHIVED'
		RETURN r
	`, map[string]any{
		"memory_id": memoryID,
		"edge_id":   edgeID,
		"end_time":  endTime.Format(time.RFC3339),
	}, true)
	if err != nil {
		return false, err
	}

	return len(result) > 0, nil
}

// EnsureVectorIndex ensures a vector index exists for Entity.embedding.
func (s *Neo4jStore) EnsureVectorIndex(ctx context.Context, memoryID string) bool {
	if !s.useVectorIndex {
		return false
	}

	if s.vectorIndexCreated {
		return true
	}

	// Check Neo4j version
	versionResult, err := s.executeQuery(ctx, "CALL dbms.components() YIELD versions RETURN versions[0] AS version", nil, false)
	if err == nil && len(versionResult) > 0 {
		if version, ok := versionResult[0]["version"].(string); ok {
			parts := strings.Split(version, ".")
			if len(parts) > 0 {
				var major int
				fmt.Sscanf(parts[0], "%d", &major)
				if major < 5 {
					log.Printf("Neo4j %s does not support vector indexes. Need 5.x+", version)
					s.useVectorIndex = false
					return false
				}
			}
		}
	}

	// Check for existing vector index
	existing, err := s.executeQuery(ctx, `
		SHOW INDEXES 
		WHERE type = 'VECTOR' 
		AND entityType = 'NODE'
	`, nil, false)
	if err == nil {
		for _, idx := range existing {
			labels, _ := idx["labelsOrTypes"].([]any)
			props, _ := idx["properties"].([]any)
			for _, l := range labels {
				if l == "Entity" {
					for _, p := range props {
						if p == "embedding" {
							if name, ok := idx["name"].(string); ok {
								s.vectorIndexName = name
								s.vectorIndexCreated = true
								log.Printf("Found existing vector index: %s", s.vectorIndexName)
								return true
							}
						}
					}
				}
			}
		}
	}

	// Create new index
	indexName := "graphmem_entity_vector_idx"
	s.vectorIndexName = indexName

	_, err = s.executeQuery(ctx, fmt.Sprintf(`
		CREATE VECTOR INDEX %s IF NOT EXISTS
		FOR (n:Entity)
		ON n.embedding
		OPTIONS {indexConfig: {
			`+"`vector.dimensions`"+`: %d,
			`+"`vector.similarity_function`"+`: 'cosine'
		}}
	`, indexName, s.embeddingDimensions), nil, true)
	if err != nil {
		log.Printf("Could not create vector index: %v", err)
		s.useVectorIndex = false
		return false
	}

	log.Printf("Created vector index %s", indexName)
	s.vectorIndexCreated = true
	return true
}

// VectorSearch performs vector similarity search using Neo4j vector index.
func (s *Neo4jStore) VectorSearch(ctx context.Context, memoryID string, queryEmbedding []float32, topK int, minScore float64, userID string) ([]*MemoryNodeWithScore, error) {
	if !s.useVectorIndex {
		return s.vectorSearchFallback(ctx, memoryID, queryEmbedding, topK, minScore, userID)
	}

	s.EnsureVectorIndex(ctx, memoryID)

	if !s.vectorIndexCreated || s.vectorIndexName == "" {
		return s.vectorSearchFallback(ctx, memoryID, queryEmbedding, topK, minScore, userID)
	}

	// Convert embedding to []any for Neo4j
	embedding := make([]any, len(queryEmbedding))
	for i, v := range queryEmbedding {
		embedding[i] = v
	}

	fetchCount := topK * 10
	if fetchCount > 1000 {
		fetchCount = 1000
	}

	result, err := s.executeQuery(ctx, fmt.Sprintf(`
		CALL db.index.vector.queryNodes('%s', $fetch_count, $embedding)
		YIELD node, score
		WHERE node.user_id = $user_id 
		  AND node.memory_id = $memory_id 
		  AND score >= $min_score
		RETURN node, score
		ORDER BY score DESC
		LIMIT $top_k
	`, s.vectorIndexName), map[string]any{
		"embedding":   embedding,
		"fetch_count": fetchCount,
		"user_id":     userID,
		"memory_id":   memoryID,
		"min_score":   minScore,
		"top_k":       topK,
	}, false)
	if err != nil {
		log.Printf("Vector search failed: %v. Using fallback.", err)
		return s.vectorSearchFallback(ctx, memoryID, queryEmbedding, topK, minScore, userID)
	}

	nodes := make([]*MemoryNodeWithScore, 0, len(result))
	for _, record := range result {
		nodeVal, ok := record["node"]
		if !ok {
			continue
		}
		score, _ := record["score"].(float64)

		n, ok := nodeVal.(neo4j.Node)
		if !ok {
			continue
		}

		props := n.Props

		var properties map[string]any
		if propsStr, ok := props["properties"].(string); ok {
			json.Unmarshal([]byte(propsStr), &properties)
		}

		aliases := make(map[string]bool)
		if aliasSlice, ok := props["aliases"].([]any); ok {
			for _, a := range aliasSlice {
				if str, ok := a.(string); ok {
					aliases[str] = true
				}
			}
		}

		importance := ImportanceMedium
		if impVal, ok := props["importance"].(int64); ok {
			importance = MemoryImportance(impVal)
		}

		// Skip EPHEMERAL nodes
		if importance == ImportanceEphemeral {
			continue
		}

		node := &MemoryNode{
			ID:            getString(props, "id"),
			Name:          getString(props, "name"),
			EntityType:    getString(props, "entity_type"),
			Description:   getString(props, "description"),
			CanonicalName: getString(props, "canonical_name"),
			Aliases:       aliases,
			Properties:    properties,
			Importance:    importance,
			AccessCount:   getInt(props, "access_count"),
			MemoryID:      memoryID,
		}

		nodes = append(nodes, &MemoryNodeWithScore{Node: node, Score: score})
	}

	return nodes, nil
}

func (s *Neo4jStore) vectorSearchFallback(ctx context.Context, memoryID string, queryEmbedding []float32, topK int, minScore float64, userID string) ([]*MemoryNodeWithScore, error) {
	nodes, err := s.loadNodes(ctx, memoryID, userID)
	if err != nil {
		return nil, err
	}

	nodesWithScores := make([]*MemoryNodeWithScore, 0)
	for _, node := range nodes {
		if len(node.Embedding) == 0 {
			continue
		}

		similarity := cosineSimilarity32(queryEmbedding, node.Embedding)
		if similarity < minScore {
			continue
		}

		if node.Importance == ImportanceEphemeral {
			continue
		}

		nodesWithScores = append(nodesWithScores, &MemoryNodeWithScore{
			Node:  node,
			Score: similarity,
		})
	}

	// Sort by score
	for i := 0; i < len(nodesWithScores); i++ {
		for j := i + 1; j < len(nodesWithScores); j++ {
			if nodesWithScores[j].Score > nodesWithScores[i].Score {
				nodesWithScores[i], nodesWithScores[j] = nodesWithScores[j], nodesWithScores[i]
			}
		}
	}

	if len(nodesWithScores) > topK {
		nodesWithScores = nodesWithScores[:topK]
	}

	return nodesWithScores, nil
}

// MemoryNodeWithScore holds a node and its similarity score.
type MemoryNodeWithScore struct {
	Node  *MemoryNode
	Score float64
}

// DeleteMemory deletes a memory from storage.
func (s *Neo4jStore) DeleteMemory(memoryID string) error {
	ctx := context.Background()

	_, err := s.executeQuery(ctx, `
		MATCH (n {memory_id: $memory_id})
		DETACH DELETE n
	`, map[string]any{"memory_id": memoryID}, true)
	if err != nil {
		return err
	}

	_, err = s.executeQuery(ctx, `
		MATCH (m:Memory {id: $memory_id})
		DELETE m
	`, map[string]any{"memory_id": memoryID}, true)
	if err != nil {
		return err
	}

	log.Printf("Cleared memory %s", memoryID)
	return nil
}

// ClearMemory clears all data in a memory.
func (s *Neo4jStore) ClearMemory(memoryID string) error {
	return s.DeleteMemory(memoryID)
}

// ListMemories lists all memory IDs for a user.
func (s *Neo4jStore) ListMemories(userID string) ([]string, error) {
	ctx := context.Background()
	result, err := s.executeQuery(ctx, `
		MATCH (m:Memory)
		RETURN m.id AS id
	`, nil, false)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(result))
	for _, record := range result {
		if id, ok := record["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// Close closes the Neo4j connection.
func (s *Neo4jStore) Close() error {
	if s.driver != nil {
		return s.driver.Close(context.Background())
	}
	return nil
}

// HealthCheck checks if the store is healthy.
func (s *Neo4jStore) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.driver.VerifyConnectivity(ctx)
	return err == nil
}

// HasVectorSupport checks if Neo4j vector index is available and enabled.
func (s *Neo4jStore) HasVectorSupport() bool {
	return s.useVectorIndex && s.vectorIndexCreated
}

// Helper functions

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(int64); ok {
		return int(v)
	}
	return 0
}

func getFloat(m map[string]any, key string, defaultVal float64) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return defaultVal
}

func cosineSimilarity32(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
