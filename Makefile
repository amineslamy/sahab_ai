# GraphMem-Go Makefile
# Convenient commands for development and testing

.PHONY: help build test test-unit test-integration lint fmt vet clean \
        docker-build docker-up docker-down docker-test docker-logs \
        services-up services-down

# Default target
help:
	@echo "GraphMem-Go Development Commands"
	@echo ""
	@echo "Local Development:"
	@echo "  make build           - Build the Go application"
	@echo "  make test            - Run all unit tests"
	@echo "  make test-unit       - Run unit tests only (no external dependencies)"
	@echo "  make test-integration - Run integration tests (requires services)"
	@echo "  make lint            - Run all linters (fmt, vet, staticcheck)"
	@echo "  make fmt             - Format Go code"
	@echo "  make vet             - Run go vet"
	@echo "  make clean           - Clean build artifacts"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-build    - Build Docker images"
	@echo "  make docker-up       - Start all services (Neo4j, Redis, LibSQL)"
	@echo "  make docker-down     - Stop and remove all containers"
	@echo "  make docker-test     - Run integration tests in Docker"
	@echo "  make docker-logs     - View container logs"
	@echo ""
	@echo "Service Management:"
	@echo "  make services-up     - Start only infrastructure services"
	@echo "  make services-down   - Stop infrastructure services"
	@echo "  make services-status - Check service health"

# =============================================================================
# Local Development
# =============================================================================

build:
	@echo "Building GraphMem-Go..."
	go build -v ./...

test: test-unit

test-unit:
	@echo "Running unit tests..."
	go test -v -short ./pkg/graphmem/...

test-integration:
	@echo "Running integration tests..."
	@echo "Make sure services are running (make services-up)"
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi && \
	if [ -f docker.env ]; then set -a && . ./docker.env && set +a; fi && \
	go test -v -tags=integration -timeout=10m ./pkg/graphmem/...

test-all: test-unit test-integration

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./pkg/graphmem/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: fmt vet staticcheck

fmt:
	@echo "Formatting code..."
	gofmt -w ./pkg/

vet:
	@echo "Running go vet..."
	go vet ./...

staticcheck:
	@echo "Running staticcheck..."
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

clean:
	@echo "Cleaning..."
	rm -f coverage.out coverage.html
	go clean -cache -testcache

# =============================================================================
# Docker Commands
# =============================================================================

docker-build:
	@echo "Building Docker images..."
	docker-compose build

docker-up:
	@echo "Starting all services..."
	docker-compose --profile full up -d

docker-down:
	@echo "Stopping all services..."
	docker-compose --profile full down -v

docker-test:
	@echo "Running integration tests in Docker..."
	docker-compose --profile test up --build --abort-on-container-exit
	@echo ""
	@echo "Cleaning up test containers..."
	docker-compose --profile test down -v

docker-logs:
	docker-compose --profile full logs -f

docker-shell:
	@echo "Opening shell in test container..."
	docker-compose run --rm test-runner /bin/sh

# =============================================================================
# Service Management (Infrastructure Only)
# =============================================================================

services-up:
	@echo "Starting infrastructure services (Neo4j, Redis, LibSQL)..."
	docker-compose --profile local up -d
	@echo ""
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@make services-status

services-down:
	@echo "Stopping infrastructure services..."
	docker-compose --profile local down

services-status:
	@echo "Service Status:"
	@echo "=============="
	@docker-compose ps 2>/dev/null || echo "No services running"
	@echo ""
	@echo "Health Checks:"
	@echo "Neo4j:  $$(curl -s -o /dev/null -w '%{http_code}' http://localhost:7474 2>/dev/null || echo 'not running')"
	@echo "Redis:  $$(docker exec graphmem-redis redis-cli ping 2>/dev/null || echo 'not running')"
	@echo "LibSQL: $$(curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/health 2>/dev/null || echo 'not running')"

services-clean:
	@echo "Removing all service data..."
	docker-compose --profile local down -v
	docker volume rm graphmem-go_neo4j_data graphmem-go_redis_data graphmem-go_libsql_data 2>/dev/null || true

# =============================================================================
# Ollama (Local LLM)
# =============================================================================

ollama-up:
	@echo "Starting Ollama..."
	docker-compose --profile local-llm up -d ollama
	@echo "Ollama is starting. Pull a model with: make ollama-pull MODEL=llama3.2"

ollama-pull:
	@if [ -z "$(MODEL)" ]; then \
		echo "Usage: make ollama-pull MODEL=llama3.2"; \
	else \
		echo "Pulling model $(MODEL)..."; \
		docker exec graphmem-ollama ollama pull $(MODEL); \
	fi

ollama-models:
	@docker exec graphmem-ollama ollama list

# =============================================================================
# Development Shortcuts
# =============================================================================

dev: services-up
	@echo ""
	@echo "Development environment ready!"
	@echo "Run tests with: make test-integration"
	@echo ""
	@echo "Service URLs:"
	@echo "  Neo4j Browser: http://localhost:7474"
	@echo "  Redis:         redis://localhost:6379"
	@echo "  LibSQL:        http://localhost:8080"

# Quick check before committing
check: fmt vet
	@echo "Running quick checks..."
	go build ./...
	go test -short ./...
	@echo "All checks passed!"

# Full CI check
ci: lint test-unit
	@echo "CI checks passed!"

