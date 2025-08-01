.PHONY: build run test lint docker-build docker-up docker-down clean generate-mocks test-unit test-integration lint-modules

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOGENERATE=$(GOCMD) generate
BINARY_NAME=main
BINARY_PATH=./cmd/server

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH)

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH) && ./$(BINARY_NAME)

# Run tests
test: generate-mocks
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage: generate-mocks
	$(GOTEST) -v -cover ./...

# Generate mocks for testing
generate-mocks:
	$(GOGENERATE) ./internal/mocks/...

# Run unit tests for individual modules
test-unit: generate-mocks
	$(GOTEST) -v ./internal/events/...
	$(GOTEST) -v ./internal/chatbot/...
	$(GOTEST) -v ./internal/llm/...
	$(GOTEST) -v ./internal/nudge/...
	$(GOTEST) -v ./internal/common/...

# Run integration tests (placeholder)
test-integration:
	@echo "Integration tests not implemented yet"

# Lint the code
lint: lint-modules
	golangci-lint run

# Lint specific modules
lint-modules:
	golangci-lint run ./internal/events/...
	golangci-lint run ./internal/chatbot/...
	golangci-lint run ./internal/llm/...
	golangci-lint run ./internal/nudge/...
	golangci-lint run ./internal/common/...

# Format code
fmt:
	$(GOCMD) fmt ./...

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Build Docker image
docker-build:
	docker build -t nudgebot-api .

# Start services with Docker Compose
docker-up:
	docker-compose up -d

# Stop services
docker-down:
	docker-compose down

# View logs
docker-logs:
	docker-compose logs -f

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Database operations
db-migrate-up:
	@echo "Database migrations not implemented yet"

db-migrate-down:
	@echo "Database migrations not implemented yet"

# Development helpers
dev: docker-up
	@echo "Development environment started"

dev-stop: docker-down
	@echo "Development environment stopped"