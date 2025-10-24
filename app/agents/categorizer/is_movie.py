from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsMovieResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents movie content and detecting if it's anime based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

Specifics (each bullet contains specifics about the task):

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent a movie.

2. Movie detection criteria:
   - File types: Look for .mp4, .mkv, .avi, .mov, .wmv, .flv, .webm video files
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

3. Anime movie detection criteria:
   - Japanese anime films (Studio Ghibli, Makoto Shinkai, etc.)
   - Anime studios in metadata (Studio Ghibli, Madhouse, Kyoto Animation, etc.)
   - Japanese language content with English subtitles
   - Anime-specific file naming patterns (e.g., "Your.Name.2016.1080p.BluRay.x264.mkv")
   - Art style indicators in metadata or descriptions
   - Japanese titles with English translations
   - Anime film distributors (Crunchyroll, Funimation, etc.)

4. Non-movie exclusions:
   - TV series with season/episode numbering
   - Documentaries (unless clearly a documentary film)
   - Music videos or concerts
   - Software tutorials or educational content
   - Sports events or highlights
   - YouTube content
   - Short films or clips (under 60 minutes typically)

5. Analysis approach:
   - Analyze the entire file set as one logical unit
   - Prefer metadata over filename cues
   - Consider directory structure and file patterns
   - Use search_movies to verify uncertain cases - search for movie titles, release information
   - When uncertain, select the closest matching response based on strongest evidence
   - Extract the movie name when identified
   - Determine the release year when possible
   - Provide brief reasoning for your decision
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="is_movie_detector",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(IsMovieResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["search_movies"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "Sword Art Online the Movie - Progressive - Aria of a Starless Night.mkv",
      ],
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
