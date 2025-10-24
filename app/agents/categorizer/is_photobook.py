from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsPhotobookResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents photobooks (including 写真集) based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

Specifics (each bullet contains specifics about the task):

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent a photobook.

2. Photobook detection criteria:
   - File types: Look for .jpg, .jpeg, .png, .gif, .bmp, .tiff, .webp image files
   - May include video files as bonus/gift content: .mp4, .mkv, .avi, .mov files
   - Filename patterns: 
     - Japanese photobook terms: "写真集", "shashinshuu", "グラビア" (gravure)
     - Photobook keywords: "photobook", "photo book", "photoset", "gallery", "collection"
     - Model/idol names followed by photobook indicators
     - Page numbers in filenames (e.g., "001.jpg", "page_01.jpg")
     - Sequential numbering indicating scanned pages
   - Directory structure:
     - Folder names containing "写真集", "photobook", "photoset", "gallery", "collection"
     - Model or idol names as folder names
     - Publisher or magazine names (e.g., "Weekly Playboy", "Young Jump")
     - Date-based organization (e.g., "2023", "2023-06")
   - Metadata indicators:
     - "photobook", "model", "idol", "gravure", "写真集" in metadata fields
     - Photographer or publisher information
     - Publication date or volume information

3. Special considerations for photobooks with videos:
   - Video files may be included as bonus content or making-of videos
   - Videos typically have short duration and are supplementary to the main image content
   - Video filenames may include "making", "behind", "bonus", "gift", "特典"
   - The presence of videos with a dominant set of images still indicates a photobook

4. Non-photobook exclusions:
   - Regular photo albums or family photos (no professional/model indicators)
   - Art galleries or digital art collections (unless specifically photobook-style)
   - Magazine scans (unless specifically photobook issues)
   - Porn content (should be categorized as porn or bango_porn)
   - Movie or TV show screenshots
   - Stock photo collections

5. Analysis approach:
   - Analyze the entire file set as one logical unit
   - Prefer metadata over filename cues
   - Consider directory structure and dominant patterns
   - Look for sequential page numbering typical of scanned photobooks
   - Allow for supplementary video content alongside main image content
   - Use web_search to verify uncertain cases - search for model names, photobook titles, or publishers
   - When uncertain, select the closest matching response based on strongest evidence
   - Provide brief reasoning for your decision
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
