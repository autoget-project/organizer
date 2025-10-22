from pydantic_ai.mcp import MCPServer
from utils.porn import filter_unrelated_files

from .models import PlanRequest, PlanResponse


async def create_plan(req: PlanRequest, mcp: MCPServer) -> PlanResponse:
  new_req, resp = filter_unrelated_files(req)
  return resp
