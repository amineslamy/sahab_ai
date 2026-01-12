# GraphMem-Go Dockerfile
# Multi-stage build for optimal image size and security

# =============================================================================
# Stage 1: Build Stage
# =============================================================================
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary (no C dependencies)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /graphmem ./...

# Build test binary for integration tests
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -tags=integration -o /graphmem-tests ./pkg/graphmem/

# =============================================================================
# Stage 2: Test Runner Image
# =============================================================================
FROM golang:1.22-alpine AS test-runner

# Install test dependencies
RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Default command runs all tests
CMD ["go", "test", "-v", "-tags=integration", "./pkg/graphmem/..."]

# =============================================================================
# Stage 3: Production Image
# =============================================================================
FROM alpine:3.19 AS production

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -g 1000 graphmem && \
    adduser -u 1000 -G graphmem -s /bin/sh -D graphmem

# Set working directory
WORKDIR /app

# Copy built binary from builder
COPY --from=builder /graphmem /app/graphmem
COPY --from=builder /graphmem-tests /app/graphmem-tests

# Copy any necessary config files
COPY env.example /app/env.example

# Change ownership
RUN chown -R graphmem:graphmem /app

# Switch to non-root user
USER graphmem

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default port (if running as a server)
EXPOSE 8080

# Default command
CMD ["/app/graphmem"]

# =============================================================================
# Stage 4: Integration Test Image
# =============================================================================
FROM golang:1.22-alpine AS integration-tests

# Install dependencies
RUN apk add --no-cache ca-certificates tzdata curl netcat-openbsd bash

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy wait-for-it script
COPY scripts/wait-for-services.sh /usr/local/bin/wait-for-services
RUN chmod +x /usr/local/bin/wait-for-services

# Default: run integration tests
CMD ["/bin/bash", "-c", "wait-for-services && go test -v -tags=integration -timeout=10m ./pkg/graphmem/..."]

