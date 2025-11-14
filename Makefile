.PHONY: all build clean test staticcheck lint fmt vet kill help

BINARY_NAME=fosilo
BUILD_DIR=.
CMD_DIR=./cmd/fosilo
GO=go
STATICCHECK=$(shell command -v staticcheck 2> /dev/null)

all: lint build

help:
	@echo "Available commands:"
	@echo "  build       - Build the dedicated server"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  staticcheck - Run staticcheck linter"
	@echo "  lint        - Run all linters"
	@echo "  fmt         - Format Go code"
	@echo "  vet         - Run go vet"
	@echo "  kill        - Kill all running processes"
	@echo "  all         - Run linters and build (default)"
	@echo ""

build:
	@echo "Building $(BINARY_NAME)..."
	@$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@$(GO) clean
	@echo "Clean complete"

test:
	@echo "Running tests..."
	@$(GO) test -v ./...

fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@echo "Format complete"

vet:
	@echo "Running go vet..."
	@$(GO) vet ./...
	@echo "Vet complete"

staticcheck:
ifndef STATICCHECK
	@echo "Error: staticcheck is not installed"
	@exit 1
endif
	@echo "Running staticcheck..."
	@staticcheck ./...
	@echo "Staticcheck complete"

lint: fmt vet staticcheck
	@echo "All linters passed"

kill:
	@echo "Killing all processes..."
	@pkill -9 fosilo || echo "No fosilo processes found"
	@echo "Kill complete"

.DEFAULT_GOAL := all
