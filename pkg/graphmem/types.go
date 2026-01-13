// Package graphmem provides a knowledge graph-based memory system.
package graphmem

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GenerateID generates a new unique ID.
func GenerateID() string {
	return uuid.New().String()
}

// MemoryImportance levels for prioritization and decay.
type MemoryImportance int

const (
	ImportanceEphemeral MemoryImportance = 0  // Immediate decay candidate
	ImportanceVeryLow   MemoryImportance = 1  // Fast decay, low priority
	ImportanceLow       MemoryImportance = 3  // Faster decay, below average priority
	ImportanceMedium    MemoryImportance = 5  // Normal decay rate, average priority
	ImportanceHigh      MemoryImportance = 6  // Slow decay, above average priority
	ImportanceVeryHigh  MemoryImportance = 8  // Extremely slow decay, high priority
	ImportanceCritical  MemoryImportance = 10 // Never decay, always retrieve first
)

// ImportanceFromScore converts a float score (0-10) to a MemoryImportance level.
func ImportanceFromScore(score float64) MemoryImportance {
	switch {
	case score >= 9:
		return ImportanceCritical
	case score >= 7:
		return ImportanceVeryHigh
	case score >= 5.5:
		return ImportanceHigh
	case score >= 4:
		return ImportanceMedium
	case score >= 2:
		return ImportanceLow
	case score >= 0.5:
		return ImportanceVeryLow
	default:
		return ImportanceEphemeral
	}
}

// Value returns the integer value of MemoryImportance.
func (mi MemoryImportance) Value() int {
	return int(mi)
}

// String returns the string representation of MemoryImportance.
func (mi MemoryImportance) String() string {
	switch mi {
	case ImportanceEphemeral:
		return "ephemeral"
	case ImportanceVeryLow:
		return "very_low"
	case ImportanceLow:
		return "low"
	case ImportanceMedium:
		return "medium"
	case ImportanceHigh:
		return "high"
	case ImportanceVeryHigh:
		return "very_high"
	case ImportanceCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// MemoryState defines memory lifecycle states.
type MemoryState string

const (
	StateActive        MemoryState = "ACTIVE"
	StateConsolidating MemoryState = "CONSOLIDATING"
	StateDecaying      MemoryState = "DECAYING"
	StateArchived      MemoryState = "ARCHIVED"
	StateDeleted       MemoryState = "DELETED"
)

// String returns the string representation of MemoryState.
func (ms MemoryState) String() string {
	return string(ms)
}

// EvolutionType defines types of memory evolution events.
type EvolutionType string

const (
	EvolutionConsolidation EvolutionType = "CONSOLIDATION"
	EvolutionReinforcement EvolutionType = "REINFORCEMENT"
	EvolutionDecay         EvolutionType = "DECAY"
	EvolutionRehydration   EvolutionType = "REHYDRATION"
	EvolutionCorrection    EvolutionType = "CORRECTION"
	EvolutionSynthesis     EvolutionType = "SYNTHESIS"
	EvolutionPruning       EvolutionType = "PRUNING"
)

// String returns the string representation of EvolutionType.
func (et EvolutionType) String() string {
	return string(et)
}

// MemoryNode represents an entity node in the memory graph.
type MemoryNode struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	EntityType    string           `json:"entity_type"`
	Properties    map[string]any   `json:"properties"`
	Description   string           `json:"description,omitempty"`
	CanonicalName string           `json:"canonical_name,omitempty"`
	Aliases       map[string]bool  `json:"aliases"`
	Embedding     []float32        `json:"embedding,omitempty"`
	Importance    MemoryImportance `json:"importance"`
	State         MemoryState      `json:"state"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	AccessedAt    time.Time        `json:"accessed_at"`
	AccessCount   int              `json:"access_count"`
	UserID        string           `json:"user_id,omitempty"`
	MemoryID      string           `json:"memory_id,omitempty"`

	mu sync.RWMutex
}

// NewMemoryNode creates a new MemoryNode with default values.
func NewMemoryNode(name, entityType, userID, memoryID string) *MemoryNode {
	node := &MemoryNode{
		Name:        name,
		EntityType:  entityType,
		Properties:  make(map[string]any),
		Aliases:     make(map[string]bool),
		Importance:  ImportanceMedium,
		State:       StateActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		AccessedAt:  time.Now().UTC(),
		AccessCount: 0,
		UserID:      userID,
		MemoryID:    memoryID,
	}
	node.ID = node.GenerateID()
	node.CanonicalName = name
	node.Aliases[name] = true
	return node
}

// GenerateID creates a deterministic ID for the node.
func (n *MemoryNode) GenerateID() string {
	content := fmt.Sprintf("%s:%s:%s:%s", n.Name, n.EntityType, n.UserID, n.MemoryID)
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:16])
}

// AddAlias adds an alias to the node.
func (n *MemoryNode) AddAlias(alias string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Aliases[alias] = true
	n.UpdatedAt = time.Now().UTC()
}

// GetAliases returns all aliases as a slice.
func (n *MemoryNode) GetAliases() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	aliases := make([]string, 0, len(n.Aliases))
	for alias := range n.Aliases {
		aliases = append(aliases, alias)
	}
	return aliases
}

// RecordAccess updates the accessed_at time and increments access_count.
func (n *MemoryNode) RecordAccess() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.AccessedAt = time.Now().UTC()
	n.AccessCount++
}

// Clone creates a deep copy of the node.
func (n *MemoryNode) Clone() *MemoryNode {
	n.mu.RLock()
	defer n.mu.RUnlock()

	clone := &MemoryNode{
		ID:            n.ID,
		Name:          n.Name,
		EntityType:    n.EntityType,
		Description:   n.Description,
		CanonicalName: n.CanonicalName,
		Importance:    n.Importance,
		State:         n.State,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
		AccessedAt:    n.AccessedAt,
		AccessCount:   n.AccessCount,
		UserID:        n.UserID,
		MemoryID:      n.MemoryID,
	}

	clone.Properties = make(map[string]any)
	for k, v := range n.Properties {
		clone.Properties[k] = v
	}

	clone.Aliases = make(map[string]bool)
	for k, v := range n.Aliases {
		clone.Aliases[k] = v
	}

	if n.Embedding != nil {
		clone.Embedding = make([]float32, len(n.Embedding))
		copy(clone.Embedding, n.Embedding)
	}

	return clone
}

// MemoryEdge represents a relationship edge in the memory graph.
type MemoryEdge struct {
	ID           string           `json:"id"`
	SourceID     string           `json:"source_id"`
	TargetID     string           `json:"target_id"`
	RelationType string           `json:"relation_type"`
	Properties   map[string]any   `json:"properties"`
	Description  string           `json:"description,omitempty"`
	Weight       float64          `json:"weight"`
	Confidence   float64          `json:"confidence"`
	Importance   MemoryImportance `json:"importance"`
	State        MemoryState      `json:"state"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	AccessedAt   time.Time        `json:"accessed_at"`
	AccessCount  int              `json:"access_count"`
	MemoryID     string           `json:"memory_id,omitempty"`
	ValidFrom    *time.Time       `json:"valid_from,omitempty"`
	ValidUntil   *time.Time       `json:"valid_until,omitempty"`
}

// NewMemoryEdge creates a new MemoryEdge with default values.
func NewMemoryEdge(sourceID, targetID, relationType, memoryID string) *MemoryEdge {
	edge := &MemoryEdge{
		SourceID:     sourceID,
		TargetID:     targetID,
		RelationType: relationType,
		Properties:   make(map[string]any),
		Weight:       1.0,
		Confidence:   1.0,
		Importance:   ImportanceMedium,
		State:        StateActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		AccessedAt:   time.Now().UTC(),
		AccessCount:  0,
		MemoryID:     memoryID,
	}
	edge.ID = edge.GenerateID()
	return edge
}

// GenerateID creates a deterministic ID for the edge.
func (e *MemoryEdge) GenerateID() string {
	content := fmt.Sprintf("%s:%s:%s:%s", e.SourceID, e.RelationType, e.TargetID, e.MemoryID)
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:16])
}

// IsValidAt checks if this relationship is valid at a given timestamp.
func (e *MemoryEdge) IsValidAt(timestamp *time.Time) bool {
	if timestamp == nil {
		now := time.Now().UTC()
		timestamp = &now
	}

	if e.ValidFrom != nil && timestamp.Before(*e.ValidFrom) {
		return false
	}
	if e.ValidUntil != nil && timestamp.After(*e.ValidUntil) {
		return false
	}
	return true
}

// Clone creates a deep copy of the edge.
func (e *MemoryEdge) Clone() *MemoryEdge {
	clone := &MemoryEdge{
		ID:           e.ID,
		SourceID:     e.SourceID,
		TargetID:     e.TargetID,
		RelationType: e.RelationType,
		Description:  e.Description,
		Weight:       e.Weight,
		Confidence:   e.Confidence,
		Importance:   e.Importance,
		State:        e.State,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
		AccessedAt:   e.AccessedAt,
		AccessCount:  e.AccessCount,
		MemoryID:     e.MemoryID,
	}

	clone.Properties = make(map[string]any)
	for k, v := range e.Properties {
		clone.Properties[k] = v
	}

	if e.ValidFrom != nil {
		t := *e.ValidFrom
		clone.ValidFrom = &t
	}
	if e.ValidUntil != nil {
		t := *e.ValidUntil
		clone.ValidUntil = &t
	}

	return clone
}

// MemoryCluster represents a cluster of related memories.
type MemoryCluster struct {
	ID             string           `json:"id"`
	Summary        string           `json:"summary"`
	Entities       []string         `json:"entities"`
	Edges          []string         `json:"edges"`
	Importance     MemoryImportance `json:"importance"`
	CoherenceScore float64          `json:"coherence_score"`
	Density        float64          `json:"density"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	MemoryID       string           `json:"memory_id,omitempty"`
	Metadata       map[string]any   `json:"metadata"`
}

// Memory represents a complete memory unit.
type Memory struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name,omitempty"`
	Description string                    `json:"description,omitempty"`
	Nodes       map[string]*MemoryNode    `json:"nodes"`
	Edges       map[string]*MemoryEdge    `json:"edges"`
	Clusters    map[string]*MemoryCluster `json:"clusters"`
	Metadata    map[string]any            `json:"metadata"`
	Importance  MemoryImportance          `json:"importance"`
	State       MemoryState               `json:"state"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	Version     int                       `json:"version"`
	ParentID    string                    `json:"parent_id,omitempty"`

	mu sync.RWMutex
}

// NewMemory creates a new Memory instance.
func NewMemory(id string) *Memory {
	if id == "" {
		id = uuid.New().String()
	}
	return &Memory{
		ID:         id,
		Nodes:      make(map[string]*MemoryNode),
		Edges:      make(map[string]*MemoryEdge),
		Clusters:   make(map[string]*MemoryCluster),
		Metadata:   make(map[string]any),
		Importance: ImportanceMedium,
		State:      StateActive,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
		Version:    1,
	}
}

// AddNode adds a node to the memory.
func (m *Memory) AddNode(node *MemoryNode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	node.MemoryID = m.ID
	m.Nodes[node.ID] = node
	m.UpdatedAt = time.Now().UTC()
}

// AddEdge adds an edge to the memory.
func (m *Memory) AddEdge(edge *MemoryEdge) {
	m.mu.Lock()
	defer m.mu.Unlock()
	edge.MemoryID = m.ID
	m.Edges[edge.ID] = edge
	m.UpdatedAt = time.Now().UTC()
}

// AddCluster adds a cluster to the memory.
func (m *Memory) AddCluster(cluster *MemoryCluster) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cluster.MemoryID = m.ID
	m.Clusters[cluster.ID] = cluster
	m.UpdatedAt = time.Now().UTC()
}

// GetNode retrieves a node by ID.
func (m *Memory) GetNode(nodeID string) *MemoryNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Nodes[nodeID]
}

// GetEdge retrieves an edge by ID.
func (m *Memory) GetEdge(edgeID string) *MemoryEdge {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Edges[edgeID]
}

// GetCluster retrieves a cluster by ID.
func (m *Memory) GetCluster(clusterID string) *MemoryCluster {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Clusters[clusterID]
}

// RemoveNode removes a node and its associated edges from the memory.
// Returns true if the node was found and removed.
func (m *Memory) RemoveNode(nodeID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Nodes[nodeID]; !exists {
		return false
	}

	// Remove all edges connected to this node
	for edgeID, edge := range m.Edges {
		if edge.SourceID == nodeID || edge.TargetID == nodeID {
			delete(m.Edges, edgeID)
		}
	}

	// Remove the node
	delete(m.Nodes, nodeID)
	m.UpdatedAt = time.Now().UTC()
	return true
}

// RemoveEdge removes an edge from the memory.
// Returns true if the edge was found and removed.
func (m *Memory) RemoveEdge(edgeID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Edges[edgeID]; !exists {
		return false
	}

	delete(m.Edges, edgeID)
	m.UpdatedAt = time.Now().UTC()
	return true
}

// GetAllNodes returns all nodes.
func (m *Memory) GetAllNodes() []*MemoryNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	nodes := make([]*MemoryNode, 0, len(m.Nodes))
	for _, n := range m.Nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// GetAllEdges returns all edges.
func (m *Memory) GetAllEdges() []*MemoryEdge {
	m.mu.RLock()
	defer m.mu.RUnlock()
	edges := make([]*MemoryEdge, 0, len(m.Edges))
	for _, e := range m.Edges {
		edges = append(edges, e)
	}
	return edges
}

// GetAllClusters returns all clusters in the memory.
func (m *Memory) GetAllClusters() []*MemoryCluster {
	m.mu.RLock()
	defer m.mu.RUnlock()
	clusters := make([]*MemoryCluster, 0, len(m.Clusters))
	for _, c := range m.Clusters {
		clusters = append(clusters, c)
	}
	return clusters
}

// NodeCount returns the number of nodes.
func (m *Memory) NodeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.Nodes)
}

// EdgeCount returns the number of edges.
func (m *Memory) EdgeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.Edges)
}

// ClusterCount returns the number of clusters.
func (m *Memory) ClusterCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.Clusters)
}

// Clone creates a deep copy of the memory.
func (m *Memory) Clone() *Memory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clone := &Memory{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Importance:  m.Importance,
		State:       m.State,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		Version:     m.Version,
		ParentID:    m.ParentID,
		Nodes:       make(map[string]*MemoryNode),
		Edges:       make(map[string]*MemoryEdge),
		Clusters:    make(map[string]*MemoryCluster),
		Metadata:    make(map[string]any),
	}

	for id, node := range m.Nodes {
		clone.Nodes[id] = node.Clone()
	}

	for id, edge := range m.Edges {
		clone.Edges[id] = edge.Clone()
	}

	for id, cluster := range m.Clusters {
		clonedCluster := *cluster
		clone.Clusters[id] = &clonedCluster
	}

	for k, v := range m.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// MemoryQuery represents a query against the memory system.
type MemoryQuery struct {
	Query           string           `json:"query"`
	MemoryID        string           `json:"memory_id,omitempty"`
	Mode            string           `json:"mode"`
	TopK            int              `json:"top_k"`
	MinSimilarity   float64          `json:"min_similarity"`
	MinImportance   MemoryImportance `json:"min_importance"`
	IncludeClusters bool             `json:"include_clusters"`
	IncludeContext  bool             `json:"include_context"`
	Filters         map[string]any   `json:"filters"`
	Metadata        map[string]any   `json:"metadata"`
}

// NewMemoryQuery creates a new MemoryQuery with defaults.
func NewMemoryQuery(query string) *MemoryQuery {
	return &MemoryQuery{
		Query:           query,
		Mode:            "hybrid",
		TopK:            10,
		MinSimilarity:   0.5,
		MinImportance:   ImportanceVeryLow,
		IncludeClusters: true,
		IncludeContext:  true,
		Filters:         make(map[string]any),
		Metadata:        make(map[string]any),
	}
}

// MemoryResponse is the response from a memory query.
type MemoryResponse struct {
	Query      string           `json:"query"`
	Answer     string           `json:"answer"`
	Confidence float64          `json:"confidence"`
	Nodes      []*MemoryNode    `json:"nodes"`
	Edges      []*MemoryEdge    `json:"edges"`
	Clusters   []*MemoryCluster `json:"clusters"`
	Context    string           `json:"context"`
	Sources    []string         `json:"sources"`
	Metadata   map[string]any   `json:"metadata"`
	LatencyMS  float64          `json:"latency_ms"`
	Timestamp  time.Time        `json:"timestamp"`
}

// HasResults checks if the response contains any results.
func (mr *MemoryResponse) HasResults() bool {
	return len(mr.Nodes) > 0 || len(mr.Edges) > 0 || len(mr.Clusters) > 0
}

// EvolutionEvent records a memory evolution event.
type EvolutionEvent struct {
	ID            string         `json:"id"`
	EvolutionType EvolutionType  `json:"evolution_type"`
	MemoryID      string         `json:"memory_id"`
	AffectedNodes []string       `json:"affected_nodes"`
	AffectedEdges []string       `json:"affected_edges"`
	BeforeState   map[string]any `json:"before_state,omitempty"`
	AfterState    map[string]any `json:"after_state,omitempty"`
	Reason        string         `json:"reason"`
	Metadata      map[string]any `json:"metadata"`
	Timestamp     time.Time      `json:"timestamp"`
	Success       bool           `json:"success"`
}

// NewEvolutionEvent creates a new EvolutionEvent.
func NewEvolutionEvent(eventType EvolutionType, memoryID string) *EvolutionEvent {
	return &EvolutionEvent{
		ID:            uuid.New().String(),
		EvolutionType: eventType,
		MemoryID:      memoryID,
		AffectedNodes: []string{},
		AffectedEdges: []string{},
		BeforeState:   make(map[string]any),
		AfterState:    make(map[string]any),
		Metadata:      make(map[string]any),
		Timestamp:     time.Now().UTC(),
		Success:       true,
	}
}

// IngestResult is the result of an ingestion operation.
type IngestResult struct {
	Success        bool    `json:"success"`
	MemoryID       string  `json:"memory_id"`
	Entities       int     `json:"entities"`
	Relationships  int     `json:"relationships"`
	Clusters       int     `json:"clusters"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`
}
