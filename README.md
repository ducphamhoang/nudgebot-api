# 🤖 NudgeBot API

[![CI/CD Pipeline](https://github.com/ducphamhoang/nudgebot-api/workflows/CI/CD%20Pipeline/badge.svg)](https://github.com/ducphamhoang/nudgebot-api/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/ducphamhoang/nudgebot-api)](https://goreportcard.com/report/github.com/ducphamhoang/nudgebot-api)
[![Coverage Status](https://codecov.io/gh/ducphamhoang/nudgebot-api/branch/main/graph/badge.svg)](https://codecov.io/gh/ducphamhoang/nudgebot-api)
[![Go Reference](https://pkg.go.dev/badge/github.com/ducphamhoang/nudgebot-api.svg)](https://pkg.go.dev/github.com/ducphamhoang/nudgebot-api)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

NudgeBot is an intelligent, conversational task assistant built for the Telegram platform that revolutionizes personal productivity through proactive task management, smart scheduling, and persistent accountability features.

## 📖 Usage Guide

### 🔄 Development Workflow

Common tasks during development:

```bash
# Daily development cycle
make dev                    # Start development environment
make test-essential-services # Quick test (35 seconds)
# ... make your changes ...
make test-essential-services # Verify changes
make dev-stop               # Stop when done

# Before committing
make test-all               # Full test suite
make lint                   # Check code quality
make precommit              # All pre-commit checks

# Restart everything (when things get weird)
make dev-stop
make clean
make setup
make dev
```

### 💻 Essential Commands

```bash
# Getting Started
make setup              # Initial setup (run once after cloning)
make dev                # Start development environment
make test-essential-services  # Quick validation (35s)

# Development Cycle
make dev-stop           # Stop development environment
make precommit          # Pre-commit checks (lint + test)
make help               # Show all available commands
```

> 📖 **For a complete list of all available commands**, see the [🛠️ Available Make Commands](#-available-make-commands) section below.

### 🎯 Core Capabilities
- **🔄 Proactive Task Management**: Goes beyond simple reminders with intelligent follow-up nudges
- **🧠 Natural Language Processing**: Add and manage tasks using conversational language via Telegram
- **📅 Smart Scheduling**: Advanced parsing of dates, times, and recurring patterns
- **⚡ Persistent Follow-ups**: Gentle but effective accountability through contextual follow-up messages
- **📊 Progress Tracking**: Monitor task completion rates and productivity insights
- **🔔 Intelligent Notifications**: Context-aware reminders that adapt to your behavior patterns

### 🏗️ Technical Features
- **🚀 High Performance**: Event-driven architecture with concurrent processing
- **🛡️ Production Ready**: Comprehensive error handling, logging, and monitoring
- **🔒 Secure**: OAuth2 authentication, rate limiting, and data encryption
- **📈 Scalable**: Horizontal scaling support with Redis caching
- **🧪 Well Tested**: 95%+ test coverage with unit, integration, and end-to-end tests
- **📦 Cloud Native**: Docker containers, Kubernetes ready, CI/CD pipeline

## 🏛️ Architecture

NudgeBot follows Clean Architecture principles with Domain-Driven Design (DDD) patterns:

```
├── 📁 cmd/                    # Application entrypoints
│   └── 📁 server/            # Main server application
├── 📁 internal/              # Core application logic (private)
│   ├── 📁 chatbot/          # Telegram bot integration & command processing
│   ├── 📁 llm/              # LLM API integration (Gemma, OpenAI)
│   ├── 📁 nudge/            # Core task/nudge domain logic
│   ├── 📁 scheduler/        # Background job scheduling & processing
│   ├── 📁 events/           # Event-driven communication system
│   ├── 📁 config/           # Configuration management
│   ├── 📁 database/         # Database connections & health checks
│   ├── 📁 common/           # Shared utilities and types
│   └── 📁 mocks/           # Generated mocks for testing
├── 📁 pkg/                  # Public shared utilities
│   └── 📁 logger/          # Structured logging with Zap
├── 📁 api/                  # HTTP transport layer
│   ├── 📁 handlers/        # HTTP request handlers
│   ├── 📁 middleware/      # HTTP middleware (logging, auth, CORS)
│   └── 📁 routes/          # Route definitions and setup
├── 📁 configs/             # Configuration files (YAML, ENV)
├── 📁 docs/               # Documentation and specifications
├── 📁 scripts/            # Build and deployment scripts
└── 📁 .github/           # GitHub Actions CI/CD workflows
```

### 🔄 Event-Driven Architecture

NudgeBot uses an event-driven architecture for loose coupling and scalability:

```
📊 Event Flow:
Telegram → Webhook → Command Processor → Event Bus → Domain Services → Database
                                      ↓
                                 Scheduler ← Event Bus ← LLM Service
```

## 🛠️ Tech Stack

### Core Technologies
- **Language**: Go 1.21+ with modern idioms and best practices
- **Web Framework**: Gin with custom middleware and error handling
- **Database**: PostgreSQL 15+ with GORM ORM and migrations
- **Caching**: Redis 7+ for sessions, rate limiting, and performance
- **Message Queue**: Redis Streams for reliable event processing

### External Integrations
- **Telegram Bot API**: Real-time messaging and webhook handling
- **LLM APIs**: Gemma, OpenAI GPT for natural language processing
- **Monitoring**: Prometheus metrics, structured logging with Zap
- **Authentication**: OAuth2, JWT tokens, API key management

### Development & Operations
- **Testing**: Testify, Gomock, Testcontainers for comprehensive testing
- **Containerization**: Docker, Docker Compose, multi-stage builds
- **CI/CD**: GitHub Actions with comprehensive pipeline
- **Configuration**: Viper with environment-specific configs
- **Documentation**: OpenAPI 3.0, automated docs generation

## 🚀 Quick Start

### 📋 Prerequisites

Before you start, ensure you have the following installed:

- **Go 1.21+**: [Download Go](https://golang.org/dl/) - Check with `go version`
- **Docker & Docker Compose**: [Install Docker](https://docs.docker.com/get-docker/) - Check with `docker --version`
- **Make**: Build automation tool
  - **Linux/macOS**: Usually pre-installed
  - **Windows**: Install via [Chocolatey](https://chocolatey.org/): `choco install make`
- **Git**: Version control - Check with `git --version`

### 🚨 Quick Troubleshooting

**Having issues?** Try these quick fixes:

```bash
# If setup fails:
make clean && make setup

# If containers fail to start:
docker-compose down && docker-compose up -d

# If port 8080 is busy:
sudo lsof -i :8080  # Find what's using the port
# Or change port: SERVER_PORT=8081 make dev

# If database connection fails:
make dev-stop && make dev  # Restart everything
```

For detailed troubleshooting, see the [🔍 Troubleshooting section](#-troubleshooting-common-issues) below.

### ⚡ One-Minute Setup

Clone the repository and start the development environment:

```bash
# 1. Clone the repository
git clone https://github.com/ducphamhoang/nudgebot-api.git
cd nudgebot-api

# 2. Run initial setup (installs dependencies, generates mocks, creates config)
make setup

# 3. Start all services (PostgreSQL database + API server)
make dev

# 🎉 Your NudgeBot API is now running!
# 📊 Health check: http://localhost:8080/health
# 📚 API docs: http://localhost:8080/docs
```

That's it! The application is now running with:
- API server on `http://localhost:8080`
- PostgreSQL database on `localhost:5432`
- All dependencies automatically installed and configured

### 🔧 Manual Setup (Step by Step)

If you prefer more control or encounter issues with the quick setup:

#### 1. Clone and Enter Directory
```bash
git clone https://github.com/ducphamhoang/nudgebot-api.git
cd nudgebot-api
```

#### 2. Install Go Dependencies
```bash
# Download and verify Go modules
go mod download
go mod verify

# Generate required mocks for testing
make generate-mocks
```

#### 3. Configure Environment (Optional)
```bash
# The application works with default settings, but you can customize:
cp configs/config.yaml configs/config.local.yaml

# Edit configs/config.local.yaml if needed:
# - Set your Telegram bot token for Telegram integration
# - Set LLM API key for AI features
# - Modify database connection settings
```

#### 4. Start Services
```bash
# Option A: Full development environment (recommended)
make dev

# Option B: Just the database (if you want to run the app manually)
docker-compose up postgres -d

# Option C: Everything manually (without Docker)
# Start your own PostgreSQL on localhost:5432 with:
# - Database: nudgebot
# - User: postgres  
# - Password: postgres
```

#### 5. Run the Application
```bash
# If using make dev, skip this step (already running)

# To run manually:
make run

# Or build first, then run:
make build
./main
```

#### 6. Verify Everything Works
```bash
# Check application health
curl http://localhost:8080/health

# Should return:
# {"status":"ok","timestamp":"2025-08-07T10:00:00Z","services":{"database":"healthy"}}

# Check if database is accessible
curl http://localhost:8080/health/db
```

### 🏠 Local Development Without Docker

If you prefer not to use Docker:

#### 1. Install PostgreSQL Locally
```bash
# Ubuntu/Debian
sudo apt-get install postgresql postgresql-contrib

# macOS (with Homebrew)
brew install postgresql
brew services start postgresql

# Windows (with Chocolatey)
choco install postgresql
```

#### 2. Create Database
```bash
# Connect to PostgreSQL
sudo -u postgres psql

# Create database and user
CREATE DATABASE nudgebot;
CREATE USER postgres WITH PASSWORD 'postgres';
GRANT ALL PRIVILEGES ON DATABASE nudgebot TO postgres;
\q
```

#### 3. Update Configuration
```bash
# Edit configs/config.yaml to match your local setup
# Usually defaults work: localhost:5432, user: postgres, password: postgres
```

#### 4. Run Application
```bash
# Install dependencies and run
make deps
make run
```

### 🔍 Troubleshooting Common Issues

#### "docker: command not found"
```bash
# Install Docker: https://docs.docker.com/get-docker/
# Verify installation:
docker --version
docker-compose --version
```

#### "make: command not found"
```bash
# Linux/Ubuntu
sudo apt-get install build-essential

# macOS
xcode-select --install

# Windows
choco install make
```

#### "Port 8080 already in use"
```bash
# Find what's using the port
sudo lsof -i :8080
# Kill the process or change port in configs/config.yaml

# Or use a different port:
SERVER_PORT=8081 make run
```

#### "Port 5432 already in use" (PostgreSQL conflict)
```bash
# If you have PostgreSQL already running locally:
# Option 1: Stop local PostgreSQL
sudo systemctl stop postgresql

# Option 2: Use different port in docker-compose.yml
# Change "5432:5432" to "5433:5432" and update config.yaml

# Option 3: Use your existing PostgreSQL
# Just run: make run (without make dev)
```

#### "Database connection failed"
```bash
# Check if PostgreSQL is running
docker-compose ps

# Check database logs
docker-compose logs postgres

# Reset database
make docker-down
make docker-up
```

#### Go Module Issues
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download

# Verify checksums
go mod verify
```

### ⚙️ Environment Variables

You can override any configuration using environment variables:

```bash
# Server configuration
export SERVER_PORT=8081
export SERVER_ENVIRONMENT=production

# Database configuration
export DATABASE_HOST=your-db-host
export DATABASE_PASSWORD=your-secure-password

# Telegram bot (optional, for Telegram integration)
export CHATBOT_TOKEN=your-telegram-bot-token

# LLM API (optional, for AI features)
export LLM_API_KEY=your-llm-api-key

# Run with custom environment
make run
```

### 🎯 Next Steps

Once the application is running:

1. **✅ Verify Setup**: 
   ```bash
   # Check if everything is working
   curl http://localhost:8080/health
   # Should return: {"status":"ok",...}
   
   # Check database connection
   make test-essential-services
   # Should show 15/16 tests passing
   ```

2. **📚 Explore the API**: 
   - Health endpoint: `http://localhost:8080/health`
   - Database health: `http://localhost:8080/health/db`
   - API documentation: `http://localhost:8080/docs` (when available)

3. **🧪 Run Tests**: 
   ```bash
   # Quick test to verify everything works
   make test-essential-services  # ~35 seconds
   
   # Comprehensive testing
   make test-all  # ~2-3 minutes
   ```

4. **📱 Optional: Set up Telegram Bot**: 
   - See the "Telegram Bot Setup" section below
   - Add your bot token with: `export CHATBOT_TOKEN=your-token`

5. **🧠 Optional: Configure LLM**: 
   - Add your LLM API key for AI features
   - Export: `export LLM_API_KEY=your-api-key`

6. **📖 Read Documentation**: 
   - Architecture: `docs/architecture.md`
   - MVP Features: `docs/mvp_prd.md`
   - Development Guide: `docs/codebase.md`

### 📱 Optional: Telegram Bot Setup

To use NudgeBot with Telegram:

1. **Create a Telegram Bot**:
   - Message [@BotFather](https://t.me/botfather) on Telegram
   - Send `/newbot` and follow the instructions
   - Save your bot token

2. **Configure the Bot**:
   ```bash
   # Set your bot token
   export CHATBOT_TOKEN=your-bot-token-here
   
   # Restart the application
   make dev-stop
   make dev
   ```

3. **Set Webhook** (for production):
   ```bash
   curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
     -H "Content-Type: application/json" \
     -d '{"url": "https://yourdomain.com/api/v1/telegram/webhook"}'
   ```
   # You should receive a welcome message
   ```

### 📝 API Usage Examples

```bash
# Health check
curl http://localhost:8080/health

# Webhook endpoint (called by Telegram)
curl -X POST http://localhost:8080/api/webhook \
  -H "Content-Type: application/json" \
  -d '{"message": {"text": "/start", "chat": {"id": 123}}}'

# Get API documentation
curl http://localhost:8080/docs
```

## 🧪 Testing

### � Essential Tests for Development

The essential test suite provides fast, reliable feedback for development workflow. These tests are optimized for speed and deterministic execution:

```bash
# Quick service validation (2-3 minutes) - Run after service changes
make test-essential-services

# End-to-end flow testing (3-4 minutes) - Run after API changes  
make test-essential-flows

# Comprehensive validation (5-6 minutes) - Run before commits
make test-essential-suite

# Full validation (10+ minutes) - Run before releases
make test-all
```

**💡 Development Workflow**: Use essential tests for fast iteration during development. They provide comprehensive coverage of critical paths while maintaining quick execution times.

**⚠️ Requirements**: Essential tests require Docker for testcontainers. Ensure Docker is running before executing test commands.

### �🏃 Running Tests

```bash
# Run all tests (unit + integration + essential)
make test-all

# Unit tests only
make test-unit

# Integration tests only
make test-integration

# Generate coverage report
make test-coverage-html

# Watch mode (requires entr)
make test-watch
```

### 🗄️ Database Tests

```bash
# Setup test database
make test-db-setup

# Run integration tests
make test-integration

# Cleanup test database
make test-db-teardown

# Full test cycle
make test-db-setup && make test-integration && make test-db-teardown
```

### 🎯 Test Structure

- **Essential Tests**: Fast integration tests in `/test/essential/` for development workflow
- **Unit Tests**: `*_test.go` files alongside source code for module validation
- **Integration Tests**: `integration_test.go` and `integration_tests_*.go` for comprehensive coverage
- **Mocks**: Generated in `internal/mocks/` using GoMock
- **Test Helpers**: Shared utilities in `/test/essential/helpers/`

### 🔧 Troubleshooting Tests

**Docker Not Running**
```bash
# Start Docker service
sudo systemctl start docker
```

**Port Conflicts**
```bash
# Find and kill conflicting processes
sudo lsof -i :5432
sudo kill -9 <PID>
```

**Container Cleanup**
```bash
# Clean up test containers
docker container prune -f
```

For more troubleshooting guidance, see `docs/impl_plan/test_overall/essential_test_status.md`.

## 🔧 Development

### 📁 Project Structure Deep Dive

```bash
internal/
├── chatbot/          # 🤖 Telegram bot logic
│   ├── service.go           # Main bot service
│   ├── telegram_provider.go # Telegram API client
│   ├── webhook_parser.go    # Webhook message parsing
│   ├── command_processor.go # Command handling logic
│   └── keyboard_builder.go  # Interactive keyboard generation
├── llm/              # 🧠 AI/LLM integration
│   ├── service.go           # LLM service orchestration
│   ├── gemma_provider.go    # Gemma API client
│   └── provider.go          # Provider interface
├── nudge/            # 📋 Core domain logic
│   ├── service.go           # Business logic
│   ├── domain.go            # Domain models
│   ├── repository.go        # Data access interface
│   ├── gorm_repository.go   # GORM implementation
│   └── business_logic.go    # Complex business rules
├── scheduler/        # ⏰ Background processing
│   ├── scheduler.go         # Job scheduling
│   ├── worker.go            # Background workers
│   └── metrics.go           # Performance monitoring
└── events/           # 📡 Event system
    ├── bus.go               # Event bus implementation
    ├── types.go             # Event type definitions
    └── integration.go       # Event flow management
```

### 🔄 Development Workflow

```bash
# 1. Create feature branch
git checkout -b feature/your-feature-name

# 2. Make changes and test
make test-unit
make lint

# 3. Run integration tests
make test-integration

# 4. Pre-commit checks
make precommit

# 5. Build and verify
make build
make docker-build

# 6. Commit and push
git add .
git commit -m "feat: add your feature"
git push origin feature/your-feature-name
```

### 🛠️ Available Make Commands

```bash
# 🏗️ Build & Run
make build              # Build the application
make run                # Build and run locally
make clean              # Clean build artifacts

# 🧪 Testing
make test               # Run unit tests
make test-unit          # Run all unit tests
make test-integration   # Run integration tests
make test-all           # Run all tests (unit + integration)
make test-coverage      # Run tests with coverage
make test-coverage-html # Generate HTML coverage report
make test-watch         # Watch files and re-run tests
make generate-mocks     # Generate test mocks

# 🗄️ Database Testing
make test-db-setup      # Start test database
make test-db-teardown   # Stop test database
make test-db-reset      # Reset test database

# 🔍 Code Quality
make lint               # Run all linters
make lint-modules       # Lint individual modules
make fmt                # Format code
make tidy               # Tidy dependencies
make precommit          # Pre-commit checks

# 🐳 Docker
make docker-build       # Build Docker image
make docker-up          # Start services
make docker-down        # Stop services
make docker-logs        # View service logs
make docker-restart     # Restart services

# 🚀 Development
make dev                # Start development environment
make dev-stop           # Stop development environment
make dev-logs           # View application logs
make dev-rebuild        # Rebuild and restart

# 🔧 CI/CD
make ci                 # Run all CI checks
make check              # Quick development checks
make audit              # Security audit

# 📊 Performance
make bench              # Run benchmarks
make profile-cpu        # Profile CPU usage
make profile-mem        # Profile memory usage

# ❓ Help
make help               # Show all available commands
```

## 🚀 Deployment

### 🐳 Docker Production Build

```bash
# Build production image
docker build -t nudgebot-api:latest .

# Run production container
docker run -d \
  --name nudgebot-api \
  -p 8080:8080 \
  -e ENV=production \
  -e DB_HOST=your-db-host \
  -e TELEGRAM_BOT_TOKEN=your-token \
  nudgebot-api:latest
```

### ☸️ Kubernetes Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nudgebot-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nudgebot-api
  template:
    metadata:
      labels:
        app: nudgebot-api
    spec:
      containers:
      - name: api
        image: nudgebot-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: ENV
          value: "production"
        # Add other environment variables
```

### 🌩️ Cloud Deployment

- **AWS ECS**: Use `deploy/aws/` configurations
- **Google Cloud Run**: Use `deploy/gcp/` configurations  
- **Azure Container Instances**: Use `deploy/azure/` configurations
- **Railway/Heroku**: Use `railway.toml` or `Procfile`

## 📊 Monitoring & Observability

### 📈 Metrics

```bash
# Prometheus metrics endpoint
curl http://localhost:8080/metrics

# Key metrics:
# - http_requests_total
# - http_request_duration_seconds
# - telegram_messages_processed_total
# - scheduler_jobs_executed_total
# - database_connections_active
```

### 📝 Logging

```bash
# Structured JSON logs with Zap
{
  "level": "info",
  "timestamp": "2024-01-01T12:00:00Z",
  "caller": "handlers/webhook.go:45",
  "message": "Processing Telegram webhook",
  "telegram_chat_id": 123456,
  "command": "/start",
  "trace_id": "abc123"
}
```

### 🔍 Health Checks

```bash
# Application health
curl http://localhost:8080/health

# Database health
curl http://localhost:8080/health/db

# Redis health
curl http://localhost:8080/health/redis

# Ready check (Kubernetes)
curl http://localhost:8080/ready
```

## 🤝 Contributing

### 🎯 Contributing Guidelines

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes**
4. **Add tests**: Ensure good test coverage
5. **Run quality checks**: `make precommit`
6. **Commit your changes**: `git commit -m 'feat: add amazing feature'`
7. **Push to the branch**: `git push origin feature/amazing-feature`
8. **Open a Pull Request**

### 📋 Code Standards

- **Go Code Style**: Follow `gofmt` and `golangci-lint` rules
- **Testing**: Maintain 90%+ test coverage
- **Documentation**: Comment public APIs and complex logic
- **Commit Messages**: Use [Conventional Commits](https://conventionalcommits.org/)

### 🔄 Pull Request Process

1. Update documentation if needed
2. Add tests for new features
3. Ensure CI pipeline passes
4. Request review from maintainers
5. Address feedback promptly

## 📚 Documentation

- 📖 **API Docs**: Available at `/docs` endpoint (OpenAPI/Swagger)
- 🏗️ **Architecture**: See `docs/architecture.md`
- 📋 **MVP Requirements**: See `docs/mvp_prd.md`
- 🔧 **Implementation Plans**: See `docs/impl_plan/`
- 📝 **Codebase Guide**: See `docs/codebase.md`

## 🐛 Troubleshooting

### Common Issues

**Database Connection Failed**
```bash
# Check database status
make test-db-setup
docker-compose logs postgres

# Verify connection
psql -h localhost -p 5432 -U postgres -d nudgebot
```

**Telegram Webhook Issues**
```bash
# Check webhook status
curl "https://api.telegram.org/bot<TOKEN>/getWebhookInfo"

# Reset webhook
curl -X POST "https://api.telegram.org/bot<TOKEN>/deleteWebhook"
```

**Build Issues**
```bash
# Clean and rebuild
make clean
make deps
make build
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Telegram Bot API](https://core.telegram.org/bots/api) for messaging platform
- [Gin Framework](https://gin-gonic.com/) for HTTP handling
- [GORM](https://gorm.io/) for database ORM
- [Zap](https://go.uber.org/zap) for structured logging
- [Testify](https://github.com/stretchr/testify) for testing utilities

## 📞 Support

- 📧 **Email**: support@nudgebot.com
- 💬 **Discord**: [Join our community](https://discord.gg/nudgebot)
- 🐛 **Issues**: [GitHub Issues](https://github.com/ducphamhoang/nudgebot-api/issues)
- 📖 **Wiki**: [GitHub Wiki](https://github.com/ducphamhoang/nudgebot-api/wiki)

---

<div align="center">
  <strong>Built with ❤️ by the NudgeBot Team</strong>
  <br>
  <em>Making productivity effortless, one nudge at a time</em>
</div>