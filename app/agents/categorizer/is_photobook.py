from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsPhotobookResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents photobooks (including 写真集) based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent a photobook.

2. Photobook detection rules & priorities (highest→lowest):
   - Metadata indicators (highest priority): any metadata field containing keywords: photobook, photoset, model, idol, 写真集, gravure, photographer, publisher, volume, issue, etc.
   - Filename & folder indicators: dominant presence of image file types (.jpg, .jpeg, .png, .webp, .tiff, .bmp, .gif). Look for keywords in filenames or folder names: 写真集, shashinshuu, グラビア, photobook, "photo book", photoset, gallery, collection, publisher/magazine names, model/celebrity names.
   - Sequential/page patterns: detect file pattern like `page[_\\-]?[0-9]+`, or repeated numeric prefixes (e.g., 001.jpg, 002.jpg). Sequential numbering strongly suggests scanned photobook pages.
   - Video allowance: video files (.mp4, .mkv, .avi, .mov) are permitted as supplementary content if they are a minority and filenames include making/behind/bonus/特典; their presence does not disqualify a photobook.
   - Exclusions (negative indicators): tags or filenames indicating screenshot, stock, generic art (unless explicitly photobook-style), or explicit porn (label as "porn" or "bango_porn" instead).
   - maybe use `web_search` with the title if uncertain.
   - Priority rule: prefer metadata evidence > filename patterns > folder structure > file counts.
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="is_photobook_detector",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(IsPhotobookResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["web_search"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "Yuki Aoi/写真集 2023/001.jpg",
        "Yuki Aoi/写真集 2023/002.jpg",
        "Yuki Aoi/写真集 2023/003.jpg",
        "Yuki Aoi/写真集 2023/making_video.mp4",
      ],
      metadata={
        "title": "Yuki Aoi 写真集 2023",
        "model": "Yuki Aoi",
        "description": "Photobook collection with making video",
      },
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
