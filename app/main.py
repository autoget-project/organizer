import os
import sys
from contextlib import asynccontextmanager
from typing import List
from fastapi import FastAPI

from .agents.models import Category, PlanRequest, PlanAction, PlanResponse, ExecuteRequest


def check_env_vars(name: str):
  var = os.getenv(name)
  if not var:
    print(f"Error: {name} environment variable is not set or is empty.", file=sys.stderr)
    sys.exit(1)


def check_any_env_vars(names: List[str]) -> bool:
  """Check if any of the given environment variables is set."""

  for name in names:
    var = os.getenv(name)
    if var:
      return True
  return False


def check_dir(path: str):
  if not os.path.exists(path):
    print(f"Error: {path} does not exist.", file=sys.stderr)
    sys.exit(1)


@asynccontextmanager
async def lifespan(app: FastAPI):
  # Startup
  check_any_env_vars(["XAI_API_KEY", "LM_STUDIO_API_BASE"])
  check_env_vars("MODEL")
  check_env_vars("SEARCH_MCP")
  check_env_vars("DOWNLOAD_COMPLETED_DIR")
  check_env_vars("TARGET_DIR")

  for cat in Category:
    check_dir(os.path.join(os.getenv("TARGET_DIR"), cat.name))

  yield
  # Shutdown


app = FastAPI(lifespan=lifespan)


@app.post("/v1/plan", response_model=PlanResponse)
async def create_plan(request: PlanRequest):
  # For now, return a fake response as the agent caller is not ready
  fake_plan = []
  for file_path in request.files:
    if "document" in file_path.lower():
      fake_plan.append(
        PlanAction(file=file_path, action="move", target=f"documents/{os.path.basename(file_path)}")
      )
    else:
      fake_plan.append(PlanAction(file=file_path, action="ignore"))
  return PlanResponse(plan=fake_plan)


@app.post("/v1/execute")
async def execute_plan(request: ExecuteRequest):
  # For now, just return a 200 OK status
  return {"message": "Plan executed successfully (fake response)"}
