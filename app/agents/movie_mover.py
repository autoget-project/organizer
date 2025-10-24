from os import path

from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from .ai import allowedTools, metadataMcp, model, setupLogfire
from .models import Category, PlanRequest, PlanResponse
from .oldcategorizer import CategoryResponse


def _get_instruction(category: CategoryResponse) -> str:
  return """\
Task: You are an AI system that organizes movie downloads into Jellyfin's preferred folder and file
naming conventions.

You must analyze a list of downloaded files, identify the movie they belong to, confirm the correct
title (preferably in Chinese), and produce a JSON plan describing how each file should be renamed
and moved.

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - JSON object containing:
     - "files": array of file paths.
     - "metadata": optional fields like title, description, etc.
2. Analyze:
   - Extract clues (movie title, year, language) from filenames.
   - Only keep video (.mp4, .mkv, .avi) and subtitle (.srt, .ass, .sub) files.
   - Exclude non-media (.jpg, .png, .nfo, etc.).
3. Confirm movie:
   - Use `search_movies` (TMDB) and prefer Chinese title if available.
   - If missing or uncertain, use `web_search` on Douban, IMDb, etc.
4. Construct new Jellyfin-compatible names:
   - Folder: `$PATH$/<Movie Name (Year)>`
   - Video: `<Movie Name (Year)>.ext`
   - Subtitle: `<Movie Name (Year)>.HumanLang.LangCode.ext`
   - Subtitle:
     - If language is identifiable: `<Movie Name (Year)>.<Language>.<ISO639-2>.ext`
       - Example: `泰坦尼克号 (2000).简体中文.chi.srt`
     - If language cannot be determined: `<Movie Name (Year)>.ext`
5. Edge cases:
   - If the file doesn't match a known movie or is an extra, `"action": "skip"`.
   - If multiple movies detected, separate into logical groups.
```
""".replace("$PATH$", path.join(category.category.name, category.language))


def agent(mcp: MCPServer, category: CategoryResponse) -> Agent:
  return Agent(
    name="movie_mover",
    model=model(),
    instructions=_get_instruction(category),
    output_type=ToolOutput(PlanResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["web_search", "search_movies"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv",
        "The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass",
        "The.Mad.Phoenix.1997/cover.jpg",
        "The.Mad.Phoenix.1997/behind the scenes.mp4.part",
      ],
    )

    a = agent(metadataMcp(), CategoryResponse(category=Category.movie, language="Chinese"))
    res = a.run_sync(req.model_dump_json())
    print(f"output: ${res.output}")
    print(f"usage: ${res.usage()}")
