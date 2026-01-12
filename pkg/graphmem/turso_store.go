package graphmem

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// TursoStore implements Store for Turso/libSQL SQLite storage.
type TursoStore struct {
	db                  *sql.DB
	dbPath              string
	tursoURL            string
	tursoAuthToken      string
	syncMode            string
	embeddingDimensions int
	nativeVectorSearch  bool
}

// TursoStoreOptions contains options for TursoStore.
type TursoStoreOptions struct {
	DBPath              string
	TursoURL            string
	TursoAuthToken      string
	SyncMode            string // "on_close", "full", "push", "pull", or ""
	EmbeddingDimensions int
}

// NewTursoStore creates a new Turso store.
func NewTursoStore(opts *TursoStoreOptions) (*TursoStore, error) {
	if opts == nil {
		return nil, fmt.Errorf("options required")
	}

	if opts.DBPath == "" {
		opts.DBPath = "graphmem.db"
	}
	if opts.SyncMode == "" {
		opts.SyncMode = "on_close"
	}
	if opts.EmbeddingDimensions <= 0 {
		opts.EmbeddingDimensions = 1536
	}

	store := &TursoStore{
		dbPath:              opts.DBPath,
		tursoURL:            opts.TursoURL,
		tursoAuthToken:      opts.TursoAuthToken,
		syncMode:            opts.SyncMode,
		embeddingDimensions: opts.EmbeddingDimensions,
	}

	// Connect to database
	if err := store.connect(); err != nil {
		return nil, err
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *TursoStore) connect() error {
	var dsn string
	if s.tursoURL != "" && s.tursoAuthToken != "" {
		dsn = fmt.Sprintf("file:%s?_auth_token=%s&_url=%s", s.dbPath, s.tursoAuthToken, s.tursoURL)
	} else {
		dsn = fmt.Sprintf("file:%s", s.dbPath)
	}

	db, err := sql.Open("libsql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	s.db = db

	if s.tursoURL != "" && s.tursoAuthToken != "" {
		log.Printf("TursoStore connected (local: %s, cloud sync available)", s.dbPath)
	} else {
		log.Printf("TursoStore connected (local-only): %s", s.dbPath)
	}

	return nil
}

func (s *TursoStore) initSchema() error {
	// Entities table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS entities (
			id TEXT PRIMARY KEY,
			memory_id TEXT NOT NULL,
			user_id TEXT,
			name TEXT NOT NULL,
			entity_type TEXT,
			description TEXT,
			importance REAL DEFAULT 0.5,
			access_count INTEGER DEFAULT 0,
			accessed_at TEXT,
			created_at TEXT,
			updated_at TEXT,
			embedding BLOB,
			metadata TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create entities table: %w", err)
	}

	// Relationships table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS relationships (
			id TEXT PRIMARY KEY,
			memory_id TEXT NOT NULL,
			user_id TEXT,
			source_id TEXT NOT NULL,
			target_id TEXT NOT NULL,
			relation_type TEXT NOT NULL,
			description TEXT,
			weight REAL DEFAULT 1.0,
			confidence REAL DEFAULT 1.0,
			valid_from TEXT,
			valid_until TEXT,
			created_at TEXT,
			metadata TEXT,
			FOREIGN KEY (source_id) REFERENCES entities(id),
			FOREIGN KEY (target_id) REFERENCES entities(id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create relationships table: %w", err)
	}

	// Clusters table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS clusters (
			id TEXT,
			memory_id TEXT NOT NULL,
			user_id TEXT,
			summary TEXT,
			entity_ids TEXT,
			metadata TEXT,
			PRIMARY KEY (id, memory_id, user_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create clusters table: %w", err)
	}

	// Memory metadata table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			created_at TEXT,
			updated_at TEXT,
			metadata TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create memories table: %w", err)
	}

	// Create indices
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_entities_memory_user ON entities(memory_id, user_id)`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_relationships_memory_user ON relationships(memory_id, user_id)`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_entities_name ON entities(name)`)

	log.Printf("TursoStore schema initialized")
	return nil
}

// SaveMemory saves a memory to Turso database.
func (s *TursoStore) SaveMemory(memory *Memory) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Save memory metadata
	createdAt := now
	if !memory.CreatedAt.IsZero() {
		createdAt = memory.CreatedAt.Format(time.RFC3339)
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO memories (id, user_id, created_at, updated_at, metadata)
		VALUES (?, ?, ?, ?, ?)
	`, memory.ID, "", createdAt, now, "{}")
	if err != nil {
		return fmt.Errorf("failed to save memory metadata: %w", err)
	}

	// Save nodes
	for _, node := range memory.Nodes {
		if err := s.saveNode(memory.ID, node, now); err != nil {
			return err
		}
	}

	// Save edges
	for _, edge := range memory.Edges {
		if err := s.saveEdge(memory.ID, edge, now); err != nil {
			return err
		}
	}

	// Save clusters
	for _, cluster := range memory.Clusters {
		if err := s.saveCluster(memory.ID, cluster); err != nil {
			return err
		}
	}

	log.Printf("Saved memory %s with %d nodes, %d edges", memory.ID, len(memory.Nodes), len(memory.Edges))
	return nil
}

func (s *TursoStore) saveNode(memoryID string, node *MemoryNode, now string) error {
	var accessedAt, createdAt *string
	if !node.AccessedAt.IsZero() {
		v := node.AccessedAt.Format(time.RFC3339)
		accessedAt = &v
	}
	if !node.CreatedAt.IsZero() {
		v := node.CreatedAt.Format(time.RFC3339)
		createdAt = &v
	}

	properties, _ := json.Marshal(node.Properties)

	// Convert embedding to blob
	var embeddingBlob []byte
	if len(node.Embedding) > 0 {
		embeddingBlob = float32SliceToBytes(node.Embedding)
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO entities 
		(id, memory_id, user_id, name, entity_type, description, 
		 importance, access_count, accessed_at, created_at, updated_at,
		 embedding, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, node.ID, memoryID, node.UserID, node.Name, node.EntityType, node.Description,
		node.Importance.Value(), node.AccessCount, accessedAt, createdAt, now,
		embeddingBlob, string(properties))

	return err
}

func (s *TursoStore) saveEdge(memoryID string, edge *MemoryEdge, now string) error {
	var validFrom, validUntil, createdAt *string
	if edge.ValidFrom != nil && !edge.ValidFrom.IsZero() {
		v := edge.ValidFrom.Format(time.RFC3339)
		validFrom = &v
	}
	if edge.ValidUntil != nil && !edge.ValidUntil.IsZero() {
		v := edge.ValidUntil.Format(time.RFC3339)
		validUntil = &v
	}
	if !edge.CreatedAt.IsZero() {
		v := edge.CreatedAt.Format(time.RFC3339)
		createdAt = &v
	}

	properties, _ := json.Marshal(edge.Properties)

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO relationships
		(id, memory_id, user_id, source_id, target_id, relation_type,
		 description, weight, confidence, valid_from, valid_until,
		 created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, edge.ID, memoryID, "", edge.SourceID, edge.TargetID, edge.RelationType,
		edge.Description, edge.Weight, edge.Confidence, validFrom, validUntil,
		createdAt, string(properties))

	return err
}

func (s *TursoStore) saveCluster(memoryID string, cluster *MemoryCluster) error {
	entityIDs, _ := json.Marshal(cluster.Entities)
	metadata, _ := json.Marshal(cluster.Metadata)

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO clusters
		(id, memory_id, user_id, summary, entity_ids, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, cluster.ID, memoryID, "", cluster.Summary, string(entityIDs), string(metadata))

	return err
}

// LoadMemory loads a memory from Turso database.
func (s *TursoStore) LoadMemory(memoryID, userID string) (*Memory, error) {
	// Check if memory exists
	row := s.db.QueryRow("SELECT id FROM memories WHERE id = ?", memoryID)
	var id string
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	memory := NewMemory(memoryID)

	// Load nodes
	nodes, err := s.loadNodes(memoryID, userID)
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		memory.Nodes[node.ID] = node
	}

	// Load edges
	edges, err := s.loadEdges(memoryID, userID)
	if err != nil {
		return nil, err
	}
	for _, edge := range edges {
		memory.Edges[edge.ID] = edge
	}

	// Load clusters
	clusters, err := s.loadClusters(memoryID, userID)
	if err != nil {
		return nil, err
	}
	for _, cluster := range clusters {
		memory.Clusters[cluster.ID] = cluster
	}

	log.Printf("Loaded memory %s: %d nodes, %d edges", memoryID, len(memory.Nodes), len(memory.Edges))
	return memory, nil
}

func (s *TursoStore) loadNodes(memoryID, userID string) ([]*MemoryNode, error) {
	var rows *sql.Rows
	var err error

	if userID != "" {
		rows, err = s.db.Query(`
			SELECT id, memory_id, user_id, name, entity_type, description,
				   importance, access_count, accessed_at, created_at, updated_at,
				   embedding, metadata
			FROM entities 
			WHERE memory_id = ? AND (user_id = ? OR user_id IS NULL OR user_id = '')
		`, memoryID, userID)
	} else {
		rows, err = s.db.Query(`
			SELECT id, memory_id, user_id, name, entity_type, description,
				   importance, access_count, accessed_at, created_at, updated_at,
				   embedding, metadata
			FROM entities 
			WHERE memory_id = ?
		`, memoryID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]*MemoryNode, 0)
	for rows.Next() {
		var (
			id, memID, uID, name, entityType, description string
			importance                                    float64
			accessCount                                   int
			accessedAt, createdAt, updatedAt, metadata    sql.NullString
			embedding                                     []byte
		)

		if err := rows.Scan(&id, &memID, &uID, &name, &entityType, &description,
			&importance, &accessCount, &accessedAt, &createdAt, &updatedAt,
			&embedding, &metadata); err != nil {
			return nil, err
		}

		var properties map[string]any
		if metadata.Valid {
			json.Unmarshal([]byte(metadata.String), &properties)
		}

		var emb []float32
		if len(embedding) > 0 {
			emb = bytesToFloat32Slice(embedding)
		}

		node := &MemoryNode{
			ID:          id,
			Name:        name,
			EntityType:  entityType,
			Description: description,
			Importance:  MemoryImportance(int(importance)),
			AccessCount: accessCount,
			Embedding:   emb,
			Properties:  properties,
			UserID:      uID,
		}

		if accessedAt.Valid {
			t, _ := time.Parse(time.RFC3339, accessedAt.String)
			node.AccessedAt = t
		}
		if createdAt.Valid {
			t, _ := time.Parse(time.RFC3339, createdAt.String)
			node.CreatedAt = t
		}

		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

func (s *TursoStore) loadEdges(memoryID, userID string) ([]*MemoryEdge, error) {
	var rows *sql.Rows
	var err error

	if userID != "" {
		rows, err = s.db.Query(`
			SELECT id, memory_id, user_id, source_id, target_id, relation_type,
				   description, weight, confidence, valid_from, valid_until,
				   created_at, metadata
			FROM relationships 
			WHERE memory_id = ? AND (user_id = ? OR user_id IS NULL OR user_id = '')
		`, memoryID, userID)
	} else {
		rows, err = s.db.Query(`
			SELECT id, memory_id, user_id, source_id, target_id, relation_type,
				   description, weight, confidence, valid_from, valid_until,
				   created_at, metadata
			FROM relationships 
			WHERE memory_id = ?
		`, memoryID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	edges := make([]*MemoryEdge, 0)
	for rows.Next() {
		var (
			id, memID, uID, sourceID, targetID, relationType, description string
			weight, confidence                                            float64
			validFrom, validUntil, createdAt, metadata                    sql.NullString
		)

		if err := rows.Scan(&id, &memID, &uID, &sourceID, &targetID, &relationType,
			&description, &weight, &confidence, &validFrom, &validUntil,
			&createdAt, &metadata); err != nil {
			return nil, err
		}

		var properties map[string]any
		if metadata.Valid {
			json.Unmarshal([]byte(metadata.String), &properties)
		}

		edge := &MemoryEdge{
			ID:           id,
			SourceID:     sourceID,
			TargetID:     targetID,
			RelationType: relationType,
			Description:  description,
			Weight:       weight,
			Confidence:   confidence,
			Properties:   properties,
		}

		if validFrom.Valid {
			t, _ := time.Parse(time.RFC3339, validFrom.String)
			edge.ValidFrom = &t
		}
		if validUntil.Valid {
			t, _ := time.Parse(time.RFC3339, validUntil.String)
			edge.ValidUntil = &t
		}
		if createdAt.Valid {
			t, _ := time.Parse(time.RFC3339, createdAt.String)
			edge.CreatedAt = t
		}

		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

func (s *TursoStore) loadClusters(memoryID, userID string) ([]*MemoryCluster, error) {
	rows, err := s.db.Query(`
		SELECT id, memory_id, user_id, summary, entity_ids, metadata
		FROM clusters 
		WHERE memory_id = ?
	`, memoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clusters := make([]*MemoryCluster, 0)
	for rows.Next() {
		var (
			id, memID, uID, summary, entityIDs, metadata string
		)

		if err := rows.Scan(&id, &memID, &uID, &summary, &entityIDs, &metadata); err != nil {
			return nil, err
		}

		var entities []string
		json.Unmarshal([]byte(entityIDs), &entities)

		cluster := &MemoryCluster{
			ID:       id,
			Summary:  summary,
			Entities: entities,
			MemoryID: memoryID,
		}

		clusters = append(clusters, cluster)
	}

	return clusters, rows.Err()
}

// VectorSearch performs vector similarity search.
func (s *TursoStore) VectorSearch(memoryID string, queryEmbedding []float32, topK int, userID string) ([]*MemoryNodeWithScore, error) {
	// Try native vector search if enabled
	if s.nativeVectorSearch {
		results, err := s.nativeVectorSearchImpl(memoryID, queryEmbedding, topK, userID)
		if err == nil {
			return results, nil
		}
		// Fall back to brute-force if native fails
		log.Printf("Native vector search failed, falling back to brute-force: %v", err)
	}

	// Load all nodes with embeddings
	nodes, err := s.loadNodes(memoryID, userID)
	if err != nil {
		return nil, err
	}

	// Calculate similarities (brute-force fallback)
	type nodeScore struct {
		node  *MemoryNode
		score float64
	}
	similarities := make([]nodeScore, 0)

	for _, node := range nodes {
		if len(node.Embedding) == 0 {
			continue
		}

		sim := cosineSimilarityFloat32(queryEmbedding, node.Embedding)
		similarities = append(similarities, nodeScore{node: node, score: sim})
	}

	// Sort by similarity
	for i := 0; i < len(similarities); i++ {
		for j := i + 1; j < len(similarities); j++ {
			if similarities[j].score > similarities[i].score {
				similarities[i], similarities[j] = similarities[j], similarities[i]
			}
		}
	}

	// Take top K
	if len(similarities) > topK {
		similarities = similarities[:topK]
	}

	results := make([]*MemoryNodeWithScore, len(similarities))
	for i, ns := range similarities {
		results[i] = &MemoryNodeWithScore{Node: ns.node, Score: ns.score}
	}

	return results, nil
}

// nativeVectorSearchImpl performs vector search using libsql_vector_idx.
func (s *TursoStore) nativeVectorSearchImpl(memoryID string, queryEmbedding []float32, topK int, userID string) ([]*MemoryNodeWithScore, error) {
	// Convert embedding to blob
	embeddingBlob := embeddingToBlob(queryEmbedding)

	query := `
		SELECT e.id, e.name, e.entity_type, e.description, e.importance,
			   e.memory_id, e.user_id, e.embedding,
			   vector_distance_cos(e.embedding, ?) AS distance
		FROM entities e
		WHERE e.memory_id = ?
		AND e.embedding IS NOT NULL
	`
	args := []any{embeddingBlob, memoryID}

	if userID != "" {
		query += " AND (e.user_id = ? OR e.user_id IS NULL OR e.user_id = '')"
		args = append(args, userID)
	}

	query += " ORDER BY distance ASC LIMIT ?"
	args = append(args, topK)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*MemoryNodeWithScore, 0)
	for rows.Next() {
		var (
			id, name, entityType, description, memID string
			userIDVal                                sql.NullString
			importance                               float64
			embeddingBlob                            []byte
			distance                                 float64
		)

		if err := rows.Scan(&id, &name, &entityType, &description, &importance,
			&memID, &userIDVal, &embeddingBlob, &distance); err != nil {
			return nil, err
		}

		node := &MemoryNode{
			ID:          id,
			Name:        name,
			EntityType:  entityType,
			Description: description,
			Importance:  MemoryImportance(importance * 10), // Scale back
			MemoryID:    memID,
			Embedding:   blobToEmbedding(embeddingBlob),
		}

		// Convert distance to similarity (1 - distance for cosine)
		similarity := 1.0 - distance

		results = append(results, &MemoryNodeWithScore{Node: node, Score: similarity})
	}

	return results, rows.Err()
}

// embeddingToBlob converts a float32 slice to a binary blob.
func embeddingToBlob(embedding []float32) []byte {
	blob := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		bits := math.Float32bits(v)
		binary.LittleEndian.PutUint32(blob[i*4:], bits)
	}
	return blob
}

// blobToEmbedding converts a binary blob to a float32 slice.
func blobToEmbedding(blob []byte) []float32 {
	if len(blob)%4 != 0 {
		return nil
	}
	embedding := make([]float32, len(blob)/4)
	for i := range embedding {
		bits := binary.LittleEndian.Uint32(blob[i*4:])
		embedding[i] = math.Float32frombits(bits)
	}
	return embedding
}

// QueryEdgesAtTime queries edges valid at a specific time.
func (s *TursoStore) QueryEdgesAtTime(memoryID string, queryTime time.Time, relationType, userID string) ([]*MemoryEdge, error) {
	queryTimeStr := queryTime.Format(time.RFC3339)

	query := `
		SELECT id, memory_id, user_id, source_id, target_id, relation_type,
			   description, weight, confidence, valid_from, valid_until,
			   created_at, metadata
		FROM relationships
		WHERE memory_id = ?
		AND (valid_from IS NULL OR valid_from <= ?)
		AND (valid_until IS NULL OR valid_until > ?)
	`
	args := []any{memoryID, queryTimeStr, queryTimeStr}

	if relationType != "" {
		query += " AND relation_type = ?"
		args = append(args, relationType)
	}

	if userID != "" {
		query += " AND (user_id = ? OR user_id IS NULL OR user_id = '')"
		args = append(args, userID)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	edges := make([]*MemoryEdge, 0)
	for rows.Next() {
		var (
			id, memID, uID, sourceID, targetID, rt, description string
			weight, confidence                                  float64
			validFrom, validUntil, createdAt, metadata          sql.NullString
		)

		if err := rows.Scan(&id, &memID, &uID, &sourceID, &targetID, &rt,
			&description, &weight, &confidence, &validFrom, &validUntil,
			&createdAt, &metadata); err != nil {
			return nil, err
		}

		edge := &MemoryEdge{
			ID:           id,
			SourceID:     sourceID,
			TargetID:     targetID,
			RelationType: rt,
			Description:  description,
			Weight:       weight,
			Confidence:   confidence,
		}

		if validFrom.Valid {
			t, _ := time.Parse(time.RFC3339, validFrom.String)
			edge.ValidFrom = &t
		}
		if validUntil.Valid {
			t, _ := time.Parse(time.RFC3339, validUntil.String)
			edge.ValidUntil = &t
		}

		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

// SupersedeRelationship marks a relationship as superseded.
func (s *TursoStore) SupersedeRelationship(memoryID, edgeID string, endTime *time.Time) (bool, error) {
	effectiveTime := time.Now().UTC()
	if endTime != nil {
		effectiveTime = *endTime
	}

	result, err := s.db.Exec(`
		UPDATE relationships 
		SET valid_until = ?, state = 'ARCHIVED'
		WHERE id = ? AND memory_id = ?
	`, effectiveTime.Format(time.RFC3339), edgeID, memoryID)
	if err != nil {
		return false, err
	}

	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

// DeleteMemory deletes a memory from the database.
func (s *TursoStore) DeleteMemory(memoryID string) error {
	s.db.Exec("DELETE FROM entities WHERE memory_id = ?", memoryID)
	s.db.Exec("DELETE FROM relationships WHERE memory_id = ?", memoryID)
	s.db.Exec("DELETE FROM clusters WHERE memory_id = ?", memoryID)
	s.db.Exec("DELETE FROM memories WHERE id = ?", memoryID)
	log.Printf("Deleted memory %s", memoryID)
	return nil
}

// ClearMemory clears all data in a memory.
func (s *TursoStore) ClearMemory(memoryID string) error {
	return s.DeleteMemory(memoryID)
}

// ListMemories lists all memory IDs.
func (s *TursoStore) ListMemories(userID string) ([]string, error) {
	rows, err := s.db.Query("SELECT id FROM memories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// Close closes the database connection.
func (s *TursoStore) Close() error {
	return s.db.Close()
}

// HealthCheck checks if the store is healthy.
func (s *TursoStore) HealthCheck() bool {
	err := s.db.Ping()
	return err == nil
}

// TursoCache implements a SQLite-based cache using Turso.
type TursoCache struct {
	db  *sql.DB
	ttl time.Duration
}

// NewTursoCache creates a new Turso cache.
func NewTursoCache(dbPath string, ttl time.Duration) (*TursoCache, error) {
	db, err := sql.Open("libsql", fmt.Sprintf("file:%s", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	cache := &TursoCache{db: db, ttl: ttl}

	// Initialize schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cache (
			key TEXT PRIMARY KEY,
			value TEXT,
			expires_at TEXT
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache table: %w", err)
	}

	db.Exec("CREATE INDEX IF NOT EXISTS idx_cache_expires ON cache(expires_at)")

	return cache, nil
}

// Get retrieves a value from cache.
func (c *TursoCache) Get(key string) (any, bool) {
	now := time.Now().UTC().Format(time.RFC3339)

	row := c.db.QueryRow(`
		SELECT value FROM cache 
		WHERE key = ? AND (expires_at IS NULL OR expires_at > ?)
	`, key, now)

	var value string
	if err := row.Scan(&value); err != nil {
		return nil, false
	}

	var result any
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, false
	}

	return result, true
}

// Set stores a value in cache.
func (c *TursoCache) Set(key string, value any) error {
	expiresAt := time.Now().UTC().Add(c.ttl).Format(time.RFC3339)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	_, err = c.db.Exec(`
		INSERT OR REPLACE INTO cache (key, value, expires_at)
		VALUES (?, ?, ?)
	`, key, string(data), expiresAt)

	return err
}

// Invalidate invalidates cache entries.
func (c *TursoCache) Invalidate(memoryID, userID string) error {
	pattern := "%" + userID + "%" + memoryID + "%"
	_, err := c.db.Exec("DELETE FROM cache WHERE key LIKE ?", pattern)
	return err
}

// Close closes the cache.
func (c *TursoCache) Close() error {
	return c.db.Close()
}

// Helper functions

func float32SliceToBytes(floats []float32) []byte {
	bytes := make([]byte, len(floats)*4)
	for i, f := range floats {
		bits := math.Float32bits(f)
		binary.LittleEndian.PutUint32(bytes[i*4:], bits)
	}
	return bytes
}

func bytesToFloat32Slice(bytes []byte) []float32 {
	numFloats := len(bytes) / 4
	floats := make([]float32, numFloats)
	for i := 0; i < numFloats; i++ {
		bits := binary.LittleEndian.Uint32(bytes[i*4:])
		floats[i] = math.Float32frombits(bits)
	}
	return floats
}

func cosineSimilarityFloat32(a, b []float32) float64 {
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
