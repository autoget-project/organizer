from utils.utils import simple_move_plan

from .ai import metadataMcp
from .categorizer import agent as categorizer_agent
from .models import Category, PlanRequest, PlanResponse, simple_move_categories
from .movie_mover import agent as movie_mover_agent
from .tv_series_mover import agent as tv_series_mover_agent


async def create_plan(req: PlanRequest) -> PlanResponse:
  mcp = metadataMcp()

  categorizer = categorizer_agent(mcp)
  categorizer_res = await categorizer.run(req.model_dump_json())
  cat = categorizer_res.output.category
  if cat in simple_move_categories:
    return simple_move_plan(cat, req.files)

  if cat == Category.movie or cat == Category.anim_movie:
    movie_mover = movie_mover_agent(mcp, categorizer_res.output)
    return await movie_mover.run(req)

  if cat == Category.tv_series or cat == Category.anim_tv_series:
    tv_series_mover = tv_series_mover_agent(mcp, categorizer_res.output)
    return await tv_series_mover.run(req)
