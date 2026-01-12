package graphmem

import (
	"fmt"
	"time"
)

// GraphMemError is the base error type for all GraphMem errors.
type GraphMemError struct {
	Message     string
	Code        string
	Context     map[string]any
	Timestamp   time.Time
	Recoverable bool
	Suggestions []string
	Cause       error
}

func (e *GraphMemError) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Code, e.Message)
	if len(e.Context) > 0 {
		msg += fmt.Sprintf(" | Context: %v", e.Context)
	}
	if len(e.Suggestions) > 0 {
		msg += fmt.Sprintf(" | Suggestions: %v", e.Suggestions)
	}
	if e.Cause != nil {
		msg += fmt.Sprintf(" | Cause: %v", e.Cause)
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *GraphMemError) Unwrap() error {
	return e.Cause
}

// ConfigurationError represents a configuration error.
type ConfigurationError struct {
	*GraphMemError
}

// NewConfigurationError creates a ConfigurationError.
func NewConfigurationError(message string) *ConfigurationError {
	return &ConfigurationError{
		GraphMemError: &GraphMemError{
			Message:   message,
			Code:      "CONFIG_ERROR",
			Context:   make(map[string]any),
			Timestamp: time.Now().UTC(),
		},
	}
}

// WithSuggestion adds a suggestion.
func (e *ConfigurationError) WithSuggestion(s string) *ConfigurationError {
	e.Suggestions = append(e.Suggestions, s)
	return e
}

// StorageError represents a storage error.
type StorageError struct {
	*GraphMemError
}

// NewStorageError creates a StorageError.
func NewStorageError(message string) *StorageError {
	return &StorageError{
		GraphMemError: &GraphMemError{
			Message:   message,
			Code:      "STORAGE_ERROR",
			Context:   make(map[string]any),
			Timestamp: time.Now().UTC(),
		},
	}
}

// WithOperation adds the operation context.
func (e *StorageError) WithOperation(op string) *StorageError {
	e.Context["operation"] = op
	return e
}

// WithStorageType adds the storage type context.
func (e *StorageError) WithStorageType(st string) *StorageError {
	e.Context["storage_type"] = st
	return e
}

// WithCause adds the cause error.
func (e *StorageError) WithCause(cause error) *StorageError {
	e.Cause = cause
	return e
}

// IngestionError represents an ingestion error.
type IngestionError struct {
	*GraphMemError
}

// NewIngestionError creates an IngestionError.
func NewIngestionError(message string) *IngestionError {
	return &IngestionError{
		GraphMemError: &GraphMemError{
			Message:   message,
			Code:      "INGESTION_ERROR",
			Context:   make(map[string]any),
			Timestamp: time.Now().UTC(),
		},
	}
}

// WithStage adds the stage context.
func (e *IngestionError) WithStage(stage string) *IngestionError {
	e.Context["stage"] = stage
	return e
}

// WithCause adds the cause error.
func (e *IngestionError) WithCause(cause error) *IngestionError {
	e.Cause = cause
	return e
}

// QueryError represents a query error.
type QueryError struct {
	*GraphMemError
}

// NewQueryError creates a QueryError.
func NewQueryError(message string) *QueryError {
	return &QueryError{
		GraphMemError: &GraphMemError{
			Message:   message,
			Code:      "QUERY_ERROR",
			Context:   make(map[string]any),
			Timestamp: time.Now().UTC(),
		},
	}
}

// WithQuery adds the query context.
func (e *QueryError) WithQuery(query string) *QueryError {
	e.Context["query"] = query
	return e
}

// WithCause adds the cause error.
func (e *QueryError) WithCause(cause error) *QueryError {
	e.Cause = cause
	return e
}

// EvolutionError represents an evolution error.
type EvolutionError struct {
	*GraphMemError
}

// NewEvolutionError creates an EvolutionError.
func NewEvolutionError(message string) *EvolutionError {
	return &EvolutionError{
		GraphMemError: &GraphMemError{
			Message:   message,
			Code:      "EVOLUTION_ERROR",
			Context:   make(map[string]any),
			Timestamp: time.Now().UTC(),
		},
	}
}

// WithCause adds the cause error.
func (e *EvolutionError) WithCause(cause error) *EvolutionError {
	e.Cause = cause
	return e
}

// ExtractionError represents an extraction error.
type ExtractionError struct {
	*GraphMemError
}

// NewExtractionError creates an ExtractionError.
func NewExtractionError(message string) *ExtractionError {
	return &ExtractionError{
		GraphMemError: &GraphMemError{
			Message:   message,
			Code:      "EXTRACTION_ERROR",
			Context:   make(map[string]any),
			Timestamp: time.Now().UTC(),
		},
	}
}

// WithCause adds the cause error.
func (e *ExtractionError) WithCause(cause error) *ExtractionError {
	e.Cause = cause
	return e
}

// EmbeddingError represents an embedding error.
type EmbeddingError struct {
	*GraphMemError
}

// NewEmbeddingError creates an EmbeddingError.
func NewEmbeddingError(message string) *EmbeddingError {
	return &EmbeddingError{
		GraphMemError: &GraphMemError{
			Message:   message,
			Code:      "EMBEDDING_ERROR",
			Context:   make(map[string]any),
			Timestamp: time.Now().UTC(),
		},
	}
}

// WithCause adds the cause error.
func (e *EmbeddingError) WithCause(cause error) *EmbeddingError {
	e.Cause = cause
	return e
}

// RateLimitError represents a rate limit error.
type RateLimitError struct {
	*GraphMemError
	RetryAfter time.Duration
}

// NewRateLimitError creates a RateLimitError.
func NewRateLimitError(message string, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		GraphMemError: &GraphMemError{
			Message:     message,
			Code:        "RATE_LIMIT_ERROR",
			Context:     make(map[string]any),
			Timestamp:   time.Now().UTC(),
			Recoverable: true,
		},
		RetryAfter: retryAfter,
	}
}

// TimeoutError represents a timeout error.
type TimeoutError struct {
	*GraphMemError
}

// NewTimeoutError creates a TimeoutError.
func NewTimeoutError(message string) *TimeoutError {
	return &TimeoutError{
		GraphMemError: &GraphMemError{
			Message:     message,
			Code:        "TIMEOUT_ERROR",
			Context:     make(map[string]any),
			Timestamp:   time.Now().UTC(),
			Recoverable: true,
		},
	}
}

// WithCause adds the cause error.
func (e *TimeoutError) WithCause(cause error) *TimeoutError {
	e.Cause = cause
	return e
}
