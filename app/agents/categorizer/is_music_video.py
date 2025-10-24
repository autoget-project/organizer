from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsMusicVideoResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents music videos and detecting their language based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

Specifics (each bullet contains specifics about the task):

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent music videos.

2. Music video detection criteria:
   - File types: Look for .mp4, .mkv, .avi, .mov, .wmv, .flv, .webm, .m4v files
   - Filename patterns: 
     - Artist - Title format (e.g., "Taylor Swift - Shake It Off.mp4")
     - Music video keywords: "MV", "music video", "official video", "lyric video"
     - Live performance indicators: "live", "concert", "performance", "festival"
     - Video quality indicators: "1080p", "720p", "4K", "HD"
   - Directory structure:
     - Music video folders (e.g., "Music Videos", "MV Collection", "Official Videos")
     - Artist-based organization (e.g., "Artist Name/Music Videos/")
     - Year or album-based organization
   - Metadata indicators:
     - "artist", "song", "track", "album", "genre" in metadata fields
     - Music-related tags like "pop", "rock", "electronic", "hip-hop"
     - Video-specific metadata like duration, resolution, bitrate

3. Non-music video exclusions:
   - Regular movies or TV shows (no music-related indicators)
   - Concert films/full concerts (these are typically categorized differently)
   - Documentaries about music (unless focused on specific music videos)
   - Video tutorials or educational content
   - Regular video content without music elements

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
   - Use web_search to verify uncertain cases - search for artists, songs, or music videos
   - When uncertain, select the closest matching response based on strongest evidence
   - Provide brief reasoning for your decision
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="is_music_video_detector",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(IsMusicVideoResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["web_search"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "Taylor Swift - Shake It Off (Official Music Video).mp4",
        "Taylor Swift - Blank Space (Official Music Video).mp4",
        "Taylor Swift - Bad Blood (Official Music Video).mp4",
      ],
      metadata={
        "artist": "Taylor Swift",
        "genre": "Pop",
        "description": "Official music videos collection",
      },
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
