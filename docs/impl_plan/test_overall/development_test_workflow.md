# Development Test Workflow Guide

## Overview

This guide provides developers with clear workflows for using the essential test suite during feature development. The essential tests are designed to provide fast, reliable feedback while maintaining comprehensive validation of critical application paths.

## Test Suite Categories

### 🎯 Essential Tests (Development Focus)

#### Quick Service Validation (2-3 minutes)
```bash
make test-essential-services
```
**When to Use**: After making changes to service layer logic
**Coverage**: 
- Core business logic validation
- Service constructor patterns
- Database connectivity
- Basic service interactions
- Mock configuration validation

**Example Workflow**:
```bash
# 1. Make changes to nudge service
vim internal/nudge/service.go

# 2. Quick validation
make test-essential-services

# 3. Continue development iteration
```

#### End-to-End Flow Testing (3-4 minutes)
```bash
make test-essential-flows
```
**When to Use**: After making changes to API endpoints, webhooks, or event flows
**Coverage**:
- Complete user workflows
- API integration testing
- Event-driven process validation
- Database transaction flows
- External service integrations

**Example Workflow**:
```bash
# 1. Update webhook handler
vim api/handlers/webhook.go

# 2. Test complete flows
make test-essential-flows

# 3. Verify end-to-end functionality
```

#### Comprehensive Validation (5-6 minutes)
```bash
make test-essential-suite
```
**When to Use**: Before committing code, during code review
**Coverage**:
- All essential test categories
- Deterministic execution order
- Full integration validation
- Error handling scenarios
- Performance baseline validation

**Example Workflow**:
```bash
# 1. Complete feature development
git add .

# 2. Pre-commit validation
make test-essential-suite

# 3. Commit if tests pass
git commit -m "feat: implement feature"
```

#### Full Validation (10+ minutes)
```bash
make test-all
```
**When to Use**: Before releases, for complete coverage analysis
**Coverage**:
- Essential tests + all unit tests
- Legacy integration tests
- Performance testing
- Comprehensive coverage metrics

## Development Workflows

### 🔄 Feature Development Cycle

#### 1. Service Layer Changes
```bash
# Quick iteration cycle
while developing:
    make test-essential-services  # 2-3 min feedback
    # Make adjustments based on feedback
    # Continue development

# Final validation
make test-essential-flows         # Ensure integration works
```

#### 2. API/Handler Changes
```bash
# Start with service validation
make test-essential-services      # Ensure services work

# Test API integration
make test-essential-flows         # 3-4 min validation

# Full pre-commit check
make test-essential-suite         # 5-6 min comprehensive
```

#### 3. Bug Fixes
```bash
# For small fixes
make test-essential-services      # Quick validation

# For complex fixes affecting multiple components
make test-essential-suite         # Comprehensive validation
```

#### 4. Refactoring
```bash
# Start with comprehensive baseline
make test-essential-suite         # Ensure everything works

# During refactoring
make test-essential-services      # Quick validation of changes

# Final validation
make test-essential-suite         # Ensure refactoring didn't break anything
```

### 🚀 Pre-Commit Workflow

```bash
# 1. Run essential suite (required)
make test-essential-suite

# 2. Code quality checks
make lint
make fmt

# 3. Build verification
make build

# 4. Commit
git commit -m "feat: your feature description"
```

### 📋 Code Review Workflow

#### For Reviewers
```bash
# 1. Checkout PR branch
git checkout feature-branch

# 2. Run essential tests
make test-essential-suite

# 3. Review test results and code changes
```

#### For Authors
```bash
# Before requesting review
make test-essential-suite         # Ensure tests pass
make lint                         # Code quality
make test-coverage               # Coverage check

# Address review feedback
make test-essential-services      # Quick validation after changes
```

## Troubleshooting Guide

### Common Issues and Solutions

#### 1. Docker Not Running
**Error**: `Cannot connect to the Docker daemon`
**Solution**:
```bash
# Check Docker status
sudo systemctl status docker

# Start Docker if needed
sudo systemctl start docker

# Verify Docker is working
docker run hello-world
```

#### 2. Database Migration Failures
**Error**: `relation 'users' does not exist`
**Status**: ✅ Fixed in integration helpers
**Manual Fix** (if needed):
```bash
# Clean up containers
docker container prune -f

# Restart test
make test-essential-services
```

#### 3. Port Conflicts
**Error**: `bind: address already in use`
**Solution**:
```bash
# Find processes using port
sudo lsof -i :5432

# Kill conflicting process
sudo kill -9 <PID>

# Or use different port in test
export TEST_DB_PORT=5433
```

#### 4. Mock Configuration Issues
**Error**: `Expected subscriber count > 0`
**Status**: ✅ Fixed in chatbot service tests
**Check**: Ensure event bus is in synchronous mode for tests

#### 5. Test Container Cleanup
**Error**: `Container name already in use`
**Solution**:
```bash
# Stop all test containers
docker container stop $(docker container ls -aq --filter name=test)

# Remove test containers
docker container rm $(docker container ls -aq --filter name=test)

# Clean up networks
docker network prune -f
```

### Performance Issues

#### Slow Test Execution
```bash
# Check Docker resource allocation
docker system df

# Clean up unused containers/images
docker system prune -f

# Monitor test execution
time make test-essential-services
```

#### Memory Issues
```bash
# Monitor memory usage during tests
docker stats

# Increase Docker memory allocation if needed
# Docker Desktop -> Settings -> Resources -> Memory
```

### Environment Issues

#### Go Module Issues
```bash
# Clean module cache
go clean -modcache

# Tidy dependencies
go mod tidy

# Download dependencies
go mod download
```

#### Build Issues
```bash
# Clean build artifacts
make clean

# Rebuild from scratch
make build
```

## Test Environment Configuration

### Environment Variables
```bash
# Automatically set by test infrastructure
GO_ENV=test
LOG_LEVEL=debug
DATABASE_HOST=localhost
DATABASE_PORT=5432

# Manual override if needed
export TEST_DB_PORT=5433
export TEST_TIMEOUT=300s
```

### Docker Configuration
```bash
# Check Docker resources
docker system info

# Recommended settings:
# Memory: 4GB+
# CPU: 2+ cores
# Disk: 20GB+ available
```

## Best Practices

### During Development
1. **Use appropriate test level**: Don't run full suite for small changes
2. **Fix tests quickly**: Don't let broken tests accumulate
3. **Monitor execution time**: Report slow tests for optimization
4. **Clean up regularly**: Remove unused containers and images

### Before Committing
1. **Always run essential suite**: `make test-essential-suite`
2. **Check coverage**: Ensure new code is tested
3. **Verify build**: `make build` should succeed
4. **Review test output**: Don't ignore warnings or flaky behavior

### Team Collaboration
1. **Share test failures**: Help teammates with common issues
2. **Update documentation**: Keep troubleshooting guide current
3. **Report infrastructure issues**: File issues for test environment problems
4. **Review test additions**: Ensure new tests follow patterns

## Integration with CI/CD

### GitHub Actions Integration
```yaml
# .github/workflows/test.yml
- name: Run Essential Tests
  run: make test-essential-suite

- name: Upload Coverage
  uses: codecov/codecov-action@v3
  with:
    file: ./coverage.out
```

### Local Pre-Push Hook
```bash
#!/bin/sh
# .git/hooks/pre-push
make test-essential-suite || exit 1
```

## Performance Metrics

### Target Execution Times
| Test Suite | Target Time | Acceptable Range |
|------------|-------------|------------------|
| Essential Services | 2-3 min | 1-4 min |
| Essential Flows | 3-4 min | 2-5 min |
| Essential Suite | 5-6 min | 4-8 min |
| Full Test Suite | 10+ min | 8-15 min |

### Monitoring
```bash
# Track test execution time
time make test-essential-suite

# Monitor resource usage
docker stats --no-stream

# Check coverage trends
make test-coverage
```

---

**Next Review**: September 7, 2025
**Maintainer**: Development Team  
**Last Updated**: August 7, 2025
