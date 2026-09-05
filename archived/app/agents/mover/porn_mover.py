from os import path
from typing import Tuple

from pydantic_ai import RunUsage

from ..categorizer.models import PlanRequestWithCategory
from ..models import MoverResponse, PlanAction, SimpleAgentResponseResult, TargetDir
from .subtitle_mover import move as subtitle_move
from .utils import filter_video_files_sub_files_and_others


async def move(dir: str, req: PlanRequestWithCategory) -> Tuple[MoverResponse, RunUsage]:
  result = MoverResponse(plan=[])
  total_usage = RunUsage()

  for file, details in req.porn.porns.items():
    target_dir = TargetDir.porn.name
    if details.is_vr == SimpleAgentResponseResult.yes:
      target_dir = TargetDir.porn_vr.name

    original_file = path.splitext(file)
    ext = original_file[1]

    name = details.id
    if not name:
      name = details.name
    if not name:
      name = original_file[0]

    result.plan.append(
      PlanAction(file=file, action="move", target=path.join(target_dir, name, f"{name}{ext}"))
    )

  _, subfiles, others = filter_video_files_sub_files_and_others(req.request.files)

  # Handle subtitle files
  if subfiles:
    subtitle_response, subtitle_usage = await subtitle_move(dir, subfiles, result)
    result.plan.extend(subtitle_response.plan)
    total_usage.incr(subtitle_usage)

  # Handle other files (set to skip)
  for other_file in others:
    result.plan.append(PlanAction(file=other_file, action="skip"))

  return result, total_usage


if __name__ == "__main__":
  import asyncio

  from ..ai import model, setupLogfire
  from ..categorizer.models import (
    GroupIsPornResponse,
    IsPornResponse,
    PlanRequestWithCategory,
  )
  from ..models import Category, Language, PlanRequest

  if model():
    setupLogfire()

    req = PlanRequestWithCategory(
      request=PlanRequest(
        files=["Long-Con-1.mp4"],
      ),
      category=Category.porn,
      porn=GroupIsPornResponse(
        is_porn=SimpleAgentResponseResult.yes,
        porns={
          "Long-Con-1.mp4": IsPornResponse(
            id="vixen-long-con-part-1",
            name="Long Con Part 1",
            is_porn=SimpleAgentResponseResult.yes,
            is_vr=SimpleAgentResponseResult.no,
            from_onlyfans=SimpleAgentResponseResult.no,
            actors=[],
            language=Language.English,
            reason="",
          )
        },
      ),
    )

    res, usage = asyncio.run(move("", req))
    print(f"Output: {res}")
    print(f"Usage: {usage}")
