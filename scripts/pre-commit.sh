#!/bin/sh

# Pre-commit hook for Go code quality checks

echo "🔍 Running pre-commit checks..."

# Check if this is a Go project commit
if git diff --cached --name-only | grep -q "\.go$"; then
    echo "📝 Formatting Go code..."
    cd backend && go fmt ./... || exit 1
    cd ../go-worker-sdk && go fmt ./... || exit 1
    cd ../worker-simulator && go fmt ./... || exit 1
    cd ..
    
    echo "🧪 Running tests..."
    cd backend && go test ./... || exit 1
    cd ..
    
    echo "🔎 Running linter..."
    cd backend && golangci-lint run ./... || exit 1
    cd ..
fi

echo "✅ Pre-commit checks passed!"
exit 0
