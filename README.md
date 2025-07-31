# NudgeBot API

NudgeBot is a conversational task assistant built for the Telegram platform that helps users manage their tasks with proactive reminders and follow-ups.

## Features

- **Proactive Task Management**: Not just reminders, but follow-up nudges to ensure task completion
- **Natural Language Processing**: Add tasks using natural language via Telegram
- **Smart Scheduling**: Intelligent parsing of dates and times
- **Persistent Follow-ups**: Gentle accountability through follow-up messages

## Architecture

This project follows Clean Architecture principles with the following structure:

```
├── cmd/                    # Application entrypoints
├── internal/              # Core application logic
│   ├── config/           # Configuration management
│   └── database/         # Database connections
├── pkg/                  # Shared utilities
│   └── logger/          # Structured logging
├── api/                  # HTTP transport layer
│   ├── handlers/        # Request handlers
│   ├── middleware/      # HTTP middleware
│   └── routes/          # Route definitions
├── configs/             # Configuration files
└── test/               # Test utilities and mocks
```

## Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: PostgreSQL with GORM
- **Logging**: Zap (structured logging)
- **Configuration**: Viper
- **Containerization**: Docker & Docker Compose

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- PostgreSQL (if running locally)

### Development Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd nudgebot-api
   ```

2. **Copy environment file**
   ```bash
   cp .env.example .env
   ```

3. **Start with Docker Compose**
   ```bash
   make dev
   # or
   docker-compose up -d
   ```

4. **Verify the setup**
   ```bash
   curl http://localhost:8080/health
   ```

### Local Development

1. **Install dependencies**
   ```bash
   go mod tidy
   ```

2. **Start PostgreSQL** (if not using Docker)
   ```bash
   # Using Docker for just the database
   docker run --name postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15-alpine
   ```

3. **Run the application**
   ```bash
   make run
   ```

## Available Commands

```bash
# Development
make dev          # Start development environment
make dev-stop     # Stop development environment
make run          # Run application locally

# Building
make build        # Build the application
make docker-build # Build Docker image

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