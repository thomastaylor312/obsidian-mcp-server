.PHONY: help generate build test test-unit test-e2e clean fmt lint tidy run

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Project variables
BINARY_NAME=obsidian-mcp-server
BUILD_DIR=bin
PKG_DIR=pkg
INTERNAL_DIR=internal

generate: ## Generate HTTP client from OpenAPI spec
	@echo "🔧 Generating HTTP client from OpenAPI spec..."
	@mkdir -p $(PKG_DIR)/obsidian
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate types -package obsidian openapi.yaml > $(PKG_DIR)/obsidian/types.go
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate client -package obsidian openapi.yaml > $(PKG_DIR)/obsidian/client.go

build: generate ## Build the binary
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/obsidian-mcp-server/main.go

test: test-unit ## Run all tests

test-unit: ## Run unit tests
	@echo "🧪 Running unit tests..."
	go test -v ./internal/... ./pkg/... -cover

test-e2e: build ## Run end-to-end tests (requires local Obsidian server)
	@echo "🔍 Running end-to-end tests..."
	@echo "⚠️  Make sure Obsidian Local REST API plugin is running on localhost:27123"
	@echo "⚠️  Set OBSIDIAN_API_TOKEN environment variable"
	go test -v ./test/e2e/... -tags=e2e

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf $(PKG_DIR)/obsidian/types.go $(PKG_DIR)/obsidian/client.go
	go clean

fmt: ## Format code
	@echo "📝 Formatting code..."
	gofumpt -w .

lint: ## Run linter
	@echo "🔍 Running linter..."
	golangci-lint run

tidy: ## Tidy dependencies
	@echo "📦 Tidying dependencies..."
	go mod tidy

run: build ## Build and run the server
	@echo "🚀 Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run in development mode (with auto-reload when files change)
	@echo "🔄 Running in development mode..."
	go run cmd/obsidian-mcp-server/main.go