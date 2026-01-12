package graphmem

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements Cache for Redis caching.
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// RedisCacheOptions contains options for RedisCache.
type RedisCacheOptions struct {
	URL    string
	TTL    time.Duration
	Prefix string
}

// NewRedisCache creates a new Redis cache.
func NewRedisCache(opts *RedisCacheOptions) (*RedisCache, error) {
	if opts == nil {
		return nil, fmt.Errorf("options required")
	}

	opt, err := redis.ParseURL(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	ttl := opts.TTL
	if ttl == 0 {
		ttl = time.Hour
	}

	prefix := opts.Prefix
	if prefix == "" {
		prefix = "graphmem"
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}, nil
}

// buildKey builds a cache key from parts.
func (c *RedisCache) buildKey(parts ...string) string {
	return c.prefix + ":" + strings.Join(parts, ":")
}

// Get retrieves a value from cache.
func (c *RedisCache) Get(key string) (any, bool) {
	ctx := context.Background()

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Redis get failed: %v", err)
		}
		return nil, false
	}

	var result any
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, false
	}

	return result, true
}

// Set stores a value in cache.
func (c *RedisCache) Set(key string, value any) error {
	return c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in cache with custom TTL.
func (c *RedisCache) SetWithTTL(key string, value any, ttl time.Duration) error {
	ctx := context.Background()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	if err := c.client.SetEx(ctx, key, string(data), ttl).Err(); err != nil {
		log.Printf("Redis set failed: %v", err)
		return err
	}

	return nil
}

// Delete deletes a value from cache.
func (c *RedisCache) Delete(key string) error {
	ctx := context.Background()
	return c.client.Del(ctx, key).Err()
}

// Invalidate invalidates all cache entries for a memory.
func (c *RedisCache) Invalidate(memoryID, userID string) error {
	ctx := context.Background()

	patterns := []string{
		c.buildKey("search", userID, memoryID, "*"),
		c.buildKey("query", userID, memoryID, "*"),
		c.buildKey("community", userID, memoryID, "*"),
		c.buildKey("state", userID, memoryID),
	}

	totalDeleted := 0
	for _, pattern := range patterns {
		iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
		keys := make([]string, 0)
		for iter.Next(ctx) {
			keys = append(keys, iter.Val())
		}
		if err := iter.Err(); err != nil {
			log.Printf("Redis scan failed: %v", err)
			continue
		}
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				log.Printf("Redis delete failed: %v", err)
				continue
			}
			totalDeleted += len(keys)
		}
	}

	log.Printf("Invalidated %d cache entries for user=%s, memory=%s", totalDeleted, userID, memoryID)
	return nil
}

// CacheMemoryState caches memory state.
func (c *RedisCache) CacheMemoryState(memoryID string, state map[string]any, userID string) bool {
	key := c.buildKey("state", userID, memoryID)
	return c.Set(key, state) == nil
}

// GetMemoryState gets cached memory state.
func (c *RedisCache) GetMemoryState(memoryID, userID string) (map[string]any, bool) {
	key := c.buildKey("state", userID, memoryID)
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	if m, ok := val.(map[string]any); ok {
		return m, true
	}
	return nil, false
}

// CacheQueryResult caches a query result.
func (c *RedisCache) CacheQueryResult(memoryID, queryHash string, result map[string]any, userID string, ttl time.Duration) bool {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	key := c.buildKey("query", userID, memoryID, queryHash)
	return c.SetWithTTL(key, result, ttl) == nil
}

// GetQueryResult gets cached query result.
func (c *RedisCache) GetQueryResult(memoryID, queryHash, userID string) (map[string]any, bool) {
	key := c.buildKey("query", userID, memoryID, queryHash)
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	if m, ok := val.(map[string]any); ok {
		return m, true
	}
	return nil, false
}

// CacheSearchResult caches semantic search results.
func (c *RedisCache) CacheSearchResult(memoryID, queryHash string, results []any, userID string, ttl time.Duration) bool {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	key := c.buildKey("search", userID, memoryID, queryHash)
	return c.SetWithTTL(key, results, ttl) == nil
}

// GetSearchResult gets cached search results.
func (c *RedisCache) GetSearchResult(memoryID, queryHash, userID string) ([]any, bool) {
	key := c.buildKey("search", userID, memoryID, queryHash)
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	if s, ok := val.([]any); ok {
		return s, true
	}
	return nil, false
}

// CacheCommunityContext caches community context.
func (c *RedisCache) CacheCommunityContext(memoryID string, communityID int, context map[string]any, userID string) bool {
	key := c.buildKey("community", userID, memoryID, fmt.Sprintf("%d", communityID))
	return c.Set(key, context) == nil
}

// GetCommunityContext gets cached community context.
func (c *RedisCache) GetCommunityContext(memoryID string, communityID int, userID string) (map[string]any, bool) {
	key := c.buildKey("community", userID, memoryID, fmt.Sprintf("%d", communityID))
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	if m, ok := val.(map[string]any); ok {
		return m, true
	}
	return nil, false
}

// CacheEmbedding caches an embedding.
func (c *RedisCache) CacheEmbedding(textHash string, embedding []float32, ttl time.Duration) bool {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	key := c.buildKey("embedding", textHash)
	return c.SetWithTTL(key, embedding, ttl) == nil
}

// GetEmbedding gets cached embedding.
func (c *RedisCache) GetEmbedding(textHash string) ([]float32, bool) {
	key := c.buildKey("embedding", textHash)
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}

	// Handle both []float32 and []interface{} from JSON unmarshaling
	switch v := val.(type) {
	case []float32:
		return v, true
	case []any:
		result := make([]float32, len(v))
		for i, f := range v {
			if fv, ok := f.(float64); ok {
				result[i] = float32(fv)
			}
		}
		return result, true
	}
	return nil, false
}

// InvalidateUser invalidates all cache entries for a user's memory.
func (c *RedisCache) InvalidateUser(userID, memoryID string) error {
	ctx := context.Background()

	pattern := c.buildKey("*", userID, memoryID, "*")
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()

	keys := make([]string, 0)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Printf("Redis scan failed: %v", err)
		return err
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			log.Printf("Redis delete failed: %v", err)
			return err
		}
	}

	log.Printf("Invalidated cache for user %s, memory %s", userID, memoryID)
	return nil
}

// Close closes the Redis connection.
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// HealthCheck checks if Redis is healthy.
func (c *RedisCache) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.client.Ping(ctx).Err() == nil
}

