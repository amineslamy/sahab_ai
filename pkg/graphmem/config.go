package graphmem

import (
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for GraphMem.
type Config struct {
	// Multi-tenant isolation
	UserID string `json:"user_id,omitempty"`

	// Storage backends
	Neo4jURI       string `json:"neo4j_uri,omitempty"`
	Neo4jUsername  string `json:"neo4j_username,omitempty"`
	Neo4jPassword  string `json:"neo4j_password,omitempty"`
	Neo4jDatabase  string `json:"neo4j_database,omitempty"`
	RedisURL       string `json:"redis_url,omitempty"`
	RedisTTL       int    `json:"redis_ttl"`
	TursoDBPath    string `json:"turso_db_path,omitempty"`
	TursoURL       string `json:"turso_url,omitempty"`
	TursoAuthToken string `json:"turso_auth_token,omitempty"`

	// LLM Configuration
	LLMProvider    string  `json:"llm_provider"`
	LLMModel       string  `json:"llm_model"`
	LLMAPIKey      string  `json:"llm_api_key,omitempty"`
	LLMAPIBase     string  `json:"llm_api_base,omitempty"`
	LLMTemperature float64 `json:"llm_temperature"`
	LLMMaxTokens   int     `json:"llm_max_tokens"`

	// Azure OpenAI Configuration
	AzureAPIVersion          string `json:"azure_api_version,omitempty"`
	AzureDeployment          string `json:"azure_deployment,omitempty"`
	AzureEmbeddingDeployment string `json:"azure_embedding_deployment,omitempty"`

	// Embedding Configuration
	EmbeddingProvider   string `json:"embedding_provider"`
	EmbeddingModel      string `json:"embedding_model"`
	EmbeddingAPIKey     string `json:"embedding_api_key,omitempty"`
	EmbeddingAPIBase    string `json:"embedding_api_base,omitempty"`
	EmbeddingDimensions int    `json:"embedding_dimensions"`

	// Extraction Configuration
	ChunkSize           int `json:"chunk_size"`
	ChunkOverlap        int `json:"chunk_overlap"`
	MaxTripletsPerChunk int `json:"max_triplets_per_chunk"`
	ExtractionWorkers   int `json:"extraction_workers"`

	// Query Configuration
	SimilarityTopK         int     `json:"similarity_top_k"`
	MinSimilarityThreshold float64 `json:"min_similarity_threshold"`
	MaxContextLength       int     `json:"max_context_length"`

	// Evolution Configuration
	EvolutionEnabled       bool             `json:"evolution_enabled"`
	ConsolidationThreshold float64          `json:"consolidation_threshold"`
	DecayEnabled           bool             `json:"decay_enabled"`
	DecayHalfLifeDays      float64          `json:"decay_half_life_days"`
	MinImportanceToKeep    MemoryImportance `json:"min_importance_to_keep"`
	RehydrationEnabled     bool             `json:"rehydration_enabled"`

	// Community Detection
	MaxClusterSize     int    `json:"max_cluster_size"`
	MinClusterSize     int    `json:"min_cluster_size"`
	CommunityAlgorithm string `json:"community_algorithm"`

	// Retry and Resilience
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	QueryTimeout      time.Duration `json:"query_timeout"`

	// Parallel Processing
	MaxWorkers int `json:"max_workers"`
	BatchSize  int `json:"batch_size"`

	// Logging and Monitoring
	LogLevel      string `json:"log_level"`
	EnableMetrics bool   `json:"enable_metrics"`
	EnableTracing bool   `json:"enable_tracing"`

	// Feature Flags
	EnableMultimodal          bool `json:"enable_multimodal"`
	EnableWebResearch         bool `json:"enable_web_research"`
	EnableSyntheticGeneration bool `json:"enable_synthetic_generation"`
}

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		UserID:         getEnv("GRAPHMEM_USER_ID", ""),
		Neo4jURI:       getEnv("GRAPHMEM_NEO4J_URI", ""),
		Neo4jUsername:  getEnv("GRAPHMEM_NEO4J_USERNAME", "neo4j"),
		Neo4jPassword:  getEnv("GRAPHMEM_NEO4J_PASSWORD", ""),
		Neo4jDatabase:  getEnv("GRAPHMEM_NEO4J_DATABASE", "neo4j"),
		RedisURL:       getEnv("GRAPHMEM_REDIS_URL", ""),
		RedisTTL:       getEnvInt("GRAPHMEM_REDIS_TTL", 3600),
		TursoDBPath:    getEnv("GRAPHMEM_TURSO_DB_PATH", ""),
		TursoURL:       getEnv("GRAPHMEM_TURSO_URL", ""),
		TursoAuthToken: getEnv("GRAPHMEM_TURSO_AUTH_TOKEN", ""),

		LLMProvider:    getEnv("GRAPHMEM_LLM_PROVIDER", "openai"),
		LLMModel:       getEnv("GRAPHMEM_LLM_MODEL", "gpt-4o-mini"),
		LLMAPIKey:      getEnvOrOpenAIKey(),
		LLMAPIBase:     getEnv("GRAPHMEM_LLM_API_BASE", ""),
		LLMTemperature: getEnvFloat("GRAPHMEM_LLM_TEMPERATURE", 0.1),
		LLMMaxTokens:   getEnvInt("GRAPHMEM_LLM_MAX_TOKENS", 8000),

		AzureAPIVersion:          getEnv("AZURE_OPENAI_API_VERSION", "2024-12-01-preview"),
		AzureDeployment:          getEnv("AZURE_OPENAI_DEPLOYMENT", ""),
		AzureEmbeddingDeployment: getEnv("AZURE_EMBEDDING_DEPLOYMENT", ""),

		EmbeddingProvider:   getEnv("GRAPHMEM_EMBEDDING_PROVIDER", "openai"),
		EmbeddingModel:      getEnv("GRAPHMEM_EMBEDDING_MODEL", "text-embedding-3-small"),
		EmbeddingAPIKey:     getEnvOrOpenAIKey(),
		EmbeddingAPIBase:    getEnv("GRAPHMEM_EMBEDDING_API_BASE", ""),
		EmbeddingDimensions: getEnvInt("GRAPHMEM_EMBEDDING_DIMENSIONS", 1536),

		ChunkSize:           getEnvInt("GRAPHMEM_CHUNK_SIZE", 2048),
		ChunkOverlap:        getEnvInt("GRAPHMEM_CHUNK_OVERLAP", 200),
		MaxTripletsPerChunk: getEnvInt("GRAPHMEM_MAX_TRIPLETS_PER_CHUNK", 40),
		ExtractionWorkers:   getEnvInt("GRAPHMEM_EXTRACTION_WORKERS", 8),

		SimilarityTopK:         getEnvInt("GRAPHMEM_SIMILARITY_TOP_K", 10),
		MinSimilarityThreshold: getEnvFloat("GRAPHMEM_MIN_SIMILARITY_THRESHOLD", 0.5),
		MaxContextLength:       getEnvInt("GRAPHMEM_MAX_CONTEXT_LENGTH", 16000),

		EvolutionEnabled:       getEnvBool("GRAPHMEM_EVOLUTION_ENABLED", true),
		ConsolidationThreshold: getEnvFloat("GRAPHMEM_CONSOLIDATION_THRESHOLD", 0.85),
		DecayEnabled:           getEnvBool("GRAPHMEM_DECAY_ENABLED", true),
		DecayHalfLifeDays:      getEnvFloat("GRAPHMEM_DECAY_HALF_LIFE_DAYS", 30.0),
		MinImportanceToKeep:    MemoryImportance(getEnvInt("GRAPHMEM_MIN_IMPORTANCE_TO_KEEP", int(ImportanceVeryLow))),
		RehydrationEnabled:     getEnvBool("GRAPHMEM_REHYDRATION_ENABLED", true),

		MaxClusterSize:     getEnvInt("GRAPHMEM_MAX_CLUSTER_SIZE", 100),
		MinClusterSize:     getEnvInt("GRAPHMEM_MIN_CLUSTER_SIZE", 2),
		CommunityAlgorithm: getEnv("GRAPHMEM_COMMUNITY_ALGORITHM", "greedy_modularity"),

		MaxRetries:        getEnvInt("GRAPHMEM_MAX_RETRIES", 3),
		RetryDelay:        time.Duration(getEnvInt("GRAPHMEM_RETRY_DELAY", 5)) * time.Second,
		ConnectionTimeout: time.Duration(getEnvInt("GRAPHMEM_CONNECTION_TIMEOUT", 30)) * time.Second,
		QueryTimeout:      time.Duration(getEnvInt("GRAPHMEM_QUERY_TIMEOUT", 60)) * time.Second,

		MaxWorkers: getEnvInt("GRAPHMEM_MAX_WORKERS", 8),
		BatchSize:  getEnvInt("GRAPHMEM_BATCH_SIZE", 500),

		LogLevel:      getEnv("GRAPHMEM_LOG_LEVEL", "INFO"),
		EnableMetrics: getEnvBool("GRAPHMEM_ENABLE_METRICS", true),
		EnableTracing: getEnvBool("GRAPHMEM_ENABLE_TRACING", false),

		EnableMultimodal:          getEnvBool("GRAPHMEM_ENABLE_MULTIMODAL", true),
		EnableWebResearch:         getEnvBool("GRAPHMEM_ENABLE_WEB_RESEARCH", true),
		EnableSyntheticGeneration: getEnvBool("GRAPHMEM_ENABLE_SYNTHETIC_GENERATION", true),
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.LLMAPIKey == "" {
		return NewConfigurationError("LLM API key is required").
			WithSuggestion("Set GRAPHMEM_LLM_API_KEY or OPENAI_API_KEY environment variable")
	}
	return nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrOpenAIKey() string {
	if key := os.Getenv("GRAPHMEM_LLM_API_KEY"); key != "" {
		return key
	}
	return os.Getenv("OPENAI_API_KEY")
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
