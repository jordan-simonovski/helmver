BINARY   := helmver
MODULE   := github.com/jordan-simonovski/helmver
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X '$(MODULE)/cmd.version=$(VERSION)'

# Cross-compile targets
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

DIST_DIR := dist

.PHONY: all build test test-unit test-e2e test-acceptance lint fmt vet clean cross-compile help

all: lint test build ## Run lint, test, and build

build: ## Build for current OS/arch
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test: ## Run all tests
	go test -v -count=1 -race ./...

test-unit: ## Run unit tests only (no e2e)
	go test -v -count=1 -race ./internal/...

test-e2e: ## Run end-to-end tests only
	go test -v -count=1 -race -run TestE2E .

test-acceptance: ## Run acceptance tests against fixture charts
	go test -v -count=1 -race ./test/acceptance/...

lint: vet ## Run golangci-lint and go vet
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping (install: https://golangci-lint.run/welcome/install/)"; \
	fi

vet: ## Run go vet
	go vet ./...

fmt: ## Run gofmt and goimports
	gofmt -s -w .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not installed, skipping"; \
	fi

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf $(DIST_DIR)

cross-compile: clean ## Cross-compile for all platforms
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output=$(DIST_DIR)/$(BINARY)-$${GOOS}-$${GOARCH}; \
		if [ "$${GOOS}" = "windows" ]; then output=$${output}.exe; fi; \
		echo "Building $${output}..."; \
		GOOS=$${GOOS} GOARCH=$${GOARCH} go build -ldflags "$(LDFLAGS)" -o $${output} . || exit 1; \
	done
	@echo "Done. Binaries in $(DIST_DIR)/"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
