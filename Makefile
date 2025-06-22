# Flux Language Makefile

# Variables
GO_VERSION := 1.23.2
BINARY_NAME := flux
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)

# Build directories
BUILD_DIR := dist
SCRIPTS_DIR := scripts

# Default target
.PHONY: all
all: clean build

# Help target
.PHONY: help
help: ## Show this help message
	@echo "Flux Language Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Clean build artifacts
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean

# Build for current platform
.PHONY: build
build: ## Build binary for current platform
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
.PHONY: build-all
build-all: ## Build binaries for all platforms
	@echo "Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	@echo "Building for linux/amd64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	
	# Linux ARM64
	@echo "Building for linux/arm64..."
	@GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	
	# macOS AMD64
	@echo "Building for darwin/amd64..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	
	# macOS ARM64
	@echo "Building for darwin/arm64..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	# Windows AMD64
	@echo "Building for windows/amd64..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	
	@echo "All builds complete!"
	@ls -la $(BUILD_DIR)/

# Create archives for distribution
.PHONY: package
package: build-all ## Create distribution packages
	@echo "Creating distribution packages..."
	@cd $(BUILD_DIR) && \
	for binary in $(BINARY_NAME)-linux-*; do \
		echo "Packaging $$binary..."; \
		tar -czf $$binary.tar.gz $$binary; \
	done
	
	@cd $(BUILD_DIR) && \
	for binary in $(BINARY_NAME)-darwin-*; do \
		echo "Packaging $$binary..."; \
		tar -czf $$binary.tar.gz $$binary; \
	done
	
	@cd $(BUILD_DIR) && \
	for binary in $(BINARY_NAME)-windows-*.exe; do \
		echo "Packaging $$binary..."; \
		zip $${binary%.*}.zip $$binary; \
	done
	
	@echo "Packaging complete!"
	@ls -la $(BUILD_DIR)/*.{tar.gz,zip} 2>/dev/null || true

# Development commands
.PHONY: dev
dev: ## Build and run for development
	@go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Development build ready: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: install
install: build ## Install binary to local Go bin
	@echo "Installing $(BINARY_NAME) to Go bin..."
	@go install -ldflags="$(LDFLAGS)" .
	@echo "Installed: $(shell go env GOPATH)/bin/$(BINARY_NAME)"

# Testing
.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Code quality
.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

.PHONY: lint
lint: fmt vet ## Run all linting

.PHONY: check
check: lint test ## Run all checks

# Dependencies
.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# Version management
.PHONY: version
version: ## Show current version
	@$(SCRIPTS_DIR)/version.sh current

.PHONY: version-suggest
version-suggest: ## Suggest next version
	@$(SCRIPTS_DIR)/version.sh suggest

.PHONY: version-create
version-create: ## Create new version (usage: make version-create VERSION=1.2.3)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make version-create VERSION=1.2.3"; \
		exit 1; \
	fi
	@$(SCRIPTS_DIR)/version.sh create $(VERSION)

# Example commands
.PHONY: example
example: build ## Build and run example
	@echo "Building example..."
	@mkdir -p examples
	@echo 'print("Hello, Flux!")' > examples/hello.flux
	@$(BUILD_DIR)/$(BINARY_NAME) run examples/hello.flux

# Documentation
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@go doc ./...

# Release preparation
.PHONY: release-prep
release-prep: clean check build-all package ## Prepare for release
	@echo "Release preparation complete!"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Date: $(DATE)"
	@echo ""
	@echo "Artifacts:"
	@ls -la $(BUILD_DIR)/

# Docker support (if needed in future)
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Docker support not implemented yet"

# Benchmarks
.PHONY: bench
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Security
.PHONY: security
security: ## Run security checks
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Performance profiling
.PHONY: profile-cpu
profile-cpu: build ## Run CPU profiling
	@echo "CPU profiling not implemented yet"

.PHONY: profile-mem
profile-mem: build ## Run memory profiling
	@echo "Memory profiling not implemented yet"

# Utilities
.PHONY: size
size: build-all ## Show binary sizes
	@echo "Binary sizes:"
	@ls -lah $(BUILD_DIR)/$(BINARY_NAME)* | awk '{print $$5 "\t" $$9}'

.PHONY: info
info: ## Show build information
	@echo "Build Information:"
	@echo "  Go Version: $(shell go version)"
	@echo "  Project Version: $(VERSION)"
	@echo "  Commit: $(COMMIT)"
	@echo "  Build Date: $(DATE)"
	@echo "  Build Flags: $(LDFLAGS)"
	@echo "  Current Platform: $(shell go env GOOS)/$(shell go env GOARCH)" 