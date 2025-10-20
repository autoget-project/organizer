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

```bash
uv run fastapi dev
```

The service will be accessible at `http://localhost:8000`. You can then use the API endpoints described above to organize your files.

## Development

### Test single agent

```
just run-agent <agent_name>
```

You can check `app/agents` for agent name.

Or you can run test:

```
uv run --env-file=.env pytest app/agents/xxx_test.py
```
