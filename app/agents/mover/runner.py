from typing import Tuple

from ..categorizer.models import PlanRequestWithCategory
from ..models import Category, PlanResponse, simple_move_categories
from pydantic_ai.usage import RunUsage
from .movie_mover import move as movie_move
from .simple_mover import simple_move_plan
from .tv_series_mover import move as tv_series_move


async def run_mover(categorizer_res: PlanRequestWithCategory) -> Tuple[PlanResponse, RunUsage]:
  cat = categorizer_res.category
  usage = RunUsage()

  if cat in simple_move_categories:
    return simple_move_plan(categorizer_res), usage

  if cat == Category.movie:
    res, move_usage = await movie_move(categorizer_res)
    usage.incr(move_usage)
    return PlanResponse(plan=res.plan), usage

  if cat == Category.tv_series:
    res, move_usage = await tv_series_move(categorizer_res)
    usage.incr(move_usage)
    return PlanResponse(plan=res.plan), usage

  return PlanResponse(), usage
