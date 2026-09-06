GO := $(shell command -v go)

SERVER_BINARY := cmd/server/server
AGENT_BINARY  := cmd/agent/agent

# Iteration checked by `make iter`, override as `make iter ITER=11`.
ITER ?= 13

SERVER_PORT ?= 4000
TEMP_FILE   ?= /tmp/metrics-db-test.json
export ADDRESS := localhost:$(SERVER_PORT)

PG_CONTAINER ?= praktikum-pg
PG_IMAGE     ?= postgres:16-alpine
PG_PORT      ?= 5433
DATABASE_DSN ?= postgres://postgres:postgres@localhost:$(PG_PORT)/praktikum?sslmode=disable

.DEFAULT_GOAL := help
.PHONY: help build build-server build-agent test test-db vet fmt iter db-up db-down clean

help: ## Show this help
	@grep -E '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: build-server build-agent ## Build both binaries

build-server: ## Build the server binary
	$(GO) build -o $(SERVER_BINARY) ./cmd/server

build-agent: ## Build the agent binary
	$(GO) build -o $(AGENT_BINARY) ./cmd/agent

test: ## Run unit tests (database tests are skipped)
	$(GO) test ./...

test-db: db-up ## Run unit tests including the ones that need postgres
	TEST_DATABASE_DSN='$(DATABASE_DSN)' $(GO) test ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format the source tree
	$(GO) fmt ./...

iter: build ## Run metricstest for one iteration (make iter ITER=10)
	metricstest -test.v -test.run='^TestIteration$(ITER)[AB]?$$' \
		-agent-binary-path=$(AGENT_BINARY) \
		-binary-path=$(SERVER_BINARY) \
		-database-dsn='$(DATABASE_DSN)' \
		-file-storage-path=$(TEMP_FILE) \
		-server-port=$(SERVER_PORT) \
		-source-path=.

db-up: ## Start the local postgres used by iterations 10+
	@docker inspect -f '{{.State.Running}}' $(PG_CONTAINER) 2>/dev/null | grep -q true \
		|| docker run -d --rm --name $(PG_CONTAINER) \
			-e POSTGRES_USER=postgres \
			-e POSTGRES_PASSWORD=postgres \
			-e POSTGRES_DB=praktikum \
			-p $(PG_PORT):5432 $(PG_IMAGE)
	@until docker exec $(PG_CONTAINER) pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
	@echo "postgres ready on localhost:$(PG_PORT)"

db-down: ## Stop the local postgres
	-docker rm -f $(PG_CONTAINER)

clean: ## Remove built binaries
	rm -f $(SERVER_BINARY) $(AGENT_BINARY)
