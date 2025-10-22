# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Python-based FastAPI web service that intelligently organizes files downloaded by AutoGet using AI agents. The system uses pydantic-ai to categorize files and generate organization plans.

## Development Commands

### Running the Service
- `just run` - Start the FastAPI development server (requires .env file)
- `uv run fastapi dev` - Alternative way to start the server

### Testing
- `just test` - Run all tests
- `just alltest` - Run all tests with environment variables loaded
- `just test-agent <agent_name>` - Run tests for a specific agent (e.g., `just test-agent categorizer`)
- `uv run pytest` - Direct pytest execution

### Agent Development
- `just run-agent <agent_name>` - Run a specific agent directly (e.g., `just run-agent categorizer`)
- Available agents: `categorizer`, `movie_mover`, `tv_series_mover`, `ai`

### Code Quality
- `just format` - Format code with ruff
- `just lint` - Run linting checks
- `uvx ruff check` - Run ruff linting
- `uvx ruff format` - Format code with ruff

## Architecture

### Core Components

1. **FastAPI Application** (`app/main.py`)
   - Main web service with two endpoints: `/v1/plan` and `/v1/execute`
   - Handles environment variable validation and startup checks
   - Executes file move operations

2. **AI Agent System** (`app/agents/`)
   - **Categorizer** (`categorizer.py`) - Determines file category based on metadata and filename
   - **Movie Mover** (`movie_mover.py`) - Handles movie file organization
   - **TV Series Mover** (`tv_series_mover.py`) - Handles TV series organization
   - **Runner** (`runner.py`) - Orchestrates the agent workflow
   - **AI Models** (`ai.py`) - Provides AI model configuration and MCP server setup

3. **Data Models** (`app/agents/models.py`)
   - Defines categories: movie, tv_series, anim_tv_series, anim_movie, photobook, porn, audio_book, book, music, music_video
   - Specialized porn categories: jav, porn, jav_vr, porn_vr, chinese
   - Request/response models for API endpoints

### Agent Workflow

1. **Categorization Phase**: AI analyzes file metadata and filenames to determine category
2. **Action Generation Phase**: Based on category, specialized agent generates organization plan
3. **Simple Categories**: photobook, audio_book, book, music, music_video use simple move logic
4. **Complex Categories**: movie, anim_movie, tv_series, anim_tv_series use specialized agents

### Environment Variables

Required environment variables:
- `MODEL` - AI model name
- `METADATA_MCP` - MCP server URL for metadata
- `DOWNLOAD_COMPLETED_DIR` - Source directory for files
- `TARGET_DIR` - Target directory for organized files
- `XAI_API_KEY` or `LM_STUDIO_API_BASE` - AI provider configuration

Optional:
- `LOGFIRE_TOKEN` - For logging/observability
- `GROK_API_KEY` - Alternative AI provider

### Directory Structure

The service expects target directories to be pre-created:
- `{TARGET_DIR}/{category}/` for each main category
- `{TARGET_DIR}/porn/{porn_category}/` for adult content categories

### API Usage

POST `/v1/plan` - Generate organization plan
POST `/v1/execute` - Execute generated plan

Both endpoints use the same `PlanAction` model with actions: "move" or "skip".

## Key Libraries and Context

- **pydantic-ai** (context7 id: `pydantic/pydantic-ai`) - Primary AI agent framework
- **FastAPI** - Web framework for the API service
- **pydantic** - Data validation and modeling
- **pytest** - Testing framework with async support

## Development Notes

- Uses Python 3.13+ and `uv` for dependency management
- Code formatting with ruff (line length: 100, 2-space indentation)
- Tests use pytest with async support
- AI agents use pydantic-ai with MCP (Model Context Protocol) for metadata access
- The service validates target directories exist on startup
