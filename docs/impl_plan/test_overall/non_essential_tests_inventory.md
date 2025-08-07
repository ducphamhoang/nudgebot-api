# Non-Essential Tests Inventory

## Overview

This document provides a comprehensive inventory of non-essential tests that exist outside the essential test suite. These tests provide additional coverage but are not critical for the day-to-day development workflow.

## Test Categories

### 1. Unit Tests in Original Locations

#### Service Layer Unit Tests
**Location**: `internal/*/service_test.go`
**Purpose**: Module-specific validation of service logic
**Execution Time**: ~30 seconds each
**When to Run**: Before releases, for comprehensive coverage

- `internal/nudge/service_test.go`
  - Tests nudge service business logic in isolation
  - Validates task creation, retrieval, and updates
  - Mocks external dependencies
  - Covered by essential flows but provides granular validation

- `internal/llm/service_test.go`
  - Tests LLM service parsing logic
  - Validates prompt construction and response parsing
  - Tests error handling for invalid responses
  - Supplementary to essential LLM integration tests

- `internal/chatbot/service_test.go`
  - Tests chatbot command processing
  - Validates message formatting and keyboard generation
  - Tests webhook parsing logic
  - Covered by essential chatbot integration tests

#### API Handler Unit Tests
**Location**: `api/handlers/*_test.go`
**Purpose**: HTTP handler validation and request/response testing
**Execution Time**: ~15 seconds each
**When to Run**: For API contract validation

- `api/handlers/webhook_test.go`
  - Tests webhook endpoint parsing
  - Validates request body handling
  - Tests error responses
  - Covered by essential webhook flow tests

- `api/handlers/health_test.go`
  - Tests health check endpoint
  - Validates database connectivity checks
  - Simple validation covered by essential health tests

#### Infrastructure Unit Tests
**Location**: `pkg/logger/logger_test.go`
**Purpose**: Logging infrastructure validation
**Execution Time**: ~5 seconds
**When to Run**: Rarely needed, covered by integration usage

- Basic logger configuration testing
- Log level filtering validation
- Output format verification

### 2. Legacy Integration Tests

#### Superseded Integration Tests
**Location**: `/integration/`
**Status**: Superseded by essential test suite
**Action**: Can be safely removed after essential tests are stable

- `integration/event_bus_failure_test.go`
  - Tests event bus failure scenarios
  - Redundant with essential event handling tests
  - More complex setup than essential equivalents

#### Duplicate Test Files
**Location**: Various
**Status**: Migrated to essential structure
**Action**: Remove after essential tests are validated

- `integration_test_fixed.go` (marked for deletion)
  - Contains outdated domain types
  - Uses non-existent `llm.TaskSuggestion` type
  - Compilation errors due to outdated interfaces

### 3. Performance and Load Tests

#### Load Testing Infrastructure
**Location**: `test/` (various load test files)
**Purpose**: Performance validation under load
**Execution Time**: 5-10 minutes
**When to Run**: Before major releases, performance testing cycles

- User concurrency testing
- Database performance under load
- API throughput validation
- Memory usage validation

#### Stress Testing
**Purpose**: System behavior under extreme conditions
**Execution Time**: 10-30 minutes
**When to Run**: Monthly, before major releases

- Resource exhaustion scenarios
- Database connection limits
- Event bus throughput limits
- Memory leak detection

### 4. End-to-End Browser Tests

#### User Interface Tests
**Location**: `test/e2e/` (if implemented)
**Purpose**: Full user workflow validation
**Execution Time**: 10-15 minutes
**When to Run**: Before releases, for complete user journey validation

- Telegram bot interaction simulation
- Multi-user scenario testing
- Real external API integration
- Complete user journey validation

## Usage Guidelines

### Development Workflow
```bash
# Primary development testing (fast feedback)
make test-essential-services    # 2-3 minutes
make test-essential-flows       # 3-4 minutes
make test-essential-suite       # 5-6 minutes

# Comprehensive validation (before releases)
make test-all                   # 10+ minutes (includes non-essential)
```

### When to Run Non-Essential Tests

#### During Active Development: ❌ Skip
- Too slow for iterative development
- Essential tests provide sufficient validation
- Can introduce false positives from environment issues

#### Before Feature Branch Merge: ✅ Optional
- Run if changes affect tested modules directly
- Consider for API changes or major refactoring
- Skip for small bug fixes or documentation changes

#### Before Releases: ✅ Required
- Full test suite including non-essential tests
- Performance and load testing
- Complete coverage validation

#### For Coverage Analysis: ✅ Useful
- Identify gaps in essential test coverage
- Validate edge cases not covered by essential tests
- Ensure comprehensive error handling coverage

## Test Maintenance Strategy

### Monthly Review Tasks
1. **Execution Time Analysis**
   - Monitor non-essential test performance
   - Identify slow or flaky tests
   - Consider migration to essential suite if valuable

2. **Coverage Overlap Analysis**
   - Identify redundant coverage between essential and non-essential tests
   - Remove or consolidate duplicate validations
   - Ensure essential tests cover critical paths

3. **Value Assessment**
   - Review non-essential test failure patterns
   - Assess if tests catch real issues
   - Consider archiving tests that don't provide value

### Cleanup Recommendations

#### Immediate Actions (Post Essential Test Stabilization)
- ✅ Remove `integration_test_fixed.go` (compilation errors)
- ✅ Consolidate duplicate webhook tests
- ✅ Archive superseded integration tests

#### Medium Term (1-2 months)
- Review unit test value vs essential test coverage
- Consolidate redundant API handler tests
- Standardize load test infrastructure

#### Long Term (3-6 months)
- Evaluate moving valuable non-essential patterns to essential suite
- Implement performance regression testing
- Create comprehensive e2e testing strategy

## Test Categories by Execution Frequency

### Never Run (Broken/Obsolete)
- `integration_test_fixed.go` - Compilation errors
- Any tests with outdated domain models
- Tests depending on removed interfaces

### Monthly (Performance/Coverage)
- Load testing suite
- Coverage analysis runs
- Performance benchmarking

### Release Cycle (Comprehensive Validation)
- Full unit test suite
- Integration tests in `/integration/`
- End-to-end testing scenarios

### On-Demand (Specific Investigation)
- Targeted unit tests for specific modules
- Performance tests for specific scenarios
- Coverage analysis for specific areas

## Migration Guidance

### Moving Tests from Non-Essential to Essential
When a non-essential test proves valuable for development workflow:

1. **Analyze Test Value**
   - Does it catch real issues during development?
   - Is execution time acceptable (< 30 seconds)?
   - Does it provide unique validation not covered by existing essential tests?

2. **Adaptation Requirements**
   - Use testcontainers for database setup
   - Follow essential test patterns and structure
   - Ensure reliable execution in CI environment

3. **Integration Process**
   - Add to appropriate essential test category
   - Update Makefile targets if needed
   - Remove from non-essential inventory

### Removing Obsolete Tests
Before removing non-essential tests:

1. **Verify Coverage**: Ensure essential tests cover the same scenarios
2. **Document Decision**: Record why test was removed
3. **Archive**: Keep test code in documentation for future reference
4. **Update Inventory**: Remove from this document

---

**Last Updated**: August 7, 2025
**Next Review**: September 7, 2025
**Maintainer**: Development Team
