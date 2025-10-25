from .categorizer.runner import run_categorizer
from .models import Category, PlanRequest, PlanResponse, simple_move_categories
from .mover.simple_mover import simple_move_plan
from .movie_mover import move as movie_move
from .tv_series_mover import move as tv_series_move


async def create_plan(req: PlanRequest) -> PlanResponse:
  categorizer_res = await run_categorizer(req)
  cat = categorizer_res.category
  if cat in simple_move_categories:
    return simple_move_plan(cat, req.files)

  if cat == Category.movie:
    res = await movie_move(categorizer_res)
    return PlanResponse(plan=res.plan)

  if cat == Category.tv_series:
    res = await tv_series_move(categorizer_res)
    return PlanResponse(plan=res.plan)
