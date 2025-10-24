from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsMusicResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents music/audio content and detecting its language based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent music.
2. Detection rules & priorities:
   - Supported audio extensions: .mp3, .flac, .wav, .aac, .m4a, .ogg, .wma, .alac, .aiff.
   - Prefer evidence in this order: metadata > filename patterns > folder names > file extensions.
   - Filename patterns to identify songs:
       • "Artist - Title" or "Artist_Title"
       • Music-related keywords: "single", "track", "song", "remix", "instrumental", "OST", "soundtrack"
       • Album or artist folder names (e.g., `/Music/Drake/`, `/Albums/Taylor Swift/`)
       • Bitrate or quality tags like "320kbps", "FLAC", "HQ", "Hi-Res"
   - Treat repeated patterns across multiple files (e.g., several .mp3 files in one artist folder) as strong music evidence.
   - Use `web_search` to verify artist or track names only when ambiguity remains.
3. Non-music exclusions:
   - Exclude podcasts, audiobooks, interviews, lectures, voice memos, ringtones, and sound effects unless clear music indicators are present.
   - If folder or filename contains words like "podcast", "audiobook", "lecture", "interview", or "sound effect", classify as non-music.
4. Language detection:
   - Use the song's metadata (language field or title text) to infer language.
   - If missing, infer from artist or song title language; when ambiguous, use brief web lookup for artist/track language.
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
