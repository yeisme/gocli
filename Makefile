# Makefile for gocli

# Variables
APP_NAME := gocli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | awk '{print $$3}')
PLATFORM := $(shell go env GOOS)/$(shell go env GOARCH)
MODIFIED := $(shell if [ -n "$$(git status --porcelain 2>/dev/null)" ]; then echo "true"; else echo "false"; fi)
MOD_SUM := $(shell if [ -f go.sum ]; then shasum -a 256 go.sum | cut -d' ' -f1 | cut -c1-16; else echo "unknown"; fi)

# Directories
BUILD_DIR := ./bin
SRC_DIR := ./cli

# LDFLAGS
LDFLAGS := -X 'github.com/yeisme/gocli/pkg/utils/version.Version=$(VERSION)' \
           -X 'github.com/yeisme/gocli/pkg/utils/version.GitCommit=$(GIT_COMMIT)' \
           -X 'github.com/yeisme/gocli/pkg/utils/version.BuildDate=$(BUILD_DATE)' \
           -X 'github.com/yeisme/gocli/pkg/utils/version.GoVersion=$(GO_VERSION)' \
           -X 'github.com/yeisme/gocli/pkg/utils/version.Platform=$(PLATFORM)' \
           -X 'github.com/yeisme/gocli/pkg/utils/version.Modified=$(MODIFIED)' \
           -X 'github.com/yeisme/gocli/pkg/utils/version.ModSum=$(MOD_SUM)'

# Release LDFLAGS (strip debug info)
RELEASE_LDFLAGS := $(LDFLAGS) -s -w

.PHONY: all build release clean test version help

# Default target
all: build

# Build development version
build:
	@echo "Building $(APP_NAME)..."
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Platform: $(PLATFORM)"
	@echo "Modified: $(MODIFIED)"
	@echo ""
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(SRC_DIR)
	@echo ""
	@echo "Build successful! Output: $(BUILD_DIR)/$(APP_NAME)"
	@echo "Size: $$(du -h $(BUILD_DIR)/$(APP_NAME) | cut -f1)"

# Build release version
release:
	@echo "Building $(APP_NAME) (release)..."
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo ""
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(RELEASE_LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(SRC_DIR)
	@echo ""
	@echo "Release build successful! Output: $(BUILD_DIR)/$(APP_NAME)"
	@echo "Size: $$(du -h $(BUILD_DIR)/$(APP_NAME) | cut -f1)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...
