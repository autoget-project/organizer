import os

from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest, SimpleAgentResponseResult
from .models import GroupIsBangoPornResponse, IsBangoPornResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents Japanese bango porn content and detecting specific characteristics like VR, Madou productions, and FC2 content based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

Specifics (each bullet contains specifics about the task):

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent bango porn.

2. Bango porn detection criteria:
   - File types: Look for .mp4, .mkv, .avi, .mov, .wmv, .flv, .webm video files
   - Filename patterns:
     - Japanese bango format (alphanumeric codes like "ABP-123", "SSIS-456")
     - Standard bango prefixes: JAV, FC2, HEYZO, CARIB, 1PON, etc.
     - Japanese actress names in filenames
     - Japanese studio names (S1, MOODYZ, IdeaPocket, etc.)
   - Directory structure:
     - JAV folders with actress names or bango codes
     - Studio-based organization
     - Date-based organization (YYYY-MM-DD format)
   - Metadata indicators:
     - Japanese adult content tags
     - Actress information
     - Studio/producer information
     - Adult content ratings

3. VR detection criteria:
   - VR bango prefixes in filenames: "IPVR", "DSVR", "HNVR", "JUVR", "MDVR", "SIVR"
   - VR indicators in metadata or tags
   - VR-related keywords in filenames: "VR", "Virtual Reality", "360°"
   - VR studio names in metadata
   - Use search_japanese_porn to verify VR content from response tags/metadata

4. Madou detection criteria:
   - Madou bango prefixes: "MD", "MDCM", "MDHG", "MDHT", "MDL", "MDSR", "MSD"
   - Madou studio indicators in metadata
   - Chinese/Japanese mixed content typical of Madou productions
   - Madou-specific naming patterns
   - Use search_japanese_porn to verify Madou productions

5. FC2 detection criteria:
   - FC2 bango format: "FC2-PPV-1234567" or "FC2-1234567"
   - FC2 indicators in filenames
   - Amateur content indicators
   - FC2-specific naming patterns
   - Can be identified directly from filename patterns

6. Actor extraction:
   - Extract Japanese actress names from filenames and metadata
   - Use search_japanese_porn to get accurate actor information
   - Handle multiple actresses in the same content
   - Format names consistently (Japanese format preferred)

7. Language detection criteria:
   - Japanese: Hiragana/Katakana/Kanji characters in filenames or metadata
   - Chinese: Chinese characters (简体/繁體) in filenames or metadata
   - English: Latin script with English words; absence of East Asian scripts
   - Other: if none apply clearly

8. Analysis approach:
   - Analyze the entire file set as one logical unit
   - Prefer metadata over filename cues
   - Consider directory structure and naming patterns
   - Use search_japanese_porn to verify uncertain cases and get accurate metadata
   - Extract bango codes when identified
   - Determine VR, Madou, and FC2 status based on prefixes and search results
   - Extract actor names from search results when available
   - Provide brief reasoning for your decision
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="is_bango_porn_detector",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(IsBangoPornResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["search_japanese_porn"]),
  )


_VIDEO_EXT = {".mp4", ".mkv", ".avi", ".mov", ".wmv", ".ts"}


async def is_bango_porn(req: PlanRequest, mcp: MCPServer) -> GroupIsBangoPornResponse:
  res = GroupIsBangoPornResponse(is_bango_porn=SimpleAgentResponseResult.no, porns={})
  a = agent(mcp)
  found_yes = False
  found_maybe = False
  for file in req.files:
    _, ext = os.path.splitext(file.lower())
    if ext in _VIDEO_EXT:
      new_req = PlanRequest(files=[file], metadata=req.metadata)
      per_file_res = await a.run(new_req.model_dump_json())
      res.porns[file] = per_file_res.output
      if per_file_res.output.is_bango_porn == SimpleAgentResponseResult.yes:
        found_yes = True
      if per_file_res.output.is_bango_porn == SimpleAgentResponseResult.maybe:
        found_maybe = True
  if found_yes:
    res.is_bango_porn = SimpleAgentResponseResult.yes
  elif found_maybe:
    res.is_bango_porn = SimpleAgentResponseResult.maybe
  return res


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "SSIS-456.mp4",
      ],
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
