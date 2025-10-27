import json
import os

import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..categorizer.models import (
  GroupIsBangoPornResponse,
  IsBangoPornResponse,
  PlanRequestWithCategory,
)
from ..models import (
  Category,
  Language,
  MoverResponse,
  PlanAction,
  PlanRequest,
  SimpleAgentResponseResult,
)
from .bango_porn_mover import move


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_bango_porn_mover(tmp_path):
  setupLogfire()

  # Create test data
  test_data = {
    "Yui Hatano": ["Yui Hatano", "波多野结衣", "波多野結衣"],
  }

  # Write test data to temp file
  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump(test_data, f)

  # Set environment variable
  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  req = PlanRequestWithCategory(
    request=PlanRequest(
      files=[
        "/downloads/SSIS-698-C.mp4",
      ],
    ),
    category=Category.bango_porn,
    bango_porn=GroupIsBangoPornResponse(
      is_bango_porn=SimpleAgentResponseResult.yes,
      porns={
        "/downloads/SSIS-698-C.mp4": IsBangoPornResponse(
          bango="SSIS-698",
          is_bango_porn=SimpleAgentResponseResult.yes,
          is_vr=SimpleAgentResponseResult.no,
          from_madou=SimpleAgentResponseResult.no,
          from_fc2=SimpleAgentResponseResult.no,
          actors=["Yua Mikami"],  # actor doesnot exist in file, will add
          language=Language.japanese,
          reason="",
        ),
      },
    ),
  )

  mcp = metadataMcp()
  res, usage = await move(req, mcp)

  want = MoverResponse(
    plan=[
      PlanAction(
        file="/downloads/SSIS-698-C.mp4", action="move", target="jav/Yua Mikami/SSIS-698-C.mp4"
      )
    ]
  )

  assert res == want
  assert usage is not None
