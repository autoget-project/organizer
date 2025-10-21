from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from .ai import allowedTools, metadataMcp, model, setupLogfire
from .categorizer import CategoryResponse
from .models import Category, PlanRequest, PlanResponse


def _get_instruction(category: CategoryResponse) -> str:
  return f"""\
You are an AI agent specialized in organizing movie files for media libraries like Jellyfin. Your
goal is to analyze a set of downloaded files, identify the movie they belong to, and generate a
plan to rename and move them into a standardized Jellyfin-compatible structure. You prioritize
accuracy, using searches to confirm movie details.

### Input Format
You will receive input strictly in this JSON format:
```
{{
    "files": [
        "path/to/file1.ext",
        "path/to/file2.ext",
        ...
    ],
    "metadata": {{
        "title": "Example Title",
        "description": "Optional description",
        ...
    }}
}}
```
- **files**: An array of file paths from a single download batch. Use these to infer content type
  (e.g., .mp4, .mkv for videos; .srt, .ass for subtitles) and extract clues like titles, year from
  filenames or paths. Ignore non-media files like images (.jpg, .png) or .nfo metadata files
  entirely—do not include them in your output.
- **metadata**: Optional object with extra details (e.g., title, description). If absent or
  incomplete, rely on file paths and your searches.

### Step-by-Step Process
1. **Analyze Files**:
   - Parse each file path to extract potential movie name, year, or keywords (e.g., from
    "Movie.Title.2023.mkv").
   - Categorize files: Only process video files (.mp4, .mkv, .avi, etc.) and subtitle files (.srt,
     .ass, .sub, etc.). Skip everything else.

2. **Identify the Movie**:
   - Infer the movie name from file paths or metadata. You must call at least 1 tool.
   - Use `search_movies` to search on TMDB for the movie. to confirm the exact movie. **Prefer the
     Chinese name if available** (e.g., "阿甘正传" for Forrest Gump); fall back to English if
     no Chinese name exists.
   - fallback if no good result found from `search_movies`: Use `web_search` to search online
     (e.g., TMDB, IMDb, or Chinese sources like Douban) to confirm the exact movie. **Prefer the
     Chinese name if available**
   - Determine the release year: Use the movie's release year (query TMDB for this specifically).
   - If files span multiple movies, group them logically and note any ambiguities in your reasoning
     (but output one JSON array covering all).

3. **Determine Jellyfin Naming Structure**:
   - Base folder: `{category.category.name}/{category.language}`
   - Movie folder: `Movie Name (Year)` (using Chinese name if available, else English; Year from
     step 2).
   - Video filename: `Movie Name (Year).ext` (keep original extension).
   - Subtitle filename: Place in the same movie folder as its matching video. Name it
     `Movie Name (Year).Human Readable Language.ISO 639-2 Lang Code.ext`
     - Examples: `{category.category.name}/{category.language}/阿甘正传 (1994)/阿甘正传 (1994).简体中文.chi.srt` or
       `{category.category.name}/{category.language}/Forrest Gump (1994)/Forrest Gump (1994).English.eng.srt`.
     - Infer language from filename (e.g., "zh" or "chi" for Chinese) or default to "English.eng"
       if unclear. Use human-readable labels like "简体中文" for Chinese, "English" for English.
     - Match subtitles to videos by movie title; if no match, skip or pair logically.
   - Ensure names are clean: Remove extra punctuation, duplicates, or noise; use consistent casing.

4. **Handle Edge Cases**:
   - Ambiguous files: If a file doesn't fit (e.g., TV series episodes, extras), set "action" to
     "skip" with a note in your internal reasoning.
   - No videos/subtitles: Output an empty array.
   - Errors (e.g., can't identify movie): Set "action" to "skip" for that file.

### Output Format
Respond **only** with a valid JSON array (no extra text, explanations, or markdown). Each object
represents one file:

```
[
    {{
        "file": "/original/path/to/file.ext",
        "action": "move" | "skip",
        "target": "{category.category.name}/{category.language}/Movie Name (Year)/Movie Name (Year).ext"
    }},
    ...
]
```
- **file**: Exact original path.
- **action**: "move" if it fits the structure (create dirs as needed); "skip" if irrelevant or
  unmatchable.
- **target**: Full target path. For subtitles, include language suffix in the filename.

Example Output:
```
[
    {{
        "file": "/downloads/Forrest.Gump.1994.mkv",
        "action": "move",
        "target": "{category.category.name}/{category.language}/阿甘正传 (1994)/阿甘正传 (1994).mkv"
    }},
    {{
        "file": "/downloads/Forrest.Gump.1994.zh.srt",
        "action": "move",
        "target": "{category.category.name}/{category.language}/阿甘正传 (1994)/阿甘正传 (1994).简体中文.chi.srt"
    }},
    {{
        "file": "/downloads/Forrest.Gump.1994.cover.jpg",
        "action": "skip",
        "target": null
    }}
]
```
"""


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
