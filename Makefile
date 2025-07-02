# Variables
BINARY_NAME=download-bucket
BINARY_PATH=./$(BINARY_NAME)
BUILD_FLAGS=-ldflags="-s -w"
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	go build $(BUILD_FLAGS) -o $(BINARY_NAME) .

# Build for multiple platforms
.PHONY: build-all
build-all:
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Run tests
.PHONY: test
test:
	go test -v ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*

# Install the binary to GOPATH/bin
.PHONY: install
install: build
	go install .

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Run the application with help
.PHONY: run-help
run-help: build
	$(BINARY_PATH) --help

# Example: Clone from S3 (requires credentials)
.PHONY: example-s3
example-s3: build
	@echo "Example: Clone from S3 bucket"
	@echo "Usage: make example-s3 BUCKET=my-bucket PREFIX=folder/ DEST=./local-folder ACCESS_KEY=xxx SECRET_KEY=yyy"
	@if [ -z "$(BUCKET)" ] || [ -z "$(ACCESS_KEY)" ] || [ -z "$(SECRET_KEY)" ]; then \
		echo "Please provide BUCKET, ACCESS_KEY, and SECRET_KEY variables"; \
		exit 1; \
	fi
	$(BINARY_PATH) clone \
		--access-key=$(ACCESS_KEY) \
		--secret-key=$(SECRET_KEY) \
		--verbose \
		s3://$(BUCKET)/$(PREFIX) $(DEST)

# Example: Configure AWS provider
.PHONY: config-aws
config-aws: build
	@echo "Configuring AWS provider"
	@echo "Usage: make config-aws ACCESS_KEY=xxx SECRET_KEY=yyy REGION=us-west-2 BUCKET=my-bucket"
	@if [ -z "$(ACCESS_KEY)" ] || [ -z "$(SECRET_KEY)" ]; then \
		echo "Please provide ACCESS_KEY and SECRET_KEY variables"; \
		exit 1; \
	fi
	$(BINARY_PATH) config set aws \
		--access-key=$(ACCESS_KEY) \
		--secret-key=$(SECRET_KEY) \
		--region=$(if $(REGION),$(REGION),us-west-2) \
		$(if $(BUCKET),--bucket=$(BUCKET))

# Example: Configure DigitalOcean provider
.PHONY: config-do
config-do: build
	@echo "Configuring DigitalOcean provider"
	@echo "Usage: make config-do ACCESS_KEY=xxx SECRET_KEY=yyy REGION=nyc3 BUCKET=my-space"
	@if [ -z "$(ACCESS_KEY)" ] || [ -z "$(SECRET_KEY)" ]; then \
		echo "Please provide ACCESS_KEY and SECRET_KEY variables"; \
		exit 1; \
	fi
	$(BINARY_PATH) config set digitalocean \
		--access-key=$(ACCESS_KEY) \
		--secret-key=$(SECRET_KEY) \
		--region=$(if $(REGION),$(REGION),nyc3) \
		$(if $(BUCKET),--bucket=$(BUCKET))

# Show configured providers
.PHONY: config-list
config-list: build
	$(BINARY_PATH) config list

# Development setup
.PHONY: dev-setup
dev-setup:
	go mod download
	go mod tidy
	@echo "Development setup complete"

# Check if the binary works
.PHONY: verify
verify: build
	$(BINARY_PATH) --help
	$(BINARY_PATH) clone --help
	$(BINARY_PATH) config --help
	@echo "All commands verified successfully"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  build-all   - Build for multiple platforms"
	@echo "  deps        - Install dependencies"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  run-help    - Run the application with help"
	@echo "  config-aws  - Configure AWS provider (requires ACCESS_KEY, SECRET_KEY)"
	@echo "  config-do   - Configure DigitalOcean provider (requires ACCESS_KEY, SECRET_KEY)"
	@echo "  config-list - List configured providers"
	@echo "  verify      - Verify the binary works"
	@echo "  dev-setup   - Set up development environment"
	@echo "  help        - Show this help" 