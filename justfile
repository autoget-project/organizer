run:
  test -f .env && uv run fastapi dev

run-agent name:
  test -f .env && uv run --env-file .env -m app.agents."{{name}}"

format:
  uvx ruff check --select I --fix
  uvx ruff format

lint:
  uvx ruff check && uvx ruff format --check

test:
  uv run pytest

alltest:
  test -f .env && uv run --env-file .env pytest
