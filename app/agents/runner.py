from .ai import metadataMcp
from .categorizer.runner import run_categorizer
from .models import  PlanRequest, PlanResponse
from .mover.runner import run_mover



async def create_plan(req: PlanRequest) -> PlanResponse:
    mcp = metadataMcp()
    categorizer_res = await run_categorizer(req, mcp)
    return await run_mover(categorizer_res)
