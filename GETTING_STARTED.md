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
