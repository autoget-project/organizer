# AutoGet Organizer Quality Gates & Tasks

# Default recipe: show help summary
default:
  @just --list

# Quality Gate: compile/build the project
build:
  @if [ -f go.mod ]; then go build ./...; else echo "go.mod not found, skipping build"; fi

# Quality Gate: execute unit/integration tests
test:
  @if [ -f go.mod ]; then go test -v ./...; else echo "go.mod not found, skipping test"; fi

# Quality Gate: format source code
fmt:
  @if [ -f go.mod ]; then go fmt ./...; else echo "go.mod not found, skipping fmt"; fi

# Quality Gate: run static analysis and linting
lint:
  @if [ -f go.mod ]; then golangci-lint run; else echo "go.mod not found, skipping lint"; fi

# Run tests for specific agent or pipeline component
# Examples:
#   just test-agent stage1_classifier
#   just test-agent stage3_planner
#   just test-agent tv_planner
test-agent name:
  #!/usr/bin/env bash
  if [ ! -f go.mod ]; then
    echo "go.mod not found, skipping test-agent"
    exit 0
  fi
  target="{{name}}"
  if [ -d "$target" ]; then
    go test -v "./$target/..."
  elif [ -d "internal/pipeline/$target" ]; then
    go test -v "./internal/pipeline/$target/..."
  elif [ -d "internal/$target" ]; then
    go test -v "./internal/$target/..."
  else
    go test -v ./... -run "$target"
  fi

# Run end-to-end (E2E) tests
test-e2e:
  @if [ -f go.mod ]; then go test -v ./test/e2e/...; else echo "go.mod not found, skipping test-e2e"; fi

# Run the HTTP service
run:
  @if [ -f go.mod ]; then go run ./cmd/server/main.go; else echo "go.mod not found, cannot run"; fi
