import os

from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest, SimpleAgentResponseResult
from .models import GroupIsBangoPornResponse, IsBangoPornResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents Japanese bango porn content and detecting specific characteristics like VR, Madou productions, and FC2 content based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.

2. Thinking order:
   - Scan filenames and directory paths for candidate bango codes, studio names, actor tokens, and VR/Madou/FC2 indicators.
   - Check metadata (title/description/tags/studio) for matching candidates — metadata takes precedence over filenames.
   - For each candidate bango code , call search_japanese_porn for metadata (studio, tags, actors, vr flag).
   - Use combined evidence to set is_bango_porn, is_vr, is_madou, is_fc2 and extract actor names. Search results override filename/metadata when present.
3. Detection rules / regex examples:
   - Bango general: `[A-Z]{2,5}-[0-9]{2,7}`  (example: ABP-123, SSIS-456)
   - Standard prefixes to look for in filenames/metadata: JAV, FC2, HEYZO, CARIB, 1PON, etc.
   - VR: common bango prefixes: IPVR, DSVR, HNVR, JUVR, MDVR, SIVR or keywords VR, 360°, Virtual Reality in metadata/tags.
   - Madou (麻豆): bango prefixes MD|MDCM|MDHG|MDHT|MDL|MDSR|MSD (- optional dash or digits follow)
   - FC2: `FC2(-PPV)?-[0-9]{4,8}`  (example: FC2-1234567, FC2-PPV-1234567)

4. Actor extraction:
   - Extract Japanese actress names from search_japanese_porn results
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
