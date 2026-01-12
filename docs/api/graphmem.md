# GraphMem API

The main interface for all GraphMem operations.

## Creating an Instance

```go
import "github.com/flancast90/GraphMem-go/pkg/graphmem"

// Create with default configuration
config := graphmem.NewConfig()
config.LLMAPIKey = "sk-..."

gm, err := graphmem.New(config)
if err != nil {
    log.Fatal(err)
}
defer gm.Close()

// Create with options
gm, err := graphmem.New(config,
    graphmem.WithUserID("user123"),
    graphmem.WithMemoryID("session456"),
    graphmem.WithAutoEvolve(true),
)
```

## Options

| Option | Description |
|--------|-------------|
| `WithUserID(id string)` | Set user ID for multi-tenant isolation |
| `WithMemoryID(id string)` | Set specific memory ID |
| `WithAutoEvolve(bool)` | Enable automatic memory evolution |

## Methods

### Ingest

Extract entities and relationships from text content.

```go
func (gm *GraphMem) Ingest(content string) (*IngestResult, error)
```

**Parameters:**
- `content` - Text content to process

**Returns:**
- `IngestResult` with extraction statistics
- `error` if extraction fails

**Example:**

```go
result, err := gm.Ingest("Apple was founded by Steve Jobs in 1976.")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Entities: %d\n", result.Entities)
fmt.Printf("Relationships: %d\n", result.Relationships)
fmt.Printf("Memory ID: %s\n", result.MemoryID)
```

### Query

Query the memory and get an answer.

```go
func (gm *GraphMem) Query(query string) (*MemoryResponse, error)
```

**Parameters:**
- `query` - Natural language question

**Returns:**
- `MemoryResponse` with answer and metadata
- `error` if query fails

**Example:**

```go
response, err := gm.Query("Who founded Apple?")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Answer:", response.Answer)
fmt.Printf("Confidence: %.2f\n", response.Confidence)
fmt.Printf("Nodes used: %d\n", len(response.Nodes))
fmt.Printf("Edges used: %d\n", len(response.Edges))
```

### Evolve

Trigger memory evolution (consolidation, decay, rehydration).

```go
func (gm *GraphMem) Evolve() ([]*EvolutionEvent, error)
```

**Returns:**
- Slice of `EvolutionEvent` describing changes
- `error` if evolution fails

**Example:**

```go
events, err := gm.Evolve()
if err != nil {
    log.Fatal(err)
}

for _, event := range events {
    fmt.Printf("%s: %s\n", event.EvolutionType, event.Description)
}
```

### Close

Clean up resources and close connections.

```go
func (gm *GraphMem) Close() error
```

**Example:**

```go
defer gm.Close()
```

## Types

### IngestResult

```go
type IngestResult struct {
    MemoryID      string
    Entities      int
    Relationships int
    Duration      time.Duration
}
```

### MemoryResponse

```go
type MemoryResponse struct {
    Answer     string
    Confidence float64
    Nodes      []*MemoryNode
    Edges      []*MemoryEdge
    Sources    []string
}
```

### EvolutionEvent

```go
type EvolutionEvent struct {
    ID            string
    EvolutionType EvolutionType  // CONSOLIDATION, DECAY, REHYDRATION, etc.
    Description   string
    AffectedNodes []string
    Timestamp     time.Time
    Metadata      map[string]any
}
```

## Error Handling

GraphMem uses custom error types for better error handling:

```go
result, err := gm.Ingest(content)
if err != nil {
    switch e := err.(type) {
    case *graphmem.LLMError:
        log.Printf("LLM error: %s", e.Message)
    case *graphmem.StorageError:
        log.Printf("Storage error: %s (type: %s)", e.Message, e.StorageType)
    case *graphmem.ValidationError:
        log.Printf("Validation error: %s", e.Message)
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

## Configuration

See [Configuration](../getting-started/configuration.md) for all configuration options.
