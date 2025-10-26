from typing import Tuple

from .ai import metadataMcp
from .categorizer.runner import run_categorizer
from .models import PlanRequest, PlanResponse
from .mover.runner import run_mover
from pydantic_ai.usage import RunUsage


async def create_plan(req: PlanRequest) -> Tuple[PlanResponse, RunUsage]:
  mcp = metadataMcp()
  categorizer_res, categorizer_usage = await run_categorizer(req, mcp)
  mover_res, mover_usage = await run_mover(categorizer_res)

  total_usage = RunUsage()
  total_usage.incr(categorizer_usage)
  total_usage.incr(mover_usage)

  return mover_res, total_usage
