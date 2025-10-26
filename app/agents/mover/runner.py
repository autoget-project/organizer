from ..models import Category, PlanResponse, simple_move_categories
from .movie_mover import move as movie_move
from .simple_mover import simple_move_plan
from .tv_series_mover import move as tv_series_move
from ..categorizer.models import PlanRequestWithCategory


async def run_mover(categorizer_res: PlanRequestWithCategory) -> PlanResponse:
    cat = categorizer_res.category

    if cat in simple_move_categories:
        return simple_move_plan(categorizer_res)

    if cat == Category.movie:
        res = await movie_move(categorizer_res)
        return PlanResponse(plan=res.plan)

    if cat == Category.tv_series:
        res = await tv_series_move(categorizer_res)
        return PlanResponse(plan=res.plan)
