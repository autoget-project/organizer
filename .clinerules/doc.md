# Cline Rules and Guidelines for AutoGet File Organizer

This document outlines the rules and guidelines for the Cline agent when contributing to the 'AutoGet File Organizer' project. Adhering to these guidelines ensures consistency, maintainability, and alignment with project standards.

## 1. Project Setup & Dependencies

*   **Python Version**: All development and deployment must target **Python 3.13**. This is enforced by `pyproject.toml`.
*   **Dependency Management**: Use `uv` for all dependency management operations (installing, adding, or removing packages).
    *   **Installation**: `uv pip install -r requirements.txt`
    *   **Development Server**: `uv run fastapi dev`
    *   Avoid using `pip` directly.

## 2. Code Style & Quality

*   **Linting and Formatting**: Adhere strictly to the `ruff` configuration defined in `pyproject.toml`.
    *   **Line Length**: 100 characters.
    *   **Indent Width**: 2 spaces.
    *   **Target Python Version**: 3.13.
    *   **Quote Style**: Double quotes.
    *   Always run `ruff format` and `ruff check --fix` before committing any changes.

## 3. API Design

*   **FastAPI Best Practices**: Follow standard FastAPI conventions for defining API endpoints, request/response models, and error handling.
    *   Use Pydantic models for robust request and response validation.
    *   Implement appropriate HTTP status codes and clear error responses.
*   **Endpoint Consistency**: Ensure API endpoints (`POST /v1/plan`, `POST /v1/execute`) and their respective request/response payloads, as documented in `README.md`, are consistently maintained in the codebase. Any changes must be reflected in both the code and the documentation.

## 4. AI Agent Development

*   **Core Libraries**: The AI agent functionality relies on `google-adk` (Google Agent Development Kit) and `litellm`.
    *   `google-adk>=1.15.1`
    *   `litellm>=1.77.5`
*   **Integration**: Ensure proper and efficient integration of these libraries. Refer to their official documentation for optimal usage and configuration.
*   **Categorization Logic**: The AI's categorization process should be robust, analyzing file metadata and filenames to determine the most appropriate category for each request.
*   **Action Generation**: The action generation should be precise and safe, with specialized category agents generating clear move plans (e.g., "move file X to folder Y").

## 5. File Organization Logic

*   **Clarity and Robustness**: Maintain clear, well-documented, and robust logic for all file categorization and action generation processes.
*   **Safety**: Prioritize the safety and integrity of user files. Ensure that move plans are well-validated to prevent data loss or incorrect organization.
*   **Edge Cases**: Consider and handle various edge cases related to file types, missing metadata, and target path conflicts.
