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
# Falls back to go vet when golangci-lint is not installed.
lint:
	#!/usr/bin/env bash
	if [ ! -f go.mod ]; then
		echo "go.mod not found, skipping lint"
		exit 0
	fi
	if command -v golangci-lint >/dev/null 2>&1; then
		golangci-lint run
	else
		echo "golangci-lint not found, falling back to go vet"
		go vet ./...
	fi

# Run tests for specific agent or pipeline component. Accepts both friendly
# aliases and existing directory names / paths:
#   just test-agent classifier        # stage1_classifier
#   just test-agent enricher          # stage2_enricher
#   just test-agent tv_planner        # TV planner tests inside stage3_planner
#   just test-agent movie_planner
#   just test-agent bango_planner
#   just test-agent simple_planner
#   just test-agent porn_planner
#   just test-agent subtitle          # stage4 subtitle pairing tests
#   just test-agent security          # full stage4 security regression suite
#   just test-agent executor          # internal/service
#   just test-agent handlers          # internal/handler
#   just test-agent stage1_classifier # raw directory probing still works
test-agent name:
  #!/usr/bin/env bash
  if [ ! -f go.mod ]; then
    echo "go.mod not found, skipping test-agent"
    exit 0
  fi
  target="{{name}}"
  case "$target" in
    classifier) \
      go test -v ./internal/pipeline/stage1_classifier/... ;;
    enricher) \
      go test -v ./internal/pipeline/stage2_enricher/... ;;
    tv_planner) \
      go test -v ./internal/pipeline/stage3_planner/... -run 'TVPlanner' ;;
    movie_planner) \
      go test -v ./internal/pipeline/stage3_planner/... -run 'MoviePlanner' ;;
    bango_planner) \
      go test -v ./internal/pipeline/stage3_planner/... -run 'BangoPlanner|ResolveBangoTargetDir' ;;
    simple_planner) \
      go test -v ./internal/pipeline/stage3_planner/... -run 'SimplePlan' ;;
    porn_planner) \
      go test -v ./internal/pipeline/stage3_planner/... -run 'PornPlanner' ;;
    subtitle) \
      go test -v ./internal/pipeline/stage4_postprocess/... -run 'Subtitle|ReadSubtitlePreview' ;;
    security) \
      go test -v ./internal/pipeline/stage4_postprocess/... ;;
    executor) \
      go test -v ./internal/service/... ;;
    handlers) \
      go test -v ./internal/handler/... ;;
    e2e) \
      go test -v ./tests/e2e/... ;;
    *) \
      if [ -d "$target" ]; then
        go test -v "./$target/..."
      elif [ -d "internal/pipeline/$target" ]; then
        go test -v "./internal/pipeline/$target/..."
      elif [ -d "internal/$target" ]; then
        go test -v "./internal/$target/..."
      elif [ -d "tests/$target" ]; then
        go test -v "./tests/$target/..."
      else
        go test -v ./... -run "$target"
      fi ;;
  esac

# Run end-to-end (E2E) tests
test-e2e:
  @if [ -f go.mod ]; then go test -v ./tests/e2e/...; else echo "go.mod not found, skipping test-e2e"; fi

# Shortcut alias for the end-to-end suite
e2e: test-e2e

# Run the HTTP service
run:
  @if [ -f go.mod ]; then go run ./cmd/server/main.go; else echo "go.mod not found, cannot run"; fi
