# BirdNET-Pi Makefile

.PHONY: all build test test-verbose test-coverage test-race lint clean help
.PHONY: dev-server dev-web install-deps generate
.PHONY: build-arm64 build-arm build-pi build-all-platforms

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
BINARY_NAME=birdnet-server
BINARY_DIR=bin

# Build flags
BUILD_FLAGS=-ldflags="-s -w"

# Test flags
TEST_FLAGS=-v
COVERAGE_FLAGS=-coverprofile=coverage.out -covermode=atomic

# Directories
CMD_DIR=./cmd/server
INTERNAL_DIR=./internal/...
WEB_DIR=./web

all: test build

## Build targets

build: ## Build the server binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) $(CMD_DIR)

build-debug: ## Build with debug symbols
	@echo "Building $(BINARY_NAME) (debug)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) $(CMD_DIR)

build-arm64: ## Build for Raspberry Pi 3/4/5 (64-bit ARM)
	@echo "Building $(BINARY_NAME) for linux/arm64..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

build-arm: ## Build for Raspberry Pi Zero/older Pi (32-bit ARM)
	@echo "Building $(BINARY_NAME) for linux/arm..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm $(CMD_DIR)

build-pi: build-arm64 ## Alias for build-arm64 (most common Pi target)
	@echo "Pi build complete: $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64"

build-all-platforms: build build-arm64 build-arm ## Build for all platforms
	@echo "All platform builds complete:"
	@ls -la $(BINARY_DIR)/

## Test targets

test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) $(INTERNAL_DIR)

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	$(GOTEST) $(TEST_FLAGS) $(INTERNAL_DIR)

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GOTEST) $(COVERAGE_FLAGS) $(INTERNAL_DIR)
	@echo "Coverage report generated: coverage.out"
	@$(GOCMD) tool cover -func=coverage.out

test-coverage-html: test-coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	$(GOTEST) -race $(INTERNAL_DIR)

test-short: ## Run only short tests
	@echo "Running short tests..."
	$(GOTEST) -short $(INTERNAL_DIR)

test-db: ## Run database tests only
	@echo "Running database tests..."
	$(GOTEST) $(TEST_FLAGS) ./internal/db/...

test-ws: ## Run WebSocket tests only
	@echo "Running WebSocket tests..."
	$(GOTEST) $(TEST_FLAGS) ./internal/ws/...

test-api: ## Run API handler tests only
	@echo "Running API tests..."
	$(GOTEST) $(TEST_FLAGS) ./internal/api/...

test-mlclient: ## Run ML client tests only
	@echo "Running ML client tests..."
	$(GOTEST) $(TEST_FLAGS) ./internal/mlclient/...

test-monitor: ## Run monitor tests only
	@echo "Running monitor tests..."
	$(GOTEST) $(TEST_FLAGS) ./internal/monitor/...

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem $(INTERNAL_DIR)

## Code quality

lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GOFMT) -s -w .

fmt-check: ## Check if code is formatted
	@echo "Checking code format..."
	@test -z "$$($(GOFMT) -l .)" || (echo "Code not formatted. Run 'make fmt'" && exit 1)

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet $(INTERNAL_DIR)

## Dependencies

install-deps: ## Install Go dependencies
	@echo "Installing Go dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

## Code generation

generate: ## Generate sqlc code
	@echo "Generating sqlc code..."
	@if command -v sqlc >/dev/null 2>&1; then \
		cd internal/db && sqlc generate; \
	else \
		echo "sqlc not installed. Install with: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"; \
	fi

## Database migrations

migrate-up: ## Run database migrations
	@echo "Running migrations..."
	@if command -v migrate >/dev/null 2>&1; then \
		migrate -database "sqlite3://$(DB_PATH)" -path migrations up; \
	else \
		echo "migrate not installed. Install with:"; \
		echo "  go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
	fi

migrate-down: ## Rollback last migration
	@echo "Rolling back migration..."
	@if command -v migrate >/dev/null 2>&1; then \
		migrate -database "sqlite3://$(DB_PATH)" -path migrations down 1; \
	else \
		echo "migrate not installed"; \
	fi

migrate-create: ## Create new migration (usage: make migrate-create NAME=migration_name)
	@echo "Creating migration: $(NAME)"
	@if command -v migrate >/dev/null 2>&1; then \
		migrate create -ext sql -dir migrations -seq $(NAME); \
	else \
		echo "migrate not installed"; \
	fi

## Development

dev-server: ## Run development server with hot reload
	@echo "Starting development server..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Running without hot reload..."; \
		$(GOCMD) run $(CMD_DIR); \
	fi

dev-web: ## Run web development server
	@echo "Starting web development server..."
	cd $(WEB_DIR) && npm run dev

install-web: ## Install web dependencies
	@echo "Installing web dependencies..."
	cd $(WEB_DIR) && npm install

build-web: ## Build web assets
	@echo "Building web assets..."
	cd $(WEB_DIR) && npm run build

## Cleanup

clean: ## Remove build artifacts
	@echo "Cleaning..."
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html
	rm -rf $(WEB_DIR)/dist
	rm -rf $(WEB_DIR)/node_modules

## Help

help: ## Show this help
	@echo "BirdNET-Pi Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
