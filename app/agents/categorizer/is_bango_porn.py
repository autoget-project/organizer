import asyncio
import os

from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, model
from ..models import PlanRequest, SimpleAgentResponseResult
from .models import GroupIsBangoPornResponse, IsBangoPornResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents Japanese bango porn content and detecting specific characteristics like VR, Madou productions, and FC2 content based solely on filenames, directory paths, and available metadata. Return an IsBangoPornResponse object with "is_bango_porn" (yes/no/maybe), "is_vr", "is_madou", "is_fc2", "bango", "actors", "language", and "reason".

Please repeat the prompt back as you understand it.

Classification Rules for is_bango_porn field:
- "yes": Files are clearly Japanese/Chinese bango porn content (have valid bango codes, studio prefixes, or confirmed porn metadata)
- "no": Files are clearly NOT bango porn content (mainstream movies, TV shows, music, regular videos, non-porn content, etc.)
- "maybe": Files could be bango porn but lack sufficient bango codes or verification

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.

2. Return "no" when:
   - Files are clearly mainstream movies or TV shows (Hollywood, Chinese, Korean dramas, etc.)
   - Files are music videos, concerts, or audio content
   - Files are regular documentaries, educational content, or non-porn media
   - Files have no bango codes and clearly indicate non-porn content in filenames/metadata
   - Files are software, games, or other non-video content
   - Web search confirms the content is not porn but mainstream media

3. Return "yes" when:
   - Valid bango codes found: `[A-Z]{2,5}-[0-9]{2,7}` (example: ABP-123, SSIS-456)
   - Known bango prefixes: JAV, FC2, HEYZO, CARIB, 1PON, etc.
   - Madou (麻豆) prefixes: MD|MDCM|MDHG|MDHT|MDL|MDSR|MSD
   - FC2 patterns: `FC2(-PPV)?-[0-9]{4,8}` (example: FC2-1234567, FC2-PPV-1234567)
   - Web search confirms the bango code is porn content
   - Metadata clearly indicates adult content with bango-like naming

4. Thinking order:
   - Scan filenames and directory paths for candidate bango codes, studio names, actor tokens, and VR/Madou/FC2 indicators.
   - Check metadata (title/description/tags/studio) for matching candidates — metadata takes precedence over filenames.
   - For each candidate bango code, call search_japanese_porn for metadata (studio, tags, actors, vr flag).
   - Use combined evidence to set is_bango_porn, is_vr, is_madou, is_fc2 and extract actor names. Search results override filename/metadata when present.
   - If web search shows the content is not porn, override to "no".

5. Special detection rules:
   - VR: common bango prefixes: IPVR, DSVR, HNVR, JUVR, MDVR, SIVR or keywords VR, 360°, Virtual Reality in metadata/tags.
   - Madou (麻豆): We consider Madou (麻豆) porn as bango porn because they also use bango system for naming.
   - Actor extraction: Extract Japanese/Chinese actress names from search_japanese_porn results

6. Verification:
   - Always use web search to verify bango codes when found
   - If search results indicate the content is mainstream media (not porn), return "no"
   - If search fails or is inconclusive but strong bango patterns exist, consider "maybe"
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
  from ..ai import setupLogfire, metadataMcp

  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "SSIS-456.mp4",
      ],
    )

    res = asyncio.run(is_bango_porn(req, metadataMcp()))
    print(f"output: {res}")
