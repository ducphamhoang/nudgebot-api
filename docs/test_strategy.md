# NudgeBot MVP Test Strategy

This document outlines the comprehensive testing approach for the NudgeBot MVP, ensuring full coverage of PRD requirements across critical system areas.

## 1. Testing Philosophy

### Three-Tier Testing Approach: Unit → Integration → End-to-End

**Unit Tests**: Focus on isolated service logic, business rules, error handling, and validation. Each service is tested in isolation with mocked dependencies to ensure fast, reliable execution.

**Integration Tests**: Validate cross-service flows, database operations, and HTTP endpoints. Use real services where practical (PostgreSQL via testcontainers, Event Bus) while stubbing external dependencies (LLM, Telegram APIs).

**End-to-End Tests**: Complete user journeys from webhook receipt to database persistence, validating the entire system works cohesively.

### MVP Requirements and PRD Traceability

All tests are mapped to specific MVP user stories to ensure comprehensive coverage. The test coverage matrix (`docs/test_coverage_matrix.md`) provides traceability between each PRD requirement and corresponding test scenarios.

### Balance Between Real Services and Mocks/Stubs

- **Real Services**: PostgreSQL (testcontainers), Event Bus, HTTP handlers for authentic behavior validation
- **Stub Services**: LLM provider, Telegram provider to avoid external dependencies and ensure test reliability
- **Mock Services**: Repositories, event buses for unit tests requiring precise control over behavior

## 2. Test Categories and Scope

### Unit Tests (`./unit/`)
- **Service Logic**: Business rule validation, data transformation, error handling
- **Error Scenarios**: LLM provider failures, network timeouts, malformed data
- **Validation**: Input validation, domain rule enforcement, edge case handling
- **Performance**: Critical path timing, resource usage patterns

### Integration Tests (`./integration/`)
- **Cross-Service Flows**: Message→Task→Reminder complete cycles
- **Database Operations**: Migration reliability, transaction integrity, concurrent access
- **HTTP Endpoints**: Webhook processing, command handling, callback query processing
- **Event Bus**: Publish/subscribe reliability, failure recovery, ordering guarantees

### End-to-End Tests
- **User Journeys**: Complete task lifecycle from creation to completion
- **Error Recovery**: System behavior during partial failures
- **Performance**: Response times, throughput under load

## 3. Testing Infrastructure

### Real Service Integration
- **PostgreSQL**: Via testcontainers for authentic database behavior
- **Event Bus**: Real implementation for cross-service communication testing
- **HTTP Handlers**: Gin framework with actual routing and middleware

### Stub Service Implementation
- **LLM Provider**: Controlled responses for task parsing scenarios
- **Telegram Provider**: Simulated webhook payloads and message sending
- **External APIs**: Predictable behavior without network dependencies

### Mock Service Usage
- **Repositories**: Precise control over data layer behavior in unit tests
- **Event Buses**: Controlled event flow for isolated service testing
- **Clock Abstraction**: Deterministic timing for scheduler and reminder tests

### Test Utilities
- **Clock Abstraction** (`internal/common/clock.go`): Enable fast, deterministic timing tests
- **Webhook Generators**: Create authentic Telegram payload structures
- **Assertion Helpers**: Domain-specific validation functions
- **Database Seeders**: Consistent test data setup

## 4. Critical Flow Coverage

### High Priority (MVP Critical)
1. **Scheduler/Reminder/Nudge Flow**: Complete timing cycle from task creation to reminder delivery
2. **Command Processing HTTP Integration**: Webhook→Command→Response flow validation
3. **Database Migration Failures**: Recovery scenarios and data integrity

### Medium Priority (Production Ready)
4. **LLM Provider Error Scenarios**: Network failures, malformed responses, validation errors
5. **Callback Query Flow**: Inline button interactions and state management
6. **Event Bus Failure Handling**: Publish failures, subscriber errors, recovery mechanisms

### Low Priority (Performance & Edge Cases)
7. **Scheduler Timing Logic**: Precise timing calculations and edge cases
8. **Concurrency & Race Conditions**: Multi-user access patterns

### Coverage Mapping
Each test directly maps to MVP user stories:
- US-01 through US-10: Covered by integration and unit test suites
- Error scenarios: Comprehensive failure mode coverage
- Performance requirements: Load and timing validation

## 5. Test Execution Strategy

### Build Tag Usage
```go
//go:build integration
```
Integration tests use build tags to separate from unit tests, enabling selective execution based on environment capabilities.

### Make Targets
- `make test-unit`: Fast unit tests without external dependencies
- `make test-integration`: Integration tests with testcontainers and real services
- `make test-all`: Complete test suite execution

### Execution Patterns
```bash
# Unit tests (fast, CI-friendly)
go test ./unit/ ./internal/...

# Integration tests (requires Docker)
go test -tags=integration ./integration/

# Specific test categories
go test -run TestLLM ./unit/
go test -tags=integration -run TestScheduler ./integration/
```

### CI/CD Integration
- **Parallel Execution**: Unit and integration tests run in separate CI stages
- **Environment Requirements**: Integration tests require Docker and sufficient resources
- **Coverage Reporting**: Combined coverage from all test tiers

## 6. Error Scenario Testing

### Database Failures and Migration Issues
- **Migration Failures**: Network interruptions, permission errors, schema conflicts
- **Transaction Rollbacks**: Partial failure recovery, data consistency validation
- **Connection Failures**: Pool exhaustion, network timeouts, connection recovery

### External Service Timeouts and Errors
- **LLM Provider**: API timeouts, rate limits, malformed responses, network failures
- **HTTP Errors**: Status code handling, retry mechanisms, circuit breaker patterns
- **Data Validation**: Malformed inputs, missing fields, type mismatches

### Event Bus Failures and Recovery
- **Publish Failures**: Bus unavailability, subscriber errors, message delivery guarantees
- **Ordering Issues**: Concurrent event processing, race condition handling
- **Recovery Patterns**: Service restart scenarios, event replay mechanisms

### Concurrent Access and Race Conditions
- **Multi-User Scenarios**: Simultaneous task operations, data consistency
- **Resource Contention**: Database locks, event bus throughput, memory usage
- **State Management**: Concurrent state changes, atomic operations

## 7. Maintenance and Evolution

### Adding New Tests
1. **Map to Requirements**: Ensure new features have corresponding test coverage
2. **Follow Patterns**: Use existing test structure and utilities
3. **Update Matrix**: Maintain traceability in test coverage matrix
4. **Performance Impact**: Monitor test execution time and CI duration

### Test Reliability Patterns
- **Deterministic Timing**: Use clock abstraction for time-dependent tests
- **Isolated State**: Each test manages its own data and cleanup
- **Stable Dependencies**: Pin testcontainer versions and external service mocks
- **Retry Mechanisms**: Handle flaky infrastructure with appropriate retries

### Performance and Execution Time
- **Parallel Execution**: Independent tests can run concurrently
- **Resource Management**: Proper cleanup prevents resource leaks
- **Test Grouping**: Logical grouping reduces setup/teardown overhead
- **Coverage Optimization**: Focus on high-value test scenarios

### PRD Requirement Traceability
The test coverage matrix ensures every MVP requirement has corresponding test coverage:
- **Direct Mapping**: Each user story maps to specific test scenarios
- **Gap Identification**: Missing coverage highlighted and prioritized
- **Regression Prevention**: Changes to requirements trigger test updates
- **Quality Gates**: Coverage thresholds enforce comprehensive testing

## Test Coverage Targets

- **Unit Tests**: >90% coverage for service logic and business rules
- **Integration Tests**: 100% coverage for critical user journeys
- **Error Scenarios**: Comprehensive coverage for all identified failure modes
- **Performance**: Response time and throughput validation for MVP requirements

This strategy ensures the NudgeBot MVP meets all functional requirements while maintaining high reliability and performance standards through comprehensive test coverage.
