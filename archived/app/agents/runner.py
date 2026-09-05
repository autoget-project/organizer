from typing import Tuple

from pydantic_ai.usage import RunUsage

from .ai import metadataMcp
from .categorizer.runner import run_categorizer
from .models import PlanRequest, PlanResponse
from .mover.runner import run_mover


async def create_plan(dir: str, req: PlanRequest) -> Tuple[PlanResponse, RunUsage]:
  mcp = metadataMcp()
  categorizer_res, categorizer_usage = await run_categorizer(req, mcp)
  mover_res, mover_usage = await run_mover(dir, categorizer_res, mcp)

  total_usage = RunUsage()
  total_usage.incr(categorizer_usage)
  total_usage.incr(mover_usage)

  return mover_res, total_usage
