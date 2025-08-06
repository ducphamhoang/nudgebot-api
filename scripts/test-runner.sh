#!/bin/bash

# test-runner.sh - Comprehensive test execution and validation script
# This script automates the entire test execution and validation process

set -euo pipefail

# Color definitions for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
LOG_FILE="test_execution_results.log"
TEST_STATUS_FILE="docs/test_status.md"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR" && pwd)"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}" | tee -a "$LOG_FILE"
}

# Update test status documentation
update_test_status() {
    local section="$1"
    local content="$2"
    
    # This would update the test_status.md file with results
    # Implementation would parse and update the markdown file
    log_info "Updating test status: $section"
}

# Validate environment
validate_environment() {
    log_section "Environment Validation"
    
    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker daemon is not running"
        return 1
    fi
    log_success "Docker daemon is running"
    
    # Check Go version
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed"
        return 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}')
    log_success "Go version: $GO_VERSION"
    
    # Check CGO_ENABLED
    if [ "${CGO_ENABLED:-1}" != "1" ]; then
        log_warning "CGO_ENABLED is not set to 1, setting it now"
        export CGO_ENABLED=1
    fi
    log_success "CGO_ENABLED=${CGO_ENABLED:-1}"
    
    # Check testcontainers support (basic check)
    if ! go list -m github.com/testcontainers/testcontainers-go >/dev/null 2>&1; then
        log_warning "testcontainers-go module not found in dependencies"
    else
        log_success "testcontainers-go is available"
    fi
    
    return 0
}

# Verify dependencies
verify_dependencies() {
    log_section "Dependency Verification"
    
    log_info "Running go mod tidy..."
    if go mod tidy 2>&1 | tee -a "$LOG_FILE"; then
        log_success "go mod tidy completed successfully"
    else
        log_error "go mod tidy failed"
        return 1
    fi
    
    log_info "Running go mod verify..."
    if go mod verify 2>&1 | tee -a "$LOG_FILE"; then
        log_success "go mod verify completed successfully"
    else
        log_error "go mod verify failed"
        return 1
    fi
    
    return 0
}

# Generate mocks
generate_mocks() {
    log_section "Mock Generation"
    
    log_info "Executing make generate-mocks..."
    if make generate-mocks 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Mock generation completed successfully"
        
        # Verify expected mock files exist
        local expected_mocks=(
            "internal/mocks/event_bus_mock.go"
            "internal/mocks/chatbot_service_mock.go"
            "internal/mocks/telegram_provider_mock.go"
            "internal/mocks/llm_service_mock.go"
            "internal/mocks/llm_provider_mock.go"
            "internal/mocks/nudge_service_mock.go"
            "internal/mocks/nudge_repository_mock.go"
            "internal/mocks/scheduler_mock.go"
        )
        
        local missing_mocks=()
        for mock in "${expected_mocks[@]}"; do
            if [ ! -f "$mock" ]; then
                missing_mocks+=("$mock")
            fi
        done
        
        if [ ${#missing_mocks[@]} -eq 0 ]; then
            log_success "All expected mock files generated successfully"
        else
            log_warning "Missing mock files: ${missing_mocks[*]}"
        fi
    else
        log_error "Mock generation failed"
        return 1
    fi
    
    return 0
}

# Compilation check
compilation_check() {
    log_section "Compilation Check"
    
    log_info "Running go vet..."
    if go vet ./... 2>&1 | tee -a "$LOG_FILE"; then
        log_success "go vet passed"
    else
        log_error "go vet found issues"
        return 1
    fi
    
    log_info "Testing compilation with go test -c..."
    if go test -c ./... >/dev/null 2>&1; then
        log_success "All packages compile successfully"
        # Clean up compiled test binaries
        find . -name "*.test" -delete 2>/dev/null || true
    else
        log_error "Compilation errors found"
        go test -c ./... 2>&1 | tee -a "$LOG_FILE"
        return 1
    fi
    
    return 0
}

# Run unit tests
run_unit_tests() {
    log_section "Unit Test Execution"
    
    log_info "Executing make test-unit..."
    local unit_test_output
    local unit_test_exit_code=0
    
    unit_test_output=$(make test-unit 2>&1) || unit_test_exit_code=$?
    echo "$unit_test_output" | tee -a "$LOG_FILE"
    
    if [ $unit_test_exit_code -eq 0 ]; then
        log_success "Unit tests passed"
        
        # Parse test results
        local total_tests=$(echo "$unit_test_output" | grep -o "RUN.*Test" | wc -l || echo "0")
        local passed_tests=$(echo "$unit_test_output" | grep -o "PASS.*Test" | wc -l || echo "0")
        
        log_info "Unit test summary: $passed_tests/$total_tests tests passed"
    else
        log_error "Unit tests failed with exit code $unit_test_exit_code"
        
        # Extract failed tests
        echo "$unit_test_output" | grep -E "(FAIL|ERROR)" | while read -r line; do
            log_error "Test failure: $line"
        done
        
        return 1
    fi
    
    return 0
}

# Run integration tests
run_integration_tests() {
    log_section "Integration Test Execution"
    
    log_info "Executing make test-integration..."
    local integration_test_output
    local integration_test_exit_code=0
    
    # Capture container startup timing
    local start_time=$(date +%s)
    
    integration_test_output=$(make test-integration 2>&1) || integration_test_exit_code=$?
    echo "$integration_test_output" | tee -a "$LOG_FILE"
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $integration_test_exit_code -eq 0 ]; then
        log_success "Integration tests passed (duration: ${duration}s)"
        
        # Parse test results
        local total_tests=$(echo "$integration_test_output" | grep -o "RUN.*Test" | wc -l || echo "0")
        local passed_tests=$(echo "$integration_test_output" | grep -o "PASS.*Test" | wc -l || echo "0")
        
        log_info "Integration test summary: $passed_tests/$total_tests tests passed"
    else
        log_error "Integration tests failed with exit code $integration_test_exit_code (duration: ${duration}s)"
        
        # Extract failed tests and container logs
        echo "$integration_test_output" | grep -E "(FAIL|ERROR)" | while read -r line; do
            log_error "Test failure: $line"
        done
        
        # Try to get container logs if available
        log_info "Checking for test container logs..."
        docker ps -a --filter "label=org.testcontainers" --format "table {{.Names}}\t{{.Status}}" 2>/dev/null | tee -a "$LOG_FILE" || true
        
        return 1
    fi
    
    return 0
}

# Cleanup test resources
cleanup_test_resources() {
    log_section "Cleanup"
    
    log_info "Cleaning up test containers..."
    
    # Stop and remove testcontainers
    local containers=$(docker ps -aq --filter "label=org.testcontainers" 2>/dev/null || true)
    if [ -n "$containers" ]; then
        docker stop $containers 2>/dev/null || true
        docker rm $containers 2>/dev/null || true
        log_success "Test containers cleaned up"
    else
        log_info "No test containers to clean up"
    fi
    
    # Clean up test volumes
    local volumes=$(docker volume ls -q --filter "label=org.testcontainers" 2>/dev/null || true)
    if [ -n "$volumes" ]; then
        docker volume rm $volumes 2>/dev/null || true
        log_success "Test volumes cleaned up"
    else
        log_info "No test volumes to clean up"
    fi
    
    # Clean up compiled test binaries
    find . -name "*.test" -delete 2>/dev/null || true
    
    log_success "Cleanup completed"
}

# Analyze results and generate summary
analyze_results() {
    log_section "Results Analysis"
    
    # Count total issues found
    local error_count=$(grep -c "\[ERROR\]" "$LOG_FILE" || echo "0")
    local warning_count=$(grep -c "\[WARNING\]" "$LOG_FILE" || echo "0")
    
    log_info "Analysis complete:"
    log_info "  - Errors found: $error_count"
    log_info "  - Warnings found: $warning_count"
    
    if [ "$error_count" -eq 0 ]; then
        log_success "All tests passed successfully!"
        return 0
    else
        log_error "Test execution completed with $error_count errors"
        return 1
    fi
}

# Main execution function
main() {
    log_section "Test Runner Script Started"
    log_info "Starting comprehensive test execution at $(date)"
    log_info "Working directory: $PROJECT_ROOT"
    
    # Initialize log file
    echo "# Test Execution Results - $(date)" > "$LOG_FILE"
    
    local exit_code=0
    local steps_completed=0
    local total_steps=8
    
    # Execute all steps
    if validate_environment; then
        steps_completed=$((steps_completed + 1))
        log_info "Step $steps_completed/$total_steps completed"
    else
        exit_code=1
    fi
    
    if [ $exit_code -eq 0 ] && verify_dependencies; then
        steps_completed=$((steps_completed + 1))
        log_info "Step $steps_completed/$total_steps completed"
    else
        exit_code=1
    fi
    
    if [ $exit_code -eq 0 ] && generate_mocks; then
        steps_completed=$((steps_completed + 1))
        log_info "Step $steps_completed/$total_steps completed"
    else
        exit_code=1
    fi
    
    if [ $exit_code -eq 0 ] && compilation_check; then
        steps_completed=$((steps_completed + 1))
        log_info "Step $steps_completed/$total_steps completed"
    else
        exit_code=1
    fi
    
    if [ $exit_code -eq 0 ] && run_unit_tests; then
        steps_completed=$((steps_completed + 1))
        log_info "Step $steps_completed/$total_steps completed"
    else
        exit_code=1
    fi
    
    if [ $exit_code -eq 0 ] && run_integration_tests; then
        steps_completed=$((steps_completed + 1))
        log_info "Step $steps_completed/$total_steps completed"
    else
        exit_code=1
    fi
    
    # Always run cleanup and analysis
    cleanup_test_resources
    steps_completed=$((steps_completed + 1))
    log_info "Step $steps_completed/$total_steps completed"
    
    analyze_results || exit_code=1
    steps_completed=$((steps_completed + 1))
    log_info "Step $steps_completed/$total_steps completed"
    
    # Final summary
    log_section "Test Runner Completed"
    if [ $exit_code -eq 0 ]; then
        log_success "All $total_steps steps completed successfully!"
        log_success "Test results logged to: $LOG_FILE"
    else
        log_error "Test runner completed with errors (completed $steps_completed/$total_steps steps)"
        log_error "Check $LOG_FILE for detailed error information"
    fi
    
    exit $exit_code
}

# Trap to ensure cleanup on script exit
trap cleanup_test_resources EXIT

# Run main function
main "$@"
