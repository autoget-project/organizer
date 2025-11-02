from os import path
from typing import Tuple

from pydantic import BaseModel
from pydantic_ai import Agent, RunUsage, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import model
from ..categorizer.models import IsBangoPornResponse, PlanRequestWithCategory
from ..models import MoverResponse, PlanAction, SimpleAgentResponseResult, TargetDir
from .jav_actor import find_a_dir_for_list_of_actor_name, read_actor_alias
from .subtitle_mover import move as subtitle_move
from .utils import filter_video_files_sub_files_and_others

_INSTRUCTION = """\
You are a specialized file mover for bango porn videos. Your task is to create a movement plan for video files based on their bango (番号) identifiers.

## Core Rules:
1. **Default behavior**: Move each file to its target directory and rename using the bango.ext format
2. **Bango extraction**: Extract the bango (番号) from filenames - these are typically alphanumeric codes like "SSIS-698", "FC2-1234567", etc.
3. **Target directory**: Each file has a pre-determined target_dir where it should be moved

## Special Cases:

### Case 1: bango-C.ext format
- If the filename already follows the pattern "bango-C.ext" (where C is a hint the video has Chinese subtitles), keep the original filename
- Example: "SSIS-698-C.mp4" → keep as "SSIS-698-C.mp4"

### Case 2: Uppercase bango
- Always use uppercase for the bango portion when renaming
- Example: "ssis-698.mp4" → rename to "SSIS-698.mp4"

### Case 3: Multi-part files (bango-A.ext, bango-B.ext, etc.)
- If you have multiple files with the same bango but different letter suffixes (A, B, C, etc.)
- Rename them to: "bango.part.1.ext", "bango.part.2.ext", "bango.part.3.ext", etc.
- Maintain the original file extension
- Example:
  - "SSIS-698-A.mp4" → "SSIS-698.part.1.mp4"
  - "SSIS-698-B.mp4" → "SSIS-698.part.2.mp4"
  - "SSIS-698-C.mp4" → "SSIS-698.part.3.mp4" (unless it's the special Case 1)

## Output Format:
For each file, output a PlanAction with:
- file: the original file path
- action: "move"
- target: the full path where the file should be moved (target_dir + new_filename)

## Important Notes:
- Always preserve the original file extension
- When renaming multi-part files, use sequential numbering starting from 1
- If uncertain about the bango extraction, use your best judgment based on common JAV naming patterns
- Skip any files that don't appear to be video files or don't contain recognizable bango patterns
"""


def mover() -> Agent:
  return Agent(
    name="bango_porn_video_mover",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(MoverResponse),
  )


class BangoPorn(BaseModel):
  file: str
  target_dir: str
  detail: IsBangoPornResponse


class VideoMoverRequest(BaseModel):
  files: list[BangoPorn]


async def move(
  dir: str, req: PlanRequestWithCategory, mcp: MCPServer
) -> Tuple[MoverResponse, RunUsage]:
  """Move bango porn files to appropriate target directories."""
  aa = read_actor_alias()
  total_usage = RunUsage()

  video_request = VideoMoverRequest(files=[])
  for file, details in req.bango_porn.porns.items():
    target_dir = TargetDir.jav.name
    if details.is_vr == SimpleAgentResponseResult.yes:
      target_dir = TargetDir.jav_vr.name
    if details.from_madou == SimpleAgentResponseResult.yes:
      target_dir = TargetDir.madou.name

    if not details.actors:
      target_dir = path.join(target_dir, "素人")
    else:
      dir, usage = await find_a_dir_for_list_of_actor_name(aa, mcp, details.actors)
      total_usage.incr(usage)
      target_dir = path.join(target_dir, dir)

    bp = BangoPorn(file=file, target_dir=target_dir, detail=details)
    video_request.files.append(bp)

  a = mover()
  res = await a.run(video_request.model_dump_json())
  total_usage.incr(res.usage())

  _, subfiles, others = filter_video_files_sub_files_and_others(req.request.files)

  # Initialize plan with video movement results
  plan = res.output.plan if res.output.plan else []

  # Handle subtitle files
  if subfiles:
    subtitle_response, subtitle_usage = await subtitle_move(dir, subfiles, MoverResponse(plan=plan))
    plan.extend(subtitle_response.plan)
    total_usage.incr(subtitle_usage)

  # Handle other files (set to skip)
  for other_file in others:
    plan.append(PlanAction(file=other_file, action="skip"))

  return MoverResponse(plan=plan), total_usage


if __name__ == "__main__":
  import asyncio
  import json
  import os
  from os import path

  from ..ai import metadataMcp, setupLogfire
  from ..categorizer.models import GroupIsBangoPornResponse
  from ..models import Category, Language, PlanRequest

  if model():
    setupLogfire()

    test_data = {
      "Yui Hatano": ["Yui Hatano", "波多野结衣", "波多野結衣"],
    }

    test_file = os.getenv("JAV_ACTOR_FILE")
    with open(test_file, "w", encoding="utf-8") as f:
      json.dump(test_data, f)

    req = PlanRequestWithCategory(
      request=PlanRequest(
        files=[
          "/downloads/SSIS-698-C.mp4",
        ],
      ),
      category=Category.bango_porn,
      bango_porn=GroupIsBangoPornResponse(
        is_bango_porn=SimpleAgentResponseResult.yes,
        porns={
          "/downloads/SSIS-698-C.mp4": IsBangoPornResponse(
            bango="SSIS-698",
            is_bango_porn=SimpleAgentResponseResult.yes,
            is_vr=SimpleAgentResponseResult.no,
            from_madou=SimpleAgentResponseResult.no,
            from_fc2=SimpleAgentResponseResult.no,
            actors=["Yua Mikami"],  # actor doesnot exist in file, will add
            language=Language.Japanese,
            reason="",
          ),
        },
      ),
    )

    mcp = metadataMcp()
    res, usage = asyncio.run(move("", req, mcp))
    print(f"Output: {res}")
    print(f"Usage: {usage}")
