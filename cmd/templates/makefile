# Project variables
BINARY_NAME ?= app
BUILD_DIR ?= ./bin
MAIN_PATH ?= .
GO_VERSION ?= 1.21

.PHONY: help build run dev test test-cover lint fmt clean deps release install-tools docker-build docker-run list

# Default target
help: list

# List all available targets
list:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application"
	@echo "  dev          - Run in development mode with hot reload"
	@echo "  test         - Run tests"
	@echo "  test-cover   - Run tests with coverage"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  release      - Build release version"
	@echo "  install-tools - Install development tools"

# Build the application
build:
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Run the application
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

# Run in development mode with hot reload
dev:
	@if ! command -v air > /dev/null; then \
		go install github.com/cosmtrek/air@latest; \
	fi
	@air

# Run tests
test:
	@go test -v ./...

# Run tests with coverage
test-cover:
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	@if ! command -v golangci-lint > /dev/null; then \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run

# Format code
fmt:
	@go fmt ./...

# Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean -cache -modcache -testcache

# Download and tidy dependencies
deps:
	@go mod download
	@go mod tidy

# Build release version
release:
	@if ! command -v goreleaser > /dev/null; then \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	@goreleaser build --snapshot --clean --single-target

# Install development tools
install-tools:
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/goreleaser/goreleaser@latest
