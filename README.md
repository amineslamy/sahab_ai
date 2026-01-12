# GraphMem-Go

A Go implementation of GraphMem - a knowledge graph-based memory system for AI applications.

[![Go Report Card](https://goreportcard.com/badge/github.com/graphmem/graphmem-go)](https://goreportcard.com/report/github.com/graphmem/graphmem-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/graphmem/graphmem-go.svg)](https://pkg.go.dev/github.com/graphmem/graphmem-go)

## Features

- **Knowledge Graph Memory**: Extracts entities and relationships from text using LLMs
- **Entity Resolution**: Deduplicates and merges similar entities using semantic similarity
- **Memory Evolution**: Implements the forgetting curve with consolidation, decay, and reinforcement
- **Temporal Validity**: Tracks relationships with valid_from and valid_until timestamps
- **Multi-tenant Isolation**: Supports per-user/per-session memory isolation
- **PageRank Importance**: Uses graph centrality for memory importance scoring
- **Hybrid Retrieval**: Combines exact match, semantic search, and graph traversal

## Installation

```bash
go get github.com/graphmem/graphmem-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/graphmem/graphmem-go/pkg/graphmem"
)

func main() {
    // Create configuration
    config := graphmem.NewConfig()
    // config.LLMAPIKey = "your-api-key" // Or set OPENAI_API_KEY env var

    // Create GraphMem instance
    gm, err := graphmem.New(config,
        graphmem.WithUserID("user123"),
        graphmem.WithAutoEvolve(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer gm.Close()

    // Ingest content
    result, err := gm.Ingest("Apple Inc. was founded by Steve Jobs in 1976. Tim Cook is the current CEO.")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Extracted %d entities, %d relationships\n", result.Entities, result.Relationships)

    // Query memory
    response, err := gm.Query("Who founded Apple?")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Answer:", response.Answer)
}
```

## Architecture

```
pkg/graphmem/
├── types.go           # Core data structures (MemoryNode, MemoryEdge, Memory, etc.)
├── errors.go          # Custom error types
├── config.go          # Configuration with environment variable support
├── llm.go             # LLM provider abstraction (OpenAI, Azure OpenAI)
├── embedding.go       # Embedding provider abstraction
├── store.go           # Storage backends (in-memory, extensible)
├── knowledge_graph.go # Knowledge graph extraction
├── entity_resolver.go # Entity deduplication and merging
├── retriever.go       # Memory retrieval and query
├── evolution.go       # Memory evolution (consolidation, decay)
└── graphmem.go        # Main GraphMem interface
```

## Core Concepts

### Memory Node
Represents an entity in the knowledge graph with:
- Name and entity type
- Aliases for deduplication
- Embedding vector for semantic similarity
- Importance level (Ephemeral to Critical)
- State (Active, Decaying, Archived, Deleted)
- Multi-tenant isolation (UserID, MemoryID)

### Memory Edge
Represents a relationship between entities with:
- Source and target node IDs
- Relation type and description
- Weight and confidence scores
- Temporal validity (valid_from, valid_until)

### Memory Evolution
Implements human-like memory behavior:
- **Consolidation**: Merges similar entities above similarity threshold
- **Decay**: Reduces importance over time based on forgetting curve
- **Reinforcement**: Updates importance based on PageRank and access patterns

## Configuration

Environment variables:

```bash
# LLM Configuration
OPENAI_API_KEY=your-api-key
GRAPHMEM_LLM_PROVIDER=openai          # openai, azure_openai
GRAPHMEM_LLM_MODEL=gpt-4o-mini

# Embedding Configuration
GRAPHMEM_EMBEDDING_PROVIDER=openai
GRAPHMEM_EMBEDDING_MODEL=text-embedding-3-small

# Evolution Configuration
GRAPHMEM_EVOLUTION_ENABLED=true
GRAPHMEM_DECAY_ENABLED=true
GRAPHMEM_DECAY_HALF_LIFE_DAYS=30
GRAPHMEM_CONSOLIDATION_THRESHOLD=0.85

# Azure OpenAI (if using)
AZURE_OPENAI_API_VERSION=2024-12-01-preview
AZURE_OPENAI_DEPLOYMENT=your-deployment
AZURE_EMBEDDING_DEPLOYMENT=your-embedding-deployment
```

## Testing

### Unit Tests (No External Services)

```bash
# Run unit tests
go test ./...

# Run with verbose output
go test ./pkg/graphmem/... -v

# Run with coverage
go test ./... -cover
```

### Integration Tests (Require API Keys)

1. Copy the environment file:
```bash
cp env.example .env
```

2. Fill in your API keys in `.env`:
```bash
# Required for LLM tests
OPENAI_API_KEY=sk-your-key-here

# Or use other providers
ANTHROPIC_API_KEY=sk-ant-your-key-here
GROQ_API_KEY=gsk_your-key-here

# For storage backend tests
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-password

REDIS_URL=redis://localhost:6379
```

3. Set test flags:
```bash
export RUN_LLM_TESTS=true        # Enable LLM tests
export RUN_NEO4J_TESTS=true      # Enable Neo4j tests
export RUN_REDIS_TESTS=true      # Enable Redis tests
export RUN_TURSO_TESTS=true      # Enable Turso tests
```

4. Run integration tests:
```bash
# Load environment
source .env

# Run integration tests
go test -tags=integration ./pkg/graphmem/... -v

# Run specific test
go test -tags=integration ./pkg/graphmem/... -v -run TestOpenAILLMIntegration

# Run all LLM provider tests
go test -tags=integration ./pkg/graphmem/... -v -run "Test.*LLM.*Integration"

# Run knowledge graph extraction test
go test -tags=integration ./pkg/graphmem/... -v -run TestKnowledgeGraphExtractionIntegration

# Run end-to-end test
go test -tags=integration ./pkg/graphmem/... -v -run TestEndToEndIntegration
```

### Available Integration Tests

| Test | Required Env Vars |
|------|-------------------|
| `TestOpenAILLMIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestAnthropicLLMIntegration` | `ANTHROPIC_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestOllamaLLMIntegration` | `RUN_LLM_TESTS=true` (Ollama running locally) |
| `TestGroqLLMIntegration` | `GROQ_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestOpenAIEmbeddingIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestKnowledgeGraphExtractionIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestEntityResolutionIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestSemanticSearchIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestCommunityDetectionIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestConsolidationIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestHighPerformancePipelineIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestQueryEngineIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestEndToEndIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestRehydrationIntegration` | `OPENAI_API_KEY`, `RUN_LLM_TESTS=true` |
| `TestNeo4jStoreIntegration` | `NEO4J_*`, `RUN_NEO4J_TESTS=true` |
| `TestRedisCacheIntegration` | `REDIS_URL`, `RUN_REDIS_TESTS=true` |
| `TestTursoStoreIntegration` | `TURSO_*`, `RUN_TURSO_TESTS=true` |

### Test Categories

- **Unit Tests**: No build tags, no external dependencies
- **Chunker Tests**: `TestDocument*ChunkerIntegration` - No API keys needed
- **LLM Tests**: Require `RUN_LLM_TESTS=true` and appropriate API keys
- **Storage Tests**: Require specific backend flags and connections

## Docker Setup

The project includes a complete Docker setup with all dependencies (Neo4j, Redis, LibSQL/Turso).

### Quick Start with Docker

```bash
# Copy environment file
cp docker.env.example .env

# Edit .env with your API keys
nano .env

# Start all services
make docker-up

# Or use docker-compose directly
docker-compose --profile full up -d
```

### Running Integration Tests in Docker

```bash
# Run all integration tests
make docker-test

# Or manually
docker-compose --profile test up --build --abort-on-container-exit
```

### Development Mode (Services Only)

Start only the infrastructure services and run tests locally:

```bash
# Start Neo4j, Redis, LibSQL
make services-up

# Check service status
make services-status

# Run integration tests locally
make test-integration

# Stop services
make services-down
```

### Using Hosted Services

To use hosted services instead of local containers, edit your `.env` file:

```bash
# Set mode to hosted
USE_LOCAL_SERVICES=false

# Configure hosted Neo4j (e.g., Neo4j Aura)
NEO4J_URI=neo4j+s://your-instance.databases.neo4j.io
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-aura-password

# Configure hosted Redis (e.g., Redis Cloud)
REDIS_URL=redis://user:password@your-host.redis-cloud.com:16379

# Configure Turso Cloud
TURSO_DATABASE_URL=libsql://your-db-name.turso.io
TURSO_AUTH_TOKEN=your-turso-auth-token
```

Then start the application without local services:

```bash
docker-compose --profile app up -d
```

### Local LLM with Ollama

```bash
# Start Ollama container
make ollama-up

# Pull a model
make ollama-pull MODEL=llama3.2

# List available models
make ollama-models
```

### Docker Profiles

| Profile | Services | Use Case |
|---------|----------|----------|
| `local` | Neo4j, Redis, LibSQL | Infrastructure only |
| `test` | Infrastructure + Test Runner | Run integration tests |
| `app` | GraphMem application | Production deployment |
| `full` | All services | Full development stack |
| `local-llm` | Ollama | Local LLM inference |

### Service URLs (Local Mode)

| Service | URL | Credentials |
|---------|-----|-------------|
| Neo4j Browser | http://localhost:7474 | neo4j / graphmem_password |
| Neo4j Bolt | bolt://localhost:7687 | neo4j / graphmem_password |
| Redis | redis://localhost:6379 | - |
| LibSQL | http://localhost:8080 | - |
| Ollama | http://localhost:11434 | - |

## Go Report Card Compliance

This project maintains 100% Go Report Card compliance:

```bash
# Format
gofmt -w ./pkg/

# Vet
go vet ./...

# Static analysis
staticcheck ./...
```

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please ensure:
1. All tests pass: `go test ./...`
2. Code is formatted: `gofmt -w .`
3. No vet issues: `go vet ./...`
4. No staticcheck issues: `staticcheck ./...`
