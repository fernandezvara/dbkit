# Makefile for dbkit - PostgreSQL integration testing
# Supports both Docker and Podman container runtimes

# Include PostgreSQL versions
include scripts/versions.mk

.PHONY: help detect-runtime start stop test test-all test-pg18 test-pg17 test-pg16 test-pg15 test-pg14 test-pg13 clean wait-healthy

# Detect container runtime (docker or podman)
DOCKER := $(shell command -v docker 2> /dev/null)
PODMAN := $(shell command -v podman 2> /dev/null)

ifdef DOCKER
    CONTAINER_RUNTIME := docker
else ifdef PODMAN
    CONTAINER_RUNTIME := podman
else
    $(error No container runtime found. Please install docker or podman)
endif

# Detect compose tool (docker-compose, docker compose, or podman-compose)
DOCKER_COMPOSE := $(shell command -v docker-compose 2> /dev/null)
DOCKER_COMPOSE_PLUGIN := $(shell docker compose version 2> /dev/null && echo "docker compose")
PODMAN_COMPOSE := $(shell command -v podman-compose 2> /dev/null)

ifdef DOCKER_COMPOSE
    COMPOSE_CMD := docker-compose
else ifdef DOCKER_COMPOSE_PLUGIN
    COMPOSE_CMD := docker compose
else ifdef PODMAN_COMPOSE
    COMPOSE_CMD := podman-compose
else
    $(error No compose tool found. Please install docker-compose, docker compose plugin, or podman-compose)
endif

# PostgreSQL versions and their ports are now in versions.mk

# Default test timeout
TEST_TIMEOUT := 5m

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "dbkit - PostgreSQL Integration Testing"
	@echo ""
	@echo "Detected runtime: $(CONTAINER_RUNTIME)"
	@echo "Detected compose: $(COMPOSE_CMD)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  help                 Show this help message"
	@echo "  detect-runtime       Show runtime and compose tool"
	@echo "  start                Start all PostgreSQL containers"
	@echo "  stop                 Stop all PostgreSQL containers"
	@echo "  clean                Stop containers and remove volumes"
	@echo "  wait-healthy         Wait for containers to be healthy"
	@echo "  test                 Run unit tests (no database)"
	@echo "  test-all             Run tests on all PostgreSQL versions"
	@echo "  test-pg18            Run tests against PostgreSQL 18"
	@echo "  test-pg17            Run tests against PostgreSQL 17"
	@echo "  test-pg16            Run tests against PostgreSQL 16"
	@echo "  test-pg15            Run tests against PostgreSQL 15"
	@echo "  test-pg14            Run tests against PostgreSQL 14"
	@echo "  test-pg13            Run tests against PostgreSQL 13"
	@echo "  logs                 Show logs from all containers"
	@echo "  status               Show status of containers"
	@echo "  test-coverage        Run tests with coverage"
	@echo "  bench                Run benchmark tests"

detect-runtime: ## Show detected container runtime and compose tool
	@echo "Container runtime: $(CONTAINER_RUNTIME)"
	@echo "Compose command: $(COMPOSE_CMD)"

start: ## Start all PostgreSQL containers
	@echo "$(GREEN)Starting PostgreSQL containers...$(NC)"
	$(COMPOSE_CMD) up -d
	@echo "$(GREEN)Waiting for containers to be healthy...$(NC)"
	@$(MAKE) wait-healthy
	@echo "$(GREEN)All PostgreSQL containers are ready!$(NC)"

stop: ## Stop all PostgreSQL containers
	@echo "$(YELLOW)Stopping PostgreSQL containers...$(NC)"
	$(COMPOSE_CMD) down
	@echo "$(GREEN)All containers stopped.$(NC)"

clean: ## Stop containers and remove volumes
	@echo "$(RED)Stopping containers and removing volumes...$(NC)"
	$(COMPOSE_CMD) down -v --remove-orphans
	@echo "$(GREEN)Cleanup complete.$(NC)"

wait-healthy: ## Wait for all PostgreSQL containers to be healthy
	@echo "Waiting for PostgreSQL 18..."
	@until $(CONTAINER_RUNTIME) exec dbkit-postgres-18 pg_isready -U postgres 2>/dev/null; do sleep 1; done
	@echo "Waiting for PostgreSQL 17..."
	@until $(CONTAINER_RUNTIME) exec dbkit-postgres-17 pg_isready -U postgres 2>/dev/null; do sleep 1; done
	@echo "Waiting for PostgreSQL 16..."
	@until $(CONTAINER_RUNTIME) exec dbkit-postgres-16 pg_isready -U postgres 2>/dev/null; do sleep 1; done
	@echo "Waiting for PostgreSQL 15..."
	@until $(CONTAINER_RUNTIME) exec dbkit-postgres-15 pg_isready -U postgres 2>/dev/null; do sleep 1; done
	@echo "Waiting for PostgreSQL 14..."
	@until $(CONTAINER_RUNTIME) exec dbkit-postgres-14 pg_isready -U postgres 2>/dev/null; do sleep 1; done
	@echo "Waiting for PostgreSQL 13..."
	@until $(CONTAINER_RUNTIME) exec dbkit-postgres-13 pg_isready -U postgres 2>/dev/null; do sleep 1; done
	@echo "$(GREEN)All PostgreSQL containers are healthy!$(NC)"

test: ## Run unit tests (no database required)
	@echo "$(GREEN)Running unit tests...$(NC)"
	go test -v -race -timeout $(TEST_TIMEOUT) ./...

test-all: start ## Run integration tests against all PostgreSQL versions
	@echo "$(GREEN)Running integration tests against all PostgreSQL versions...$(NC)"
	@$(MAKE) test-pg18
	@$(MAKE) test-pg17
	@$(MAKE) test-pg16
	@$(MAKE) test-pg15
	@$(MAKE) test-pg14
	@$(MAKE) test-pg13
	@echo "$(GREEN)All integration tests passed!$(NC)"

test-pg18: ## Run integration tests against PostgreSQL 18
	@echo "$(GREEN)Testing against PostgreSQL $(PG_18_VERSION)...$(NC)"
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5418/dbkit_test?sslmode=disable" \
		go test -v -race -parallel 1 -timeout $(TEST_TIMEOUT) ./...

test-pg17: ## Run integration tests against PostgreSQL 17
	@echo "$(GREEN)Testing against PostgreSQL $(PG_17_VERSION)...$(NC)"
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5417/dbkit_test?sslmode=disable" \
		go test -v -race -parallel 1 -timeout $(TEST_TIMEOUT) ./...

test-pg16: ## Run integration tests against PostgreSQL 16
	@echo "$(GREEN)Testing against PostgreSQL $(PG_16_VERSION)...$(NC)"
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5416/dbkit_test?sslmode=disable" \
		go test -v -race -parallel 1 -timeout $(TEST_TIMEOUT) ./...

test-pg15: ## Run integration tests against PostgreSQL 15
	@echo "$(GREEN)Testing against PostgreSQL $(PG_15_VERSION)...$(NC)"
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5415/dbkit_test?sslmode=disable" \
		go test -v -race -parallel 1 -timeout $(TEST_TIMEOUT) ./...

test-pg14: ## Run integration tests against PostgreSQL 14
	@echo "$(GREEN)Testing against PostgreSQL $(PG_14_VERSION)...$(NC)"
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5414/dbkit_test?sslmode=disable" \
		go test -v -race -parallel 1 -timeout $(TEST_TIMEOUT) ./...

test-pg13: ## Run integration tests against PostgreSQL 13
	@echo "$(GREEN)Testing against PostgreSQL $(PG_13_VERSION)...$(NC)"
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5413/dbkit_test?sslmode=disable" \
		go test -v -race -parallel 1 -timeout $(TEST_TIMEOUT) ./...

logs: ## Show logs from all PostgreSQL containers
	$(COMPOSE_CMD) logs -f

status: ## Show status of PostgreSQL containers
	$(COMPOSE_CMD) ps

# Start a specific PostgreSQL version
start-pg%:
	@echo "$(GREEN)Starting PostgreSQL $*...$(NC)"
	$(COMPOSE_CMD) up -d postgres-$*
	@echo "Waiting for PostgreSQL $* to be healthy..."
	@until $(CONTAINER_RUNTIME) exec dbkit-postgres-$* pg_isready -U postgres 2>/dev/null; do sleep 1; done
	@echo "$(GREEN)PostgreSQL $* is ready!$(NC)"

# Stop a specific PostgreSQL version
stop-pg%:
	@echo "$(YELLOW)Stopping PostgreSQL $*...$(NC)"
	$(COMPOSE_CMD) stop postgres-$*
	@echo "$(GREEN)PostgreSQL $* stopped.$(NC)"

# Run tests with coverage
test-coverage: start ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5417/dbkit_test?sslmode=disable" \
		go test -v -race -coverprofile=coverage.out -timeout $(TEST_TIMEOUT) ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# Benchmark tests
bench: start ## Run benchmark tests
	@echo "$(GREEN)Running benchmarks against PostgreSQL $(PG_17_VERSION)...$(NC)"
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5417/dbkit_test?sslmode=disable" \
		go test -v -bench=. -benchmem -timeout $(TEST_TIMEOUT) ./...
