.PHONY: all fmt lint test clean build test-cover ci

# Main pipeline
all: fmt lint test

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .

# Linting (golangci-lint must be installed)
lint:
	@echo "Running linter..."
	@golangci-lint run || echo "warning: golangci-lint not installed"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

# Clean up temporary files and test output
clean:
	@echo "Cleaning up..."
	@rm -f coverage.out

# Build a simple test binary (if needed)
build:
	@echo "Building CLI..."
	@go build -o bin/demo ./cmd/demo

# CI target (simulate pipeline)
ci: clean all test-cover

# Help message
help:
	@echo "Available targets:"
	@echo "  make           - run fmt, lint, test"
	@echo "  make fmt       - format code with gofmt"
	@echo "  make lint      - run golangci-lint"
	@echo "  make test      - run unit tests"
	@echo "  make test-cover - run tests with coverage report"
	@echo "  make clean     - clean coverage output and temp files"
	@echo "  make build     - build CLI test harness (cmd/demo)"
	@echo "  make ci        - run full test and coverage pipeline"