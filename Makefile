GO := $(shell command -v go)

SERVER_BINARY := cmd/server/server
AGENT_BINARY  := cmd/agent/agent
LINT_BINARY   := cmd/staticlint/staticlint

# Build info printed by the server and the agent at start, override as
# `make build BUILD_VERSION=v1.0.0`.
BUILD_VERSION ?= $(or $(shell git describe --tags --always 2>/dev/null),N/A)
BUILD_DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_COMMIT  ?= $(or $(shell git rev-parse --short HEAD 2>/dev/null),N/A)
LDFLAGS := -X main.buildVersion=$(BUILD_VERSION) \
	-X main.buildDate=$(BUILD_DATE) \
	-X main.buildCommit=$(BUILD_COMMIT)

# Iteration checked by `make iter`, override as `make iter ITER=11`.
ITER ?= 14

SERVER_PORT ?= 4000
TEMP_FILE   ?= /tmp/metrics-db-test.json
# Signing key for iterations 14+; metricstest hands it to the agent and server.
KEY         ?= metrics-test-key
export ADDRESS := localhost:$(SERVER_PORT)

PG_CONTAINER ?= praktikum-pg
PG_IMAGE     ?= postgres:16-alpine
PG_PORT      ?= 5433
DATABASE_DSN ?= postgres://postgres:postgres@localhost:$(PG_PORT)/praktikum?sslmode=disable

# Memory profiling, see `make profile`.
PPROF_ADDRESS ?= localhost:6060
PROFILE       ?= base
LOAD_DATA     := profiles/testdata
LOAD_REQUESTS ?= 20000

.DEFAULT_GOAL := help
.PHONY: help build build-server build-agent build-lint generate test test-db cover bench vet lint fmt iter profile db-up db-reset db-down clean

help: ## Show this help
	@grep -E '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: build-server build-agent ## Build both binaries

build-server: ## Build the server binary
	$(GO) build -ldflags '$(LDFLAGS)' -o $(SERVER_BINARY) ./cmd/server

build-agent: ## Build the agent binary
	$(GO) build -ldflags '$(LDFLAGS)' -o $(AGENT_BINARY) ./cmd/agent

build-lint: ## Build the staticlint multichecker
	$(GO) build -o $(LINT_BINARY) ./cmd/staticlint

generate: ## Generate Reset methods for structs marked // generate:reset
	$(GO) run ./cmd/reset

test: ## Run unit tests under the race detector (database tests are skipped)
	$(GO) test -race ./...

test-db: db-up ## Run unit tests including the ones that need postgres
	TEST_DATABASE_DSN='$(DATABASE_DSN)' $(GO) test -race ./...

cover: ## Run unit tests and print the total coverage
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -1

bench: ## Run the benchmarks
	$(GO) test -run='^$$' -bench=. -benchmem ./...

vet: ## Run go vet
	$(GO) vet ./...

lint: build-lint ## Run the staticlint multichecker on the whole project
	./$(LINT_BINARY) ./...

fmt: ## Format the source tree
	$(GO) fmt ./...

iter: build ## Run metricstest for one iteration (make iter ITER=10)
	metricstest -test.v -test.run='^TestIteration$(ITER)[AB]?$$' \
		-agent-binary-path=$(AGENT_BINARY) \
		-binary-path=$(SERVER_BINARY) \
		-database-dsn='$(DATABASE_DSN)' \
		-file-storage-path=$(TEMP_FILE) \
		-key='$(KEY)' \
		-server-port=$(SERVER_PORT) \
		-source-path=.

profile: build-server ## Load the server with hey and save its memory profile (make profile PROFILE=result)
	gzip -c $(LOAD_DATA)/updates.json > $(LOAD_DATA)/updates.json.gz
	@$(SERVER_BINARY) -f '' -l /dev/null -pprof $(PPROF_ADDRESS) & pid=$$!; \
	sleep 1; \
	echo "POST /updates/"; \
	hey -n $(LOAD_REQUESTS) -c 50 -m POST -T application/json -H 'Content-Encoding: gzip' \
		-D $(LOAD_DATA)/updates.json.gz http://$(ADDRESS)/updates/ | grep -E 'Requests/sec|\[200\]'; \
	echo "POST /value/"; \
	hey -n $(LOAD_REQUESTS) -c 50 -m POST -T application/json \
		-D $(LOAD_DATA)/value.json http://$(ADDRESS)/value/ | grep -E 'Requests/sec|\[200\]'; \
	echo "GET /"; \
	hey -n $(LOAD_REQUESTS) -c 50 http://$(ADDRESS)/ | grep -E 'Requests/sec|\[200\]'; \
	curl -s -o profiles/$(PROFILE).pprof http://$(PPROF_ADDRESS)/debug/pprof/allocs; \
	kill $$pid
	@echo "saved profiles/$(PROFILE).pprof"

db-up: ## Start the local postgres used by iterations 10+
	@docker inspect -f '{{.State.Running}}' $(PG_CONTAINER) 2>/dev/null | grep -q true \
		|| docker run -d --rm --name $(PG_CONTAINER) \
			-e POSTGRES_USER=postgres \
			-e POSTGRES_PASSWORD=postgres \
			-e POSTGRES_DB=praktikum \
			-p $(PG_PORT):5432 $(PG_IMAGE)
	@until docker exec $(PG_CONTAINER) pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
	@echo "postgres ready on localhost:$(PG_PORT)"

db-reset: db-up ## Empty the metrics table before an autotest run
	@docker exec $(PG_CONTAINER) psql -U postgres -d praktikum -c 'TRUNCATE metrics;' >/dev/null
	@echo "metrics table emptied"

db-down: ## Stop the local postgres
	-docker rm -f $(PG_CONTAINER)

clean: ## Remove built binaries
	rm -f $(SERVER_BINARY) $(AGENT_BINARY) $(LINT_BINARY)
