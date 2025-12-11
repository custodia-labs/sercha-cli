# Sercha CLI Makefile
# Provides build, test, and development commands

.PHONY: all build build-cgo clean test lint fmt vet check clib install help

# Build configuration
BINARY_NAME := sercha
BUILD_DIR := ./dist
CMD_DIR := ./cmd/sercha
CLIB_DIR := ./clib
CLIB_BUILD_DIR := $(CLIB_DIR)/build

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Default target
all: check build

# Build without CGO (pure Go, stub implementations)
build:
	@echo "Building $(BINARY_NAME) (no CGO)..."
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Build with CGO (requires clib and system dependencies)
build-cgo: clib
	@echo "Building $(BINARY_NAME) (with CGO)..."
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Build C++ libraries
clib:
	@echo "Building C++ libraries..."
	cmake -S $(CLIB_DIR) -B $(CLIB_BUILD_DIR) -DCMAKE_BUILD_TYPE=Release
	cmake --build $(CLIB_BUILD_DIR) --config Release

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf $(CLIB_BUILD_DIR)
	go clean -cache -testcache

# Run tests
test:
	@echo "Running tests..."
	CGO_ENABLED=0 go test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	CGO_ENABLED=0 go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Run the binary
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Update dependencies
deps:
	@echo "Updating dependencies..."
	go mod tidy
	go mod verify

# Generate mocks (for testing)
mocks:
	@echo "Generating mocks..."
	go generate ./...

# Help
help:
	@echo "Sercha CLI Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build        Build without CGO (pure Go stubs)"
	@echo "  make build-cgo    Build with CGO (requires clib)"
	@echo "  make clib         Build C++ wrapper libraries"
	@echo "  make test         Run tests"
	@echo "  make test-coverage Run tests with coverage report"
	@echo "  make lint         Run golangci-lint"
	@echo "  make fmt          Format code"
	@echo "  make vet          Run go vet"
	@echo "  make check        Run all checks (fmt, vet, lint, test)"
	@echo "  make clean        Clean build artifacts"
	@echo "  make install      Install binary to GOPATH/bin"
	@echo "  make deps         Update dependencies"
	@echo "  make help         Show this help"
