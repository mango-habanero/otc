.PHONY: help build clean fmt fmt-check lint vet test test-unit test-integration test-e2e test-coverage check all deps tidy

# Variables
BINARY_NAME=otc
BUILD_DIR=bin
GO=go
GOLANGCI_LINT=golangci-lint

# Default target
help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Code quality targets
fmt: ## Format code with gofmt and goimports
	@echo "Formatting code..."
	@gofmt -w -s .
	@goimports -w .
	@echo "Formatting complete"

fmt-check: ## Check if code is formatted
	@echo "Checking code formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)
	@echo "Code formatting check passed"

lint: ## Run golangci-lint
	@echo "Running linters..."
	@$(GOLANGCI_LINT) run ./...
	@echo "Linting complete"

vet: ## Run go vet
	@echo "Running go vet..."
	@$(GO) vet ./...
	@echo "Vet complete"

# Test targets
test: ## Run all tests with race detector
	@echo "Running tests..."
	@$(GO) test -race -v ./...
	@echo "Tests complete"

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@$(GO) test -race -v ./pkg/... ./internal/... ./cmd/...
	@echo "Unit tests complete"

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Composite targets
check: fmt-check lint vet test ## Run all pre-commit checks
	@echo "All checks passed!"

all: check build ## Run checks and build
	@echo "All tasks complete!"

# Dependency management
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@$(GO) mod download
	@echo "Dependencies downloaded"

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@$(GO) mod tidy
	@echo "Tidy complete"