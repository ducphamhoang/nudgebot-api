.PHONY: build run test lint docker-build docker-up docker-down clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=main
BINARY_PATH=./cmd/server

# Build the application
build:
    $(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH)

# Run the application
run:
    $(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH) && ./$(BINARY_NAME)

# Run tests
test:
    $(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
    $(GOTEST) -v -cover ./...

# Lint the code
lint:
    golangci-lint run

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