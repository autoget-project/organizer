from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsTVSeriesResponse

_INSTRUCTION = """\
Task: You are an AI agent that analyzes a group of media file paths and optional metadata to determine whether they represent a TV series and whether it is anime. You must always use `search_tv_shows` to retrieve additional show metadata before final classification.

Please repeat the prompt back as you understand it.

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent a TV series.

2. Preprocessing (Mandatory):
   - Extract potential show titles from file and directory names.
   - Invoke `search_tv_shows(<detected title or folder name>)`.
   - Use returned information (title, release_year, genres, episode_count, type, language, anime flag) to enrich your local metadata.
   - Prefer `search_tv_shows` results over local metadata if conflicting.

3. TV Series Detection:
   - Valid video extensions: .mp4, .mkv, .avi, .mov, .wmv,.
   - Episodic filename patterns: SxxExx, 1x01, Ep01, “Season 1 Episode 1”, consistent numbering.
   - Directory structure: “TV Shows”, “Series”, “Season 1”, or year-based folder names (e.g., “Show (2010-2015)”).
   - `search_tv_shows` confirmation: if type == "TV Series", "Series", or episode_count > 1 → `is_tv_series: true`.

4. Anime Detection:
   - Indicators: Japanese scripts in filenames/metadata, anime distributors (Crunchyroll, Funimation), `search_tv_shows` type = "Anime", or anime genres.
   - Prefer `search_tv_shows` confirmation for final `is_anime` decision.

5. Non-TV Series Exclusions:
   - Single movies (1 video file), concerts, music videos, tutorials, sports, or YouTube content unless episodic.
   - Documentaries only if confirmed as a series by `search_tv_shows`.

6. Language Detection:
   - Analyze filenames/metadata for language scripts (Japanese, Chinese, Korean, English).
   - Use `search_tv_shows` `original_language` or `language` field if available to confirm.

7. Output:
   - If `search_tv_shows` returns no match, rely on filename and metadata heuristics.
   - Always output reason describing what evidence was used.
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="is_tv_series_detector",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(IsTVSeriesResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["search_tv_shows"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "Two and a Half Men/two-and-a-half-men-s01e01.mkv",
        "Two and a Half Men/two-and-a-half-men-s01e02.mkv",
        "Two and a Half Men/two-and-a-half-men-s01e02.mkv",
      ],
      metadata={"title": "Two and a Half Men", "genre": "Comedy", "network": "CBS"},
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
