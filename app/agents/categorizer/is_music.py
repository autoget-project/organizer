from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsMusicResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents music/audio content and detecting its language based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

Specifics (each bullet contains specifics about the task):

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent music.

2. Music detection criteria:
   - File types: Look for .mp3, .m4a, .flac, .wav, .aac, .ogg, .wma, .opus files
   - Filename patterns: 
     - Artist - Title format (e.g., "The Beatles - Hey Jude.mp3")
     - Track numbers (e.g., "01 - Song Title.mp3")
     - Album/Artist/Track structure in directories
     - Music-related keywords: "song", "track", "album", "single", "remix", "live"
   - Directory structure:
     - Album folders with artist names (e.g., "Artist Name/Album Name/")
     - Music folders like "Music", "Albums", "Singles", "Compilations"
     - Year-based organization (e.g., "2023", "2020s")
   - Metadata indicators:
     - "artist", "album", "track", "genre", "year" in metadata fields
     - Music-related tags like "rock", "pop", "jazz", "classical", "electronic"
     - Bitrate, duration, or other audio-specific metadata

3. Non-music exclusions:
   - Podcast files with "podcast", "episode", "show" indicators
   - Audiobooks (should be categorized as audio_book)
   - Sound effects or samples (short duration, generic names)
   - Voice recordings or memos
   - Music videos (video files with music content - these are music_video category)

4. Language detection criteria:
   - Japanese: Hiragana/Katakana/Kanji characters in filenames or metadata
   - Chinese: Chinese characters (简体/繁體) in filenames or metadata
   - Korean: Hangul characters in filenames or metadata
   - English: Latin script with English words; absence of East Asian scripts
   - Other: if none apply clearly

5. Analysis approach:
   - Analyze the entire file set as one logical unit
   - Prefer metadata over filename cues
   - Consider directory structure and dominant patterns
   - Use web_search to verify uncertain cases - search for artists, albums, or songs
   - When uncertain, select the closest matching response based on strongest evidence
   - Provide brief reasoning for your decision
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="is_music_detector",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(IsMusicResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["web_search"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "The Beatles/Abbey Road/01 - Come Together.mp3",
        "The Beatles/Abbey Road/02 - Something.mp3",
        "The Beatles/Abbey Road/03 - Maxwell's Silver Hammer.mp3",
      ],
      metadata={"artist": "The Beatles", "album": "Abbey Road", "genre": "Rock"},
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
