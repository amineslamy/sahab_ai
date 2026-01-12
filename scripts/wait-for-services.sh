#!/bin/bash
# wait-for-services.sh
# Wait for all required services to be ready before running tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Wait for a TCP port to be available
wait_for_port() {
    local host=$1
    local port=$2
    local service=$3
    local timeout=${4:-60}
    local start_time=$(date +%s)
    
    log_info "Waiting for $service at $host:$port..."
    
    while ! nc -z "$host" "$port" 2>/dev/null; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout ]; then
            log_error "$service not available after ${timeout}s"
            return 1
        fi
        
        sleep 1
    done
    
    log_info "$service is available at $host:$port"
    return 0
}

# Wait for HTTP endpoint to return 200
wait_for_http() {
    local url=$1
    local service=$2
    local timeout=${3:-60}
    local start_time=$(date +%s)
    
    log_info "Waiting for $service at $url..."
    
    while ! curl -sf "$url" > /dev/null 2>&1; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout ]; then
            log_error "$service not available after ${timeout}s"
            return 1
        fi
        
        sleep 1
    done
    
    log_info "$service is available at $url"
    return 0
}

# Main service checks
main() {
    local exit_code=0
    
    log_info "Starting service availability checks..."
    
    # Check Neo4j (if configured)
    if [ -n "$NEO4J_HOST" ] && [ -n "$NEO4J_PORT" ]; then
        if ! wait_for_port "$NEO4J_HOST" "$NEO4J_PORT" "Neo4j" 120; then
            exit_code=1
        fi
    else
        log_warn "Neo4j host/port not configured, skipping..."
    fi
    
    # Check Redis (if configured)
    if [ -n "$REDIS_HOST" ] && [ -n "$REDIS_PORT" ]; then
        if ! wait_for_port "$REDIS_HOST" "$REDIS_PORT" "Redis" 60; then
            exit_code=1
        fi
    else
        log_warn "Redis host/port not configured, skipping..."
    fi
    
    # Check LibSQL/Turso (if configured)
    if [ -n "$LIBSQL_HOST" ] && [ -n "$LIBSQL_PORT" ]; then
        if ! wait_for_http "http://$LIBSQL_HOST:$LIBSQL_PORT/health" "LibSQL" 60; then
            exit_code=1
        fi
    else
        log_warn "LibSQL host/port not configured, skipping..."
    fi
    
    # Check Ollama (if configured)
    if [ -n "$OLLAMA_HOST" ] && [ -n "$OLLAMA_PORT" ]; then
        if ! wait_for_http "http://$OLLAMA_HOST:$OLLAMA_PORT/api/tags" "Ollama" 60; then
            log_warn "Ollama not available, LLM tests may be skipped"
        fi
    fi
    
    if [ $exit_code -eq 0 ]; then
        log_info "All required services are available!"
    else
        log_error "Some services failed to start"
    fi
    
    return $exit_code
}

main "$@"

