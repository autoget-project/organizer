from pydantic import BaseModel, Field
from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from .ai import allowedTools, metadataMcp, model, setupLogfire
from .models import Category, PlanRequest, category_list

_INSTRUCTION: str = (
  """\
Task: You are an AI classifier that analyzes downloaded files and metadata to determine both
content category and primary language.

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - JSON object containing:
     - "files": array of file paths.
     - "metadata": optional fields like title, description, etc.
2. Classification:
   - Assign a single category from this list: $CATEGORY_LIST$.
   - Treat all files as one group. Select the closest match if uncertain—no new categories allowed.
3. Detection:
   - Extract clues from filenames and metadata (titles, keywords, extensions, episode indicators).
   - Recognize file types (video, audio, image, document, etc.).
   - Detect dominant language (English, Chinese, Japanese, Korean).
4. Validation:
   - Use `search_movies`, `search_tv_shows`, `search_porn`, or `search_japanese_porn` to confirm
     title guesses.
   - If unsuccessful, fallback to `web_search` for external verification.
5. Output:
   - Return two fields:
     - `category`: selected from $CATEGORY_LIST$.
     - `language`: dominant detected language.
"""
).replace("$CATEGORY_LIST$", ", ".join(category_list))


class CategoryResponse(BaseModel):
  category: Category = Field(description="The detected category of the download.")
  language: str = Field(description="The detected language of the download.")


def agent(mcp: MCPServer) -> Agent:
  a = Agent(
    model=model(),
    name="categorizer",
    instructions=_INSTRUCTION,
    output_type=ToolOutput(CategoryResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(
      [
        "web_search",
        "search_movies",
        "search_tv_shows",
        "search_porn",
        "search_japanese_porn",
      ]
    ),
  )

  return a


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "The.Lychee.Road.2025.E01.mkv",
      ],
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: ${res.output}")
    print(f"usage: ${res.usage()}")
