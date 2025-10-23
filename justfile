run:
  test -f .env && uv run --env-file .env fastapi dev

# Run agent directly (supports both main agents and subpackages)
# Examples:
#   just run-agent categorizer
#   just run-agent utils.utils
run-agent name:
  test -f .env && uv run --env-file .env -m app.agents."{{name}}"

# Run tests for specific agent (supports both main agents and subpackages)
# Examples:
#   just test-agent categorizer
#   just test-agent utils.utils
test-agent name:
  #!/usr/bin/env bash
  # Convert dot notation to path for test files
  if echo "{{name}}" | grep -q "\."; then
    # Subpackage case: convert "utils.porn" to "app/agents/utils/porn_test.py"
    test_path="app/agents/$(echo '{{name}}' | sed 's/\./\//g')_test.py"
  else
    # Main package case: "categorizer" to "app/agents/categorizer_test.py"
    test_path="app/agents/{{name}}_test.py"
  fi
  test -f .env && uv run --env-file .env pytest "$test_path" -vv

format:
  uvx ruff check --select I --fix
  uvx ruff format

lint:
  uvx ruff check && uvx ruff format --check

test:
  uv run pytest

alltest:
  test -f .env && uv run --env-file .env pytest
