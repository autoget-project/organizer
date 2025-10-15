run:
    test -f .local/env.sh && source .local/env.sh && uv run fastapi dev

format:
    uvx ruff format

lint:
    uvx ruff check && uvx ruff format --check

test:
    uv run pytest

adkweb:
    test -f .local/env.sh && source .local/env.sh && cd app && uv run adk web --port 8001
