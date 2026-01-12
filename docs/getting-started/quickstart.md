# Quick Start

This guide will get you up and running with GraphMem-Go in 5 minutes.

## Prerequisites

- Go 1.21+
- OpenAI API key

## Step 1: Install

```bash
go get github.com/flancast90/GraphMem-go
```

## Step 2: Create Your First Agent

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/flancast90/GraphMem-go/pkg/graphmem"
)

func main() {
    // Initialize configuration
    config := graphmem.NewConfig()
    config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
    config.LLMModel = "gpt-4o-mini"
    config.EmbeddingModel = "text-embedding-3-small"

    // Create GraphMem instance
    gm, err := graphmem.New(config,
        graphmem.WithUserID("user123"),
        graphmem.WithAutoEvolve(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer gm.Close()

    // Ingest some knowledge
    content := `
    Apple Inc. was founded by Steve Jobs, Steve Wozniak, and Ronald Wayne 
    in 1976. The company is headquartered in Cupertino, California. 
    Tim Cook became CEO in 2011 after Steve Jobs resigned due to health issues.
    Apple created the iPhone, which revolutionized the smartphone industry.
    `

    result, err := gm.Ingest(content)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✅ Extracted %d entities and %d relationships\n", 
        result.Entities, result.Relationships)

    // Query the memory
    response, err := gm.Query("Who founded Apple?")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\n🔍 Query: Who founded Apple?\n")
    fmt.Printf("📝 Answer: %s\n", response.Answer)
    fmt.Printf("🎯 Confidence: %.2f\n", response.Confidence)

    // Try another query
    response, err = gm.Query("What products did Apple create?")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\n🔍 Query: What products did Apple create?\n")
    fmt.Printf("📝 Answer: %s\n", response.Answer)
}
```

## Step 3: Run

```bash
export OPENAI_API_KEY=sk-your-key-here
go run main.go
```

Expected output:

```
✅ Extracted 10 entities and 8 relationships

🔍 Query: Who founded Apple?
📝 Answer: Apple was founded by Steve Jobs, Steve Wozniak, and Ronald Wayne in 1976.
🎯 Confidence: 0.92

🔍 Query: What products did Apple create?
📝 Answer: Apple created the iPhone, which revolutionized the smartphone industry.
```

## Understanding the Output

### Entities Extracted

GraphMem automatically extracts entities like:
- **People**: Steve Jobs, Steve Wozniak, Ronald Wayne, Tim Cook
- **Organizations**: Apple Inc.
- **Products**: iPhone
- **Locations**: Cupertino, California
- **Dates**: 1976, 2011

### Relationships Discovered

It also discovers relationships:
- Steve Jobs → founded → Apple Inc.
- Apple Inc. → headquartered in → Cupertino
- Tim Cook → became CEO of → Apple Inc.
- Apple Inc. → created → iPhone

## Adding Persistence

To persist data between runs, add Neo4j:

```go
config := graphmem.NewConfig()
config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
config.Neo4jURI = "bolt://localhost:7687"
config.Neo4jUser = "neo4j"
config.Neo4jPassword = "password"

gm, err := graphmem.New(config,
    graphmem.WithUserID("user123"),
)
```

## Memory Evolution

GraphMem can automatically evolve memory over time:

```go
// Enable auto-evolution
gm, err := graphmem.New(config,
    graphmem.WithAutoEvolve(true),
)

// Or manually trigger evolution
events, err := gm.Evolve()
fmt.Printf("Evolution events: %d\n", len(events))
```

Evolution includes:
- **Consolidation**: Merging similar entities ("Steve Jobs" + "Jobs" → "Steve Jobs")
- **Decay**: Reducing importance of rarely-accessed memories
- **Rehydration**: Restoring relevant archived memories

## Next Steps

- [Configuration](configuration.md) - All configuration options
- [Storage Options](storage.md) - Neo4j, Redis, Turso setup
- [Building Agents](../agents/guide.md) - Create production agents
- [API Reference](../api/graphmem.md) - Full API documentation
