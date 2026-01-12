# 🧠 GraphMem-Go

## **The Human Brain for Your AI Agents**

<p align="center">
  <a href="https://goreportcard.com/report/github.com/flancast90/GraphMem-go"><img src="https://goreportcard.com/badge/github.com/flancast90/GraphMem-go" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/flancast90/GraphMem-go"><img src="https://pkg.go.dev/badge/github.com/flancast90/GraphMem-go.svg" alt="Go Reference"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
  <a href="https://github.com/flancast90/GraphMem-go"><img src="https://img.shields.io/badge/github-flancast90/GraphMem--go-blue.svg" alt="GitHub"></a>
</p>

> **"Memory is the treasury and guardian of all things."** — Cicero

GraphMem is the **first memory system that thinks like a human brain**. It doesn't just store data—it **forgets**, **consolidates**, **prioritizes**, and **evolves** exactly like biological memory does.

**This is the future of enterprise AI agents.**

---

## 🧬 Why GraphMem Changes Everything

### The Problem with Current AI Memory

Every production AI agent faces the same crisis:

```
Day 1:     "Who is the CEO?" → "Elon Musk" ✅
Day 100:   Context window: OVERFLOW 💥
Day 365:   "Who is the CEO?" → "John... or was it Jane... maybe Elon?" 🤯
```

**Vector databases don't forget.** They accumulate garbage until your agent drowns in irrelevant, conflicting, outdated information.

### The GraphMem Solution: Memory That Thinks

GraphMem implements the **four pillars of human memory**:

| Human Brain | GraphMem | Why It Matters |
|-------------|----------|----------------|
| 🧠 **Forgetting Curve** | Memory Decay | Irrelevant memories fade naturally |
| 🔗 **Neural Networks** | Knowledge Graph | Relationships between concepts |
| ⭐ **Importance Weighting** | PageRank Centrality | Hub concepts (Elon Musk) > peripheral ones |
| ⏰ **Episodic Memory** | Temporal Validity | "CEO in 2015" vs "CEO now" |

---

## 🚀 Quick Start

### Installation

```bash
go get github.com/flancast90/GraphMem-go
```

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/flancast90/GraphMem-go/pkg/graphmem"
)

func main() {
    // Create configuration (reads from environment)
    config := graphmem.NewConfig()
    config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")

    // Create GraphMem instance
    gm, err := graphmem.New(config,
        graphmem.WithUserID("my_agent"),
        graphmem.WithAutoEvolve(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer gm.Close()

    // That's it. 3 methods:
    result, _ := gm.Ingest("Tesla is led by CEO Elon Musk...")  // ← Extract knowledge
    response, _ := gm.Query("Who is the CEO?")                   // ← Ask questions
    gm.Evolve()                                                  // ← Let memory mature

    fmt.Printf("Extracted %d entities\n", result.Entities)
    fmt.Println("Answer:", response.Answer)
}
```

### With Persistence (Neo4j)

```go
config := graphmem.NewConfig()
config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
config.Neo4jURI = "bolt://localhost:7687"
config.Neo4jUser = "neo4j"
config.Neo4jPassword = "password"

gm, err := graphmem.New(config,
    graphmem.WithUserID("my_agent"),
)
// Data persists between restarts!
```

### With Caching (Redis)

```go
config := graphmem.NewConfig()
config.RedisURL = "redis://localhost:6379"

gm, err := graphmem.New(config,
    graphmem.WithUserID("my_agent"),
)
// Queries are cached for performance!
```

### Using Different LLM Providers

```go
// OpenAI (default)
config := graphmem.NewConfig()
config.LLMProvider = "openai"
config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
config.LLMModel = "gpt-4o-mini"

// Anthropic Claude
config.LLMProvider = "anthropic"
config.LLMAPIKey = os.Getenv("ANTHROPIC_API_KEY")
config.LLMModel = "claude-3-haiku-20240307"

// Azure OpenAI
config.LLMProvider = "azure_openai"
config.AzureOpenAIEndpoint = "https://your-resource.openai.azure.com"
config.AzureOpenAIDeployment = "gpt-4"
config.LLMAPIKey = os.Getenv("AZURE_OPENAI_API_KEY")

// Local Ollama
config.LLMProvider = "ollama"
config.OllamaBaseURL = "http://localhost:11434"
config.LLMModel = "llama3.2"
```

---

## 🎯 Revolutionary Features

### Point-in-Time Memory

Query the past: *"Who was CEO in 2015?"*

[Learn more →](concepts/temporal.md)

### Knowledge Graph

Automatic entity extraction and relationship mapping

[Learn more →](concepts/knowledge-graph.md)

### Self-Evolution

Memory that consolidates, decays, and improves

[Learn more →](concepts/evolution.md)

### Multi-Tenant Isolation

Complete data separation for enterprise

[Learn more →](concepts/multi-tenancy.md)

---

## 📊 Performance

| Metric | Naive RAG | GraphMem | Advantage |
|--------|-----------|----------|-----------|
| **1K conversations** | 💥 Context overflow | ✅ Bounded | Handles growth |
| **10K entities** | O(n) = 2.3s | O(1) = 50ms | **46x faster** |
| **1 year history** | 3,650 entries | ~100 consolidated | **97% reduction** |
| **Entity conflicts** | Duplicates | Auto-resolved | Clean data |
| **Temporal queries** | ❌ Impossible | ✅ Native | Unique capability |

---

## 🐳 Docker Setup

```bash
# Start all services (Neo4j, Redis, LibSQL)
make services-up

# Run integration tests
make test-integration

# Stop services
make services-down
```

See [docker-compose.yml](../docker-compose.yml) for full configuration.

---

## 📚 Documentation

- **[Getting Started](getting-started/installation.md)** - Installation, quick start, and configuration
- **[Core Concepts](concepts/overview.md)** - Understanding how GraphMem works
- **[Building Agents](agents/guide.md)** - Complete guide to building AI agents
- **[Production](production/architecture.md)** - Deploy at scale with confidence

---

## 🤝 Contributing

We're building the future of AI memory. Join us!

- 🐛 [Report bugs](https://github.com/flancast90/GraphMem-go/issues)
- 💡 [Request features](https://github.com/flancast90/GraphMem-go/issues)
- 🔀 [Submit PRs](https://github.com/flancast90/GraphMem-go/pulls)

---

<div align="center">

**GraphMem-Go** - A Go implementation of GraphMem

*"Give your AI agents the memory they deserve."*

</div>
