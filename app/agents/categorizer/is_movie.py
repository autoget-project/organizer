import asyncio
from typing import Tuple

from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer
from pydantic_ai.usage import RunUsage

from ..ai import allowedTools, model
from ..models import PlanRequest
from .models import IsMovieResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents movie content and detecting if it's anime based solely on filenames, directory paths, and available metadata. Return an IsMovieResponse object with "is_movie" (yes/no/maybe), "is_anim", "movie_name", "movie_name_in_chinese", "release_year", "language", and "reason".

Please repeat the prompt back as you understand it.

Classification Rules for is_movie field:
- "yes": Files are clearly movies (single narrative films, standalone content, no episode structure)
- "no": Files are clearly NOT movies (TV series, music videos, software, educational content, etc.)
- "maybe": Files could be movies but lack sufficient information for definitive classification

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", "find_by_imdb_id_result", etc.
   - Treat all files as a single group to determine if they represent a movie.

2. Priority Order for Analysis:
   - First check for "find_by_imdb_id_result" in metadata - this is the highest authority
   - Then check other metadata fields (title, description, tags)
   - Finally analyze filename and directory patterns

3. IMDb Metadata Analysis (when find_by_imdb_id_result exists):
   - If result contains "movie_results": Strong evidence for "yes"
   - If result contains "tv_results": Strong evidence for "no" (it's a TV series)
   - Extract movie_name, release_year, language from the IMDb result
   - Check genre information in IMDb result for anime indicators
   - Use IMDb data to override filename-based assumptions

4. Return "no" when:
   - find_by_imdb_id_result contains "tv_results" (TV series data)
   - Files have TV series patterns: season/episode numbering (S01E01, 1x01, etc.)
   - Files are music videos, concerts, or audio-only content
   - Files are software, games, applications, or educational tutorials
   - Files are sports events, news broadcasts, or YouTube content
   - Files are porn content (use porn categorizers instead)
   - Files are books, audiobooks, or text-based content
   - Web search confirms content is not a movie but something else
   - Files are multiple short clips that don't form a coherent movie

5. Return "yes" when:
   - find_by_imdb_id_result contains "movie_results" (movie data)
   - File types: .mp4, .mkv, .avi, .mov, .wmv, .flv, .webm video files
   - Filename patterns:
     - Movie titles with years (e.g., "Inception.2010.1080p.BluRay.x264.mkv")
     - Single large video files (typical movie length)
     - Movie naming conventions: "Movie.Name.Year.Resolution.Source.Codec.ext"
     - No season/episode numbering patterns
   - Directory structure:
     - Movie folders with single video file
     - Movie folders like "Movies", "Films", "Cinema"
     - Year-based organization (e.g., "Movies/2023/", "Inception (2010)/")
     - Director or studio folders
   - Metadata indicators:
     - "movie", "film", "cinema", "director", "cast" in metadata fields
     - Movie-related tags like "action", "comedy", "drama", "thriller", "horror"
     - Runtime information indicating movie length (90-180 minutes typical)
     - Production company, director, actor information
   - Web search confirms the title is a movie

6. Anime movie detection criteria (for is_anim field):
   - Japanese anime films (Studio Ghibli, Makoto Shinkai, etc.)
   - Anime studios in metadata (Studio Ghibli, Madhouse, Kyoto Animation, etc.)
   - Japanese language content with English subtitles
   - Anime-specific file naming patterns (e.g., "Your.Name.2016.1080p.BluRay.x264.mkv")
   - Art style indicators in metadata or descriptions
   - Japanese titles with English translations
   - Anime film distributors (Crunchyroll, Funimation, etc.)
   - IMDb result genres indicating "Animation", "Anime", or similar

7. Language Detection:
   - Use find_by_imdb_id_result language information first if available
   - Then use search_movie original_language or language field if available
   - Finally infer from filenames and metadata

8. Analysis approach:
   - Always check for find_by_imdb_id_result first and prioritize it
   - Analyze the entire file set as one logical unit
   - Prefer metadata over filename cues
   - Consider directory structure and file patterns
   - Use search_movies to verify uncertain cases - search for movie titles, release information
   - When uncertain, select the closest matching response based on strongest evidence
   - Extract the movie name when identified (prefer IMDb result title)
   - Determine the release year when possible (prefer IMDb result year)
   - Provide brief reasoning for your decision, explicitly mentioning if IMDb data was used
"""


async def is_movie(req: PlanRequest, mcp: MCPServer) -> Tuple[IsMovieResponse, RunUsage]:
  a = Agent(
    name="is_movie_detector",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(IsMovieResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["search_movies"]),
  )
  res = await a.run(req.model_dump_json())
  return res.output, res.usage()


if __name__ == "__main__":
  from ..ai import metadataMcp, setupLogfire

  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "Sword Art Online the Movie - Progressive - Aria of a Starless Night.mkv",
      ],
    )

    res, usage = asyncio.run(is_movie(req, metadataMcp()))
    print(f"output: {res}")
    print(f"usage: {usage}")
