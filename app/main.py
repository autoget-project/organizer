import os
import sys
from contextlib import asynccontextmanager
from typing import List

import logfire
from fastapi import FastAPI
from fastapi.responses import JSONResponse

from .agents.models import (
  APIExecuteRequest,
  APIPlanRequest,
  ExecuteResponse,
  PlanFailed,
  PlanRequest,
  PlanResponse,
  TargetDir,
)
from .agents.runner import create_plan as ai_create_plan


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


def startup_check():
  check_any_env_vars(["XAI_API_KEY", "LM_STUDIO_API_BASE"])
  check_env_vars("MODEL")
  check_env_vars("METADATA_MCP")
  check_env_vars("DOWNLOAD_COMPLETED_DIR")
  check_env_vars("TARGET_DIR")
  check_env_vars("JAV_ACTOR_FILE")

  for t in TargetDir:
    check_dir(os.path.join(os.getenv("TARGET_DIR"), t.name))


@asynccontextmanager
async def lifespan(app: FastAPI):
  # Startup
  startup_check()

  yield
  # Shutdown


app = FastAPI(lifespan=lifespan)
if os.getenv("LOGFIRE_TOKEN"):
  logfire.configure()
  logfire.instrument_fastapi(app)


@app.post("/v1/plan", response_model=PlanResponse)
async def create_plan(request: APIPlanRequest):
  # Create PlanRequest from APIPlanRequest
  plan_request = PlanRequest(files=request.files, metadata=request.metadata)
  plan_response, _ = await ai_create_plan(request.dir, plan_request)
  return plan_response


@app.post("/v1/execute", response_model=ExecuteResponse)
async def execute_plan(request: APIExecuteRequest):
  resp = ExecuteResponse(failed_move=[])

  for action in request.plan:
    if action.action == "move":
      # check original file exists using completed_dir/request.dir/file
      original_file = os.path.join(os.getenv("DOWNLOAD_COMPLETED_DIR"), request.dir, action.file)
      if not os.path.exists(original_file):
        resp.failed_move.append(
          PlanFailed(
            file=action.file, action=action.action, target=action.target, reason="file not found"
          )
        )
        continue
      # check target folder exists
      target_file = os.path.join(os.getenv("TARGET_DIR"), action.target)
      os.makedirs(os.path.dirname(target_file), exist_ok=True)
      # move file
      os.rename(original_file, target_file)

  if resp.failed_move:
    return JSONResponse(content=resp.model_dump(), status_code=400)
  else:
    # Create archive directory if it doesn't exist
    archive_dir = os.path.join(os.getenv("DOWNLOAD_COMPLETED_DIR"), "archive")
    os.makedirs(archive_dir, exist_ok=True)

    # Check if source directory still exists before moving
    source_dir = os.path.join(os.getenv("DOWNLOAD_COMPLETED_DIR"), request.dir)
    if os.path.exists(source_dir):
      os.rename(
        source_dir,
        os.path.join(archive_dir, request.dir),
      )
    return resp
