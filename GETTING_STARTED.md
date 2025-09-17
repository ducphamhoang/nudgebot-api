# 🚀 Getting Started with NudgeBot API

## Quick Start (3 Steps)

```bash
# 1. Clone repository
git clone https://github.com/ducphamhoang/nudgebot-api.git
cd nudgebot-api

# 2. Setup dependencies 
make setup

# 3. Start development environment
make dev
```

**✅ Done!** Your API is running at `http://localhost:8080`

## Verify It Works

```bash
# Health check
curl http://localhost:8080/health

# Run tests
make test-essential-services
```

## Common Tasks

```bash
# View logs
make dev-logs

# Stop everything
make dev-stop

# Restart everything
make dev-stop && make dev

# Get help
make help
```

## Troubleshooting

**Port 8080 busy?**
```bash
SERVER_PORT=8081 make dev
```

**Docker issues?**
```bash
docker-compose down
make dev
```

**Need fresh start?**
```bash
make dev-stop
make clean
make setup
make dev
```

For detailed documentation, see [README.md](README.md).

## Deploying to Render

If you plan to deploy to Render using the project `Dockerfile`, ensure the builder image Go version matches the project's `go.mod` toolchain.

- Required Go version: `go 1.23.0` with toolchain `go1.24.5` (we use `golang:1.24.5-alpine` in the `Dockerfile`).
- Recommended build approach: Deploy the repository to Render as a Docker web service so Render builds the image using the included `Dockerfile`.

If you prefer Render's native Go build instead of Docker, set the Go version to `1.24.5` in Render's environment and use the following build command:

```bash
go mod download && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go
```

Start command (if running outside Docker):

```bash
./main
```

Do NOT store secrets in the repo. Add `DATABASE_*`, `CHATBOT_TOKEN`, `LLM_API_KEY`, etc. as environment variables in Render's dashboard.
