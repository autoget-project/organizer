# AutoGet File Organizer

This is a Python-based FastAPI web service designed to intelligently organize files downloaded by AutoGet. It leverages an AI agent to categorize files and generate optimal organization plans.

## API Endpoints

### POST /v1/plan

This endpoint allows you to submit a list of files and associated metadata to generate an organization plan. The AI agent will analyze the input and propose actions (e.g., move, ignore) for each file.

**Request Payload:**

```json5
{
    "files": [
        "path/to/file1",
        "path/to/file2",
        ...
    ],
    "metadata": {
        "title": "Document Title",
        // ...
    }
}
```

**Response Payload:**

```json5
{
    "plan": [
        {
            "file": "path/to/file1",
            "action": "move",
            "target": "target/path/to/file1"
        },
        {
            "file": "path/to/file2",
            "action": "ignore"
        }
    ]
}
```
*   `action`: Can be `"move"` or `"ignore"`.
*   `target`: The proposed destination path if the action is `"move"`, otherwise `null`.

### POST /v1/execute

This endpoint executes a previously generated organization plan.

**Request Payload:**

```json5
{
    "plan": [
        {
            "file": "path/to/file1",
            "action": "move",
            "target": "target/path/to/file1"
        },
        // ...
    ]
}
```

**Response:**

Returns a `200 OK` status upon successful execution of the plan.

## AI Agent Architecture

The AI Agent, built with pydantic-ai. It operates in two main steps:

-   **Categorization**: The AI analyzes file metadata (if provided) and filenames to determine the most appropriate category for each request.
-   **Action Generation**: Based on the determined category, a specialized category agent generates a precise action plan, such as moving a file to a specific folder (e.g., "move file X to folder Y").

## Getting Started

To set up and run the File Organizer, follow these steps:

### Prerequisites

*   Python 3.13
*   `uv` for dependency management
*   Create your `.env` based on `.env.example`

## Usage

To start the FastAPI service for dev, run the following command:

```sh
uv run fastapi dev
```

The service will be accessible at `http://localhost:8000`. You can then use the API endpoints described above to organize your files.

## Docker Run

```sh
docker run -p 8000:8000 \
  --name organizer \
  -e LOGFIRE_TOKEN="your-logfire-token" \
  -e GROK_API_KEY="your-grok-api-key" \
  -e MODEL="grok:grok-4-fast-reasoning" \
  -e METADATA_MCP="http://your-mcp-server-url" \
  -e DOWNLOAD_COMPLETED_DIR="/app/downloads" \
  -e TARGET_DIR="/app/target" \
  -e JAV_ACTOR_FILE="/app/target/actor.json" \
  -v $(pwd)/downloads:/app/downloads \
  -v $(pwd)/target:/app/target \
  ghcr.io/autoget-project/organizer:main
```

### Environment Variables

When running with Docker, you need to provide these environment variables:

**Required:**
- `MODEL` - AI model name (e.g., `grok:grok-4-fast-reasoning`)
- `METADATA_MCP` - MCP server URL for metadata access
- `DOWNLOAD_COMPLETED_DIR` - Source directory containing files to organize
- `TARGET_DIR` - Target directory where organized files will be moved
- `JAV_ACTOR_FILE` - Path to JAV actor mapping file

**AI Provider (choose one):**
- `GROK_API_KEY` - For Grok API access
- `LM_STUDIO_API_BASE` - For LM Studio or other local API endpoints

**Optional:**
- `LOGFIRE_TOKEN` - For logging and observability

## Development

### Test single agent

```sh
just run-agent <agent_name>
```

You can check `app/agents` for agent name.

Or you can run their tests:

```sh
just test-agent <agent_name>
```
