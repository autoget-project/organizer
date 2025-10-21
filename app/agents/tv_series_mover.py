from os import path

from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from .ai import allowedTools, metadataMcp, model, setupLogfire
from .categorizer import CategoryResponse
from .models import Category, PlanRequest, PlanResponse


def _get_instruction(category: CategoryResponse) -> str:
  return """\
Task: You are an AI agent specialized in organizing and renaming TV series files for media
libraries like Jellyfin.

Your goal is to analyze a batch of downloaded files, identify which TV series each belongs to, and
generate a JSON plan to rename and move them into Jellyfin's standard directory structure. Accuracy
is prioritized — confirm series details using TMDB or online searches.

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - JSON object containing:
     - "files": array of file paths.
     - "metadata": optional fields like title, description, etc.
2. **File Analysis**:
   - Parse filenames to extract series name, season, and episode.
   - Process only video (.mp4, .mkv, .avi, etc.) and subtitle (.srt, .ass, .sub, etc.) files.
   - Ignore non-media files (.jpg, .png, .nfo, etc.).
3. **Identify Series**:
   - Use `search_tv_shows` (TMDB) to confirm title and year.
   - If uncertain, fallback to `web_search`.
   - Prefer the **Chinese title** if available; otherwise, use English.
4. **Jellyfin Naming Rules**:
   - Use the **year of the show's first season** for all folder and file naming.
   - Folder: `$PATH$/Series Name (Year)/Season XX`
   - Video: `Series Name (Year) SXXEYY.ext`
   - Subtitle:
     - If language is identifiable: `Series Name (Year) SXXEYY.<Language>.<ISO639-2>.ext`
       - Example: `权力的游戏 (2011) S01E01.简体中文.chi.srt`
     - If language cannot be determined: `Series Name (Year) SXXEYY.ext`
   - Place subtitles in the same season folder as their matching video and pair by episode number.
5. **Edge Cases**:
   - Mark ambiguous or unsupported files with `"action": "skip"`.
   - Group multi-season batches logically.
```
""".replace("$PATH$", path.join(category.category.name, category.language))


def agent(mcp: MCPServer, category: CategoryResponse) -> Agent:
  return Agent(
    name="tv_series_mover",
    model=model(),
    instructions=_get_instruction(category),
    output_type=ToolOutput(PlanResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["web_search", "search_tv_shows"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv",
        "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP2.mkv",
        "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass",
        "My.Date.with.a.Vampire.Season.02.2000/cover.jpg",
        "My.Date.with.a.Vampire.Season.02.2000/behind the scenes.mp4.part",
      ],
    )

    a = agent(metadataMcp(), CategoryResponse(category=Category.tv_series, language="Chinese"))
    res = a.run_sync(req.model_dump_json())
    print(f"output: ${res.output}")
    print(f"usage: ${res.usage()}")
