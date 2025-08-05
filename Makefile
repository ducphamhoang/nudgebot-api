.PHONY: build run test lint docker-build docker-up docker-down clean generate-mocks test-unit test-integration lint-modules test-coverage test-coverage-html test-all test-db-setup test-db-teardown precommit test-watch help

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

# Test parameters
COVERAGE_OUT=coverage.out
COVERAGE_HTML=coverage.html
TEST_TIMEOUT=300s
INTEGRATION_TAG=integration

# Default target
all: help

# ==============================================================================
# Build and Run Targets
# ==============================================================================

# Build the application
build:
	@echo "🔨 Building application..."
	$(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH)

# Run the application
run: build
	@echo "🚀 Running application..."
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_OUT)
	rm -f $(COVERAGE_HTML)
	rm -f logger.test
	@echo "✅ Clean completed"

# ==============================================================================
# Testing Infrastructure
# ==============================================================================

# Generate all mocks
generate-mocks:
	@echo "🔄 Generating mocks..."
	$(GOGENERATE) ./internal/mocks/...
	@echo "✅ Mocks generated"

# Run all unit tests
test-unit: generate-mocks
	@echo "🧪 Running unit tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/...
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./api/...
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./pkg/...
	@echo "✅ Unit tests completed"

# Run integration tests with build tag (requires Docker for testcontainers)
test-integration: test-db-setup
	@echo "🔧 Running integration tests..."
	@echo "ℹ️  Note: Integration tests require Docker to be running for testcontainers"
	CGO_ENABLED=1 $(GOTEST) -v -timeout=5m -tags=$(INTEGRATION_TAG) ./integration/...
	@echo "✅ Integration tests completed"

# Run all tests (unit + integration)
test-all: test-unit test-integration
	@echo "✅ All tests completed"

# Run tests with coverage
test-coverage: generate-mocks
	@echo "📊 Running tests with coverage..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -coverprofile=$(COVERAGE_OUT) ./...
	$(GOCMD) tool cover -func=$(COVERAGE_OUT)
	@echo "✅ Coverage analysis completed"

# Generate HTML coverage report
test-coverage-html: test-coverage
	@echo "📈 Generating HTML coverage report..."
	$(GOCMD) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "✅ HTML coverage report generated: $(COVERAGE_HTML)"

# Watch tests (requires entr: brew install entr or apt-get install entr)
test-watch:
	@echo "👀 Watching for changes and running tests..."
	find . -name "*.go" | grep -v vendor | entr -c make test-unit

# Pre-commit hook: format, lint, and test
precommit: fmt lint test-unit
	@echo "✅ Pre-commit checks completed"

# Run basic tests (alias for test-unit)
test: test-unit

# ==============================================================================
# Database Testing
# ==============================================================================

# Setup test database using Docker Compose
test-db-setup:
	@echo "🗄️  Setting up test database..."
	docker-compose -f docker-compose.test.yml up -d postgres-test
	@echo "⏳ Waiting for database to be ready..."
	@for i in $$(seq 1 30); do \
		if docker-compose -f docker-compose.test.yml exec -T postgres-test pg_isready -U test_user -d test_nudgebot 2>/dev/null; then \
			echo "✅ Test database ready"; \
			exit 0; \
		fi; \
		echo "Waiting... ($$i/30)"; \
		sleep 1; \
	done; \
	echo "❌ Database failed to start within 30 seconds"; \
	exit 1

# Teardown test database
test-db-teardown:
	@echo "🗑️  Tearing down test database..."
	docker-compose -f docker-compose.test.yml down
	@echo "✅ Test database stopped"

# Reset test database
test-db-reset: test-db-teardown test-db-setup

# ==============================================================================
# Code Quality and Linting
# ==============================================================================

# Lint the code
lint: lint-modules
	@echo "🔍 Running comprehensive linting..."
	golangci-lint run --timeout 5m
	@echo "✅ Linting completed"

# Lint specific modules
lint-modules:
	@echo "🔍 Linting individual modules..."
	golangci-lint run ./internal/events/...
	golangci-lint run ./internal/chatbot/...
	golangci-lint run ./internal/llm/...
	golangci-lint run ./internal/nudge/...
	golangci-lint run ./internal/common/...
	golangci-lint run ./internal/config/...
	golangci-lint run ./internal/database/...
	golangci-lint run ./internal/scheduler/...
	golangci-lint run ./api/...
	golangci-lint run ./pkg/...
	@echo "✅ Module linting completed"

# Format code
fmt:
	@echo "🎨 Formatting code..."
	$(GOCMD) fmt ./...
	@echo "✅ Code formatted"

# Tidy dependencies
tidy:
	@echo "📦 Tidying dependencies..."
	$(GOMOD) tidy
	@echo "✅ Dependencies tidied"

# Verify dependencies
verify:
	@echo "🔒 Verifying dependencies..."
	$(GOMOD) verify
	@echo "✅ Dependencies verified"

# Download dependencies
deps:
	@echo "⬇️  Downloading dependencies..."
	$(GOMOD) download
	@echo "✅ Dependencies downloaded"

# ==============================================================================
# Docker Operations
# ==============================================================================

# Build Docker image
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t nudgebot-api .
	@echo "✅ Docker image built"

# Start development services
docker-up:
	@echo "🐳 Starting development services..."
	docker-compose up -d
	@echo "✅ Services started"

# Stop services
docker-down:
	@echo "🛑 Stopping services..."
	docker-compose down
	@echo "✅ Services stopped"

# View logs
docker-logs:
	docker-compose logs -f

# Restart services
docker-restart: docker-down docker-up

# ==============================================================================
# Database Operations
# ==============================================================================

# Run database migrations (placeholder)
db-migrate-up:
	@echo "⬆️  Running database migrations..."
	@echo "⚠️  Database migrations not implemented yet"

# Rollback database migrations (placeholder)
db-migrate-down:
	@echo "⬇️  Rolling back database migrations..."
	@echo "⚠️  Database migrations not implemented yet"

# Create new migration (placeholder)
db-migration-create:
	@echo "📝 Creating new migration..."
	@echo "⚠️  Migration creation not implemented yet"

# ==============================================================================
# Development Helpers
# ==============================================================================

# Start development environment
dev: docker-up
	@echo "🚀 Development environment ready"
	@echo "📊 Health check: http://localhost:8080/health"

# Stop development environment
dev-stop: docker-down
	@echo "🛑 Development environment stopped"

# Show application logs
dev-logs:
	docker-compose logs -f app

# Rebuild and restart development environment
dev-rebuild: docker-down docker-build docker-up

# ==============================================================================
# CI/CD and Quality Checks
# ==============================================================================

# Run all CI checks (used by GitHub Actions)
ci: deps generate-mocks fmt lint test-all
	@echo "✅ All CI checks passed"

# Quick development check
check: fmt lint test-unit
	@echo "✅ Quick checks completed"

# Security audit
audit:
	@echo "🔐 Running security audit..."
	$(GOCMD) list -json -m all | nancy sleuth
	@echo "✅ Security audit completed"

# ==============================================================================
# Benchmarking and Performance
# ==============================================================================

# Run benchmarks
bench:
	@echo "🏃 Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...
	@echo "✅ Benchmarks completed"

# Profile CPU usage
profile-cpu:
	@echo "🔬 Profiling CPU usage..."
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./...
	$(GOCMD) tool pprof cpu.prof

# Profile memory usage
profile-mem:
	@echo "🧠 Profiling memory usage..."
	$(GOTEST) -memprofile=mem.prof -bench=. ./...
	$(GOCMD) tool pprof mem.prof

# ==============================================================================
# Help and Documentation
# ==============================================================================

# Show help
help:
	@echo "🤖 NudgeBot API - Available Make Targets"
	@echo ""
	@echo "📋 Build and Run:"
	@echo "  build              Build the application"
	@echo "  run                Build and run the application"
	@echo "  clean              Clean build artifacts"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  test               Run unit tests (alias for test-unit)"
	@echo "  test-unit          Run all unit tests"
	@echo "  test-integration   Run integration tests"
	@echo "  test-all           Run all tests (unit + integration)"
	@echo "  test-coverage      Run tests with coverage"
	@echo "  test-coverage-html Generate HTML coverage report"
	@echo "  test-watch         Watch files and run tests on change"
	@echo "  generate-mocks     Generate mock implementations"
	@echo ""
	@echo "🗄️  Database Testing:"
	@echo "  test-db-setup      Start test database"
	@echo "  test-db-teardown   Stop test database"
	@echo "  test-db-reset      Reset test database"
	@echo ""
	@echo "🔍 Code Quality:"
	@echo "  lint               Run all linters"
	@echo "  lint-modules       Lint individual modules"
	@echo "  fmt                Format code"
	@echo "  tidy               Tidy dependencies"
	@echo "  verify             Verify dependencies"
	@echo "  precommit          Run pre-commit checks"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  docker-build       Build Docker image"
	@echo "  docker-up          Start services"
	@echo "  docker-down        Stop services"
	@echo "  docker-logs        View service logs"
	@echo "  docker-restart     Restart services"
	@echo ""
	@echo "🚀 Development:"
	@echo "  dev                Start development environment"
	@echo "  dev-stop           Stop development environment"
	@echo "  dev-logs           View application logs"
	@echo "  dev-rebuild        Rebuild and restart environment"
	@echo ""
	@echo "🔧 CI/CD:"
	@echo "  ci                 Run all CI checks"
	@echo "  check              Quick development checks"
	@echo "  audit              Security audit"
	@echo ""
	@echo "📊 Performance:"
	@echo "  bench              Run benchmarks"
	@echo "  profile-cpu        Profile CPU usage"
	@echo "  profile-mem        Profile memory usage"
	@echo ""
	@echo "❓ Help:"
	@echo "  help               Show this help message"