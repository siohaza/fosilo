.PHONY: all build clean test staticcheck lint fmt vet kill help goreleaser-docker release release-dry-run

BINARY_NAME=fosilo
BUILD_DIR=.
CMD_DIR=./cmd/fosilo
GO=go
STATICCHECK=$(shell command -v staticcheck 2> /dev/null)

GO_RELEASER_CROSS_VERSION=v1.25.7
DOCKER_IMAGE=fosilo-goreleaser-cross

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
	@echo "  kill              - Kill all running processes"
	@echo "  all               - Run linters and build (default)"
	@echo "  goreleaser-docker - Build Docker image for cross-compilation"
	@echo "  release           - Run GoReleaser release"
	@echo "  release-dry-run   - Run GoReleaser release of dirty builds"
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

goreleaser-docker:
	@echo "Building GoReleaser Docker image..."
	@docker build -f Dockerfile.goreleaser \
		--build-arg GO_RELEASER_CROSS_VERSION=$(GO_RELEASER_CROSS_VERSION) \
		-t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

release: goreleaser-docker
	@docker run --rm \
		-v "$(CURDIR):/go/src/github.com/siohaza/fosilo" \
		-w /go/src/github.com/siohaza/fosilo \
		-e CGO_ENABLED=1 \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
		$(DOCKER_IMAGE) \
		release --clean

release-dry-run: goreleaser-docker
	@docker run --rm \
		-v "$(CURDIR):/go/src/github.com/siohaza/fosilo" \
		-w /go/src/github.com/siohaza/fosilo \
		-e CGO_ENABLED=1 \
		$(DOCKER_IMAGE) \
		release --snapshot --clean

.DEFAULT_GOAL := all
