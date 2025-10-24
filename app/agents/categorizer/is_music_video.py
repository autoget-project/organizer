from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsMusicVideoResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents music videos and detecting their language based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent music videos.
2. Detection rules & priorities:
   - Consider only these video extensions: .mp4, .mkv, .avi, .mov, .wmv, .flv, .webm, .m4v.
   - Prefer signals in this order: metadata > filename patterns > directory names > file extensions.
   - Filename / pattern indicators: "Artist - Title" format; keywords like "MV", "music video", "official video", "lyric video"; live indicators "live", "concert", "performance", "festival".
   - Dominant or repeated patterns across multiple files raise confidence (e.g., many files in "Music Videos" folder, or many filenames with "MV").
   - Use web_search mcp tool to verify ambiguous artist/title cases or to infer language when metadata is missing.
3. Non-music-video exclusions:
   - If files show movie/TV episode patterns (season/episode tags, known film titles), documentaries (descriptive tags), tutorials, or purely spoken/educational content, classify as not music videos unless strong MV indicators appear.
4. Language detection:
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
