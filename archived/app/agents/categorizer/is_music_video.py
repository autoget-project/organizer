from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, model
from ..models import PlanRequest
from .models import IsMusicVideoResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents music videos and detecting their language based solely on filenames, directory paths, and available metadata. Return an IsMusicVideoResponse object with "is_music_video" (yes/no/maybe), "language", and "reason".

Please repeat the prompt back as you understand it.

Classification Rules for is_music_video field:
- "yes": Files are clearly music videos (visual representations of musical performances or songs)
- "no": Files are clearly NOT music videos (movies, TV shows, regular videos, documentaries, etc.)
- "maybe": Files could be music videos but lack sufficient information for definitive classification

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent music videos.

2. Return "no" when:
   - Files are movies or TV shows (use movie/TV categorizers instead)
   - Files have season/episode numbering patterns or series structures
   - Files are documentaries, educational content, or tutorials
   - Files are porn content (use porn categorizers instead)
   - Files are concerts, live performances, or events (unless specifically music video format)
   - Files are regular videos with no musical elements (vlogs, interviews, etc.)
   - File paths contain keywords like "movie", "episode", "documentary", "tutorial", "lecture"
   - Web search confirms content is not a music video but something else

3. Return "yes" when:
   - Video extensions: .mp4, .mkv, .avi, .mov, .wmv, .flv, .webm, .m4v.
   - Filename / pattern indicators:
     • "Artist - Title" format typical of music videos
     • Keywords like "MV", "music video", "official video", "lyric video"
     • Live performance indicators "live", "concert", "performance", "festival" (when in music video context)
     • Visual music indicators like "visualizer", "animation", "remix video"
   - Metadata indicators:
     • Artist, song title, album information
     • Music video-related tags and descriptions
     • Director or production information typical of music videos
   - Directory structure indicating music video organization
   - Dominant or repeated patterns across multiple files raise confidence (e.g., many files in "Music Videos" folder, or many filenames with "MV").
   - Web search confirms the content is music video

4. Detection rules & priorities:
   - Prefer signals in this order: metadata > filename patterns > directory names > file extensions.
   - Use web_search mcp tool to verify ambiguous artist/title cases or to infer language when metadata is missing.
   - Look for consistent musical themes and artist/song information

5. Language detection:
   - Prefer explicit metadata language fields, then artist/title language cues; if still ambiguous, perform brief web verification for the song/artist.
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
  from ..ai import metadataMcp, setupLogfire

  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "Luis Fonsi - Despacito.mp4",
      ],
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
