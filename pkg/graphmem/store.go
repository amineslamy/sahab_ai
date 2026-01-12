package graphmem

import (
	"sync"
	"time"
)

// Store defines the interface for memory storage backends.
type Store interface {
	SaveMemory(memory *Memory) error
	LoadMemory(memoryID, userID string) (*Memory, error)
	DeleteMemory(memoryID string) error
	ClearMemory(memoryID string) error
	ListMemories(userID string) ([]string, error)
	Close() error
	HealthCheck() bool
}

// Cache defines the interface for caching.
type Cache interface {
	Get(key string) (any, bool)
	Set(key string, value any) error
	Invalidate(memoryID, userID string) error
	Close() error
}

// InMemoryStore implements Store for in-memory storage.
type InMemoryStore struct {
	mu       sync.RWMutex
	memories map[string]*Memory
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		memories: make(map[string]*Memory),
	}
}

// SaveMemory saves a memory to storage.
func (s *InMemoryStore) SaveMemory(memory *Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy to prevent external modifications
	s.memories[memory.ID] = memory.Clone()
	return nil
}

// LoadMemory loads a memory from storage.
func (s *InMemoryStore) LoadMemory(memoryID, userID string) (*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	memory, found := s.memories[memoryID]
	if !found {
		return nil, nil
	}

	// Return deep copy filtered by userID
	loaded := memory.Clone()

	// Filter nodes by userID if specified
	if userID != "" {
		filtered := make(map[string]*MemoryNode)
		for id, node := range loaded.Nodes {
			if node.UserID == "" || node.UserID == userID {
				filtered[id] = node
			}
		}
		loaded.Nodes = filtered
	}

	return loaded, nil
}

// DeleteMemory deletes a memory from storage.
func (s *InMemoryStore) DeleteMemory(memoryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.memories, memoryID)
	return nil
}

// ClearMemory clears all data in a memory.
func (s *InMemoryStore) ClearMemory(memoryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.memories, memoryID)
	return nil
}

// ListMemories lists all memory IDs for a user.
func (s *InMemoryStore) ListMemories(userID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.memories))
	for k := range s.memories {
		keys = append(keys, k)
	}
	return keys, nil
}

// Close closes the store.
func (s *InMemoryStore) Close() error {
	return nil
}

// HealthCheck checks if the store is healthy.
func (s *InMemoryStore) HealthCheck() bool {
	return true
}

// InMemoryCache implements Cache for in-memory caching.
type InMemoryCache struct {
	mu    sync.RWMutex
	cache map[string]cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// NewInMemoryCache creates a new InMemoryCache.
func NewInMemoryCache(ttlSeconds int) *InMemoryCache {
	return &InMemoryCache{
		cache: make(map[string]cacheEntry),
		ttl:   time.Duration(ttlSeconds) * time.Second,
	}
}

// Get retrieves a value from the cache.
func (c *InMemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.cache[key]
	if !found {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.value, true
}

// Set stores a value in the cache.
func (c *InMemoryCache) Set(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	return nil
}

// Invalidate invalidates cache entries for a memory.
func (c *InMemoryCache) Invalidate(memoryID, userID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple implementation: clear everything
	c.cache = make(map[string]cacheEntry)
	return nil
}

// Close closes the cache.
func (c *InMemoryCache) Close() error {
	return nil
}

// SimpleEmbeddingCache provides embedding-specific caching.
type SimpleEmbeddingCache struct {
	mu    sync.RWMutex
	cache map[string][]float32
}

// NewSimpleEmbeddingCache creates a new embedding cache.
func NewSimpleEmbeddingCache() *SimpleEmbeddingCache {
	return &SimpleEmbeddingCache{
		cache: make(map[string][]float32),
	}
}

// Get retrieves an embedding from cache.
func (c *SimpleEmbeddingCache) Get(key string) ([]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	emb, found := c.cache[key]
	return emb, found
}

// Set stores an embedding in cache.
func (c *SimpleEmbeddingCache) Set(key string, embedding []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = embedding
}

