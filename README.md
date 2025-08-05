# 🤖 NudgeBot API

[![CI/CD Pipeline](https://github.com/yourusername/nudgebot-api/workflows/CI/CD%20Pipeline/badge.svg)](https://github.com/yourusername/nudgebot-api/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/nudgebot-api)](https://goreportcard.com/report/github.com/yourusername/nudgebot-api)
[![Coverage Status](https://codecov.io/gh/yourusername/nudgebot-api/branch/main/graph/badge.svg)](https://codecov.io/gh/yourusername/nudgebot-api)
[![Go Reference](https://pkg.go.dev/badge/github.com/yourusername/nudgebot-api.svg)](https://pkg.go.dev/github.com/yourusername/nudgebot-api)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

NudgeBot is an intelligent, conversational task assistant built for the Telegram platform that revolutionizes personal productivity through proactive task management, smart scheduling, and persistent accountability features.

## ✨ Features

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

- **Go 1.21+**: [Download Go](https://golang.org/dl/)
- **Docker & Docker Compose**: [Install Docker](https://docs.docker.com/get-docker/)
- **Make**: Build tool for running commands
- **Git**: Version control

### ⚡ One-Minute Setup

```bash
# Clone and start development environment
git clone https://github.com/yourusername/nudgebot-api.git
cd nudgebot-api
make dev

# 🎉 Your NudgeBot API is now running!
# 📊 Health check: http://localhost:8080/health
# 📚 API docs: http://localhost:8080/docs
```

### 🔧 Manual Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/nudgebot-api.git
   cd nudgebot-api
   ```

2. **📝 Configure environment**
   ```bash
   # Copy example environment file
   cp configs/config.example.yaml configs/config.yaml
   
   # Edit configuration with your settings
   # Set Telegram bot token, database credentials, etc.
   ```

3. **🐳 Start services with Docker**
   ```bash
   # Start all services (PostgreSQL, Redis, API)
   make dev
   
   # Or manually with Docker Compose
   docker-compose up -d
   ```

4. **✅ Verify setup**
   ```bash
   # Check health endpoint
   curl http://localhost:8080/health
   
   # Check logs
   make dev-logs
   ```

### 🏠 Local Development (Without Docker)

1. **📦 Install dependencies**
   ```bash
   # Download Go modules
   make deps
   
   # Generate mocks for testing
   make generate-mocks
   ```

2. **🗄️ Setup PostgreSQL**
   ```bash
   # Using Docker for database only
   docker run --name nudgebot-postgres \
     -e POSTGRES_DB=nudgebot \
     -e POSTGRES_USER=postgres \
     -e POSTGRES_PASSWORD=postgres \
     -p 5432:5432 -d postgres:15-alpine
   ```

3. **🚀 Run the application**
   ```bash
   # Build and run
   make build && ./main
   
   # Or directly
   make run
   ```

## 📖 Usage Guide

### 🤖 Telegram Bot Setup

1. **Create Telegram Bot**
   ```bash
   # Message @BotFather on Telegram
   /newbot
   # Follow prompts to get your bot token
   ```

2. **Configure webhook**
   ```bash
   # Set webhook URL (replace with your domain)
   curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
     -H "Content-Type: application/json" \
     -d '{"url": "https://yourdomain.com/api/webhook"}'
   ```

3. **Test the bot**
   ```bash
   # Send "/start" to your bot on Telegram
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

### 🏃 Running Tests

```bash
# Run all tests
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

- **Unit Tests**: `*_test.go` files alongside source code
- **Integration Tests**: `integration_test.go` and `integration_tests_*.go`
- **Mocks**: Generated in `internal/mocks/` using GoMock
- **Test Helpers**: `integration_test_helpers.go` for common utilities

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
- 🐛 **Issues**: [GitHub Issues](https://github.com/yourusername/nudgebot-api/issues)
- 📖 **Wiki**: [GitHub Wiki](https://github.com/yourusername/nudgebot-api/wiki)

---

<div align="center">
  <strong>Built with ❤️ by the NudgeBot Team</strong>
  <br>
  <em>Making productivity effortless, one nudge at a time</em>
</div>

# Testing
make test         # Run tests
make test-coverage # Run tests with coverage

# Code Quality
make lint         # Run linter
make fmt          # Format code

# Docker
make docker-up    # Start Docker services
make docker-down  # Stop Docker services
make docker-logs  # View logs

# Utilities
make clean        # Clean build artifacts
make tidy         # Tidy dependencies
```

## Configuration

The application uses a hierarchical configuration system:

1. **Default values** (defined in code)
2. **Configuration file** (`configs/config.yaml`)
3. **Environment variables** (override config file)

### Environment Variables

All configuration can be overridden using environment variables with the format `SECTION_KEY`:

```bash
SERVER_PORT=8080
DATABASE_HOST=localhost
DATABASE_PASSWORD=secretpassword
```

## API Endpoints

### Health Check
- **GET** `/health` - Application health status
- **GET** `/api/v1/health` - API health status

## Development Guidelines

This project follows the coding standards outlined in `docs/rules.md`:

- **Clean Architecture** with clear separation of concerns
- **Interface-driven development** with dependency injection
- **Comprehensive error handling** with context propagation
- **Structured logging** with request correlation
- **Test-driven development** with high coverage
- **Security best practices** with input validation

## Contributing

1. Follow the coding standards in `docs/rules.md`
2. Write tests for new functionality
3. Ensure all tests pass: `make test`
4. Run linting: `make lint`
5. Format code: `make fmt`

## License

[License information to be added]