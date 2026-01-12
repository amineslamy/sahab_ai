# Installation

## Requirements

- Go 1.21 or higher
- OpenAI API key (or other LLM provider)

## Install

```bash
go get github.com/flancast90/GraphMem-go
```

## Optional Dependencies

For full functionality, you may want to set up:

### Neo4j (Graph Storage)

```bash
# Using Docker
docker run -d \
  --name neo4j \
  -p 7474:7474 -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/password \
  neo4j:5.15-community
```

### Redis (Caching)

```bash
# Using Docker
docker run -d \
  --name redis \
  -p 6379:6379 \
  redis:7.2-alpine
```

### LibSQL/Turso (SQLite Storage)

```bash
# Using Docker
docker run -d \
  --name libsql \
  -p 8080:8080 \
  ghcr.io/tursodatabase/libsql-server:latest
```

## Docker Compose (Recommended)

The easiest way to get all services running:

```bash
# Clone the repo
git clone https://github.com/flancast90/GraphMem-go.git
cd GraphMem-go

# Copy and configure environment
cp docker.env.example .env
# Edit .env with your API keys

# Start all services
make services-up

# Verify services are running
make services-status
```

## Environment Variables

Create a `.env` file or set these environment variables:

```bash
# Required: LLM Provider
OPENAI_API_KEY=sk-your-key-here

# Optional: Storage backends
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password

REDIS_URL=redis://localhost:6379

# Optional: Alternative LLM providers
ANTHROPIC_API_KEY=sk-ant-your-key
AZURE_OPENAI_API_KEY=your-azure-key
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
```

## Verify Installation

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/flancast90/GraphMem-go/pkg/graphmem"
)

func main() {
    config := graphmem.NewConfig()
    config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
    
    gm, err := graphmem.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer gm.Close()
    
    fmt.Println("GraphMem initialized successfully!")
}
```

Run it:

```bash
go run main.go
```

## Next Steps

- [Quick Start Guide](quickstart.md) - Build your first memory-powered agent
- [Configuration](configuration.md) - Detailed configuration options
- [Storage Options](storage.md) - Choose the right storage backend
