#!/bin/bash

# setup-deps.sh - Network-resilient dependency setup for nudgebot-api

set -e

echo "🔧 Setting up dependencies for nudgebot-api..."

# Function to try different proxy configurations
try_download() {
    local cmd="$1"
    local description="$2"
    
    echo "📦 $description..."
    
    # Try direct first (fastest if it works)
    echo "  🔄 Trying direct mode..."
    if GOPROXY=direct timeout 30 $cmd; then
        echo "  ✅ Success with direct mode"
        return 0
    fi
    
    # Try default proxy
    echo "  🔄 Trying default proxy..."
    if timeout 60 $cmd; then
        echo "  ✅ Success with default proxy"
        return 0
    fi
    
    # Try with longer timeout
    echo "  🔄 Trying with extended timeout..."
    if timeout 120 $cmd; then
        echo "  ✅ Success with extended timeout"
        return 0
    fi
    
    echo "  ⚠️  Failed to download, continuing with cached modules"
    return 1
}

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
fi

echo "ℹ️  Go version: $(go version)"

# Download core dependencies
echo "📥 Downloading core dependencies..."
try_download "go mod download" "Downloading all dependencies"

# Try to get specific dependencies that are commonly problematic
echo "📥 Ensuring critical dependencies are available..."

# Core testing dependencies
try_download "go get github.com/stretchr/testify@latest" "Updating testify"
try_download "go get go.uber.org/zap@latest" "Updating zap"
try_download "go get go.uber.org/mock@latest" "Updating mock"

# Testcontainers dependencies
try_download "go get github.com/testcontainers/testcontainers-go@latest" "Updating testcontainers"
try_download "go get github.com/testcontainers/testcontainers-go/modules/postgres@latest" "Updating postgres testcontainer"

echo "🔍 Verifying module integrity..."
if go mod verify; then
    echo "✅ Module verification successful"
else
    echo "⚠️  Module verification failed, but continuing"
fi

echo "🧹 Cleaning up module cache..."
go clean -modcache || echo "⚠️  Failed to clean mod cache, continuing"

echo "📋 Tidying go.mod..."
go mod tidy || echo "⚠️  Failed to tidy modules, continuing"

echo "✅ Dependency setup completed!"
echo ""
echo "📊 Dependency Summary:"
echo "  - Core modules: $(go list -m all | wc -l) total"
echo "  - Direct dependencies: $(go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all | grep -v '^$' | wc -l)"
echo ""
echo "🎯 Next steps:"
echo "  - Run 'make test-essential' to test the essential test suite"
echo "  - Run 'make generate-mocks' to generate mocks"
echo "  - Run 'make test-essential-services' to test services"
