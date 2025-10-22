import pytest

from .ai import model, setupLogfire
from .models import PlanRequest
from .porn_is_vr import VRResponse, agent


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_vr_agent_with_vr_content():
  setupLogfire()

  req = PlanRequest(
    files=[
      "VRPorn/ExampleScene.180.sbs.mp4",
      "VRPorn/ExampleScene.360.mp4",
      "VRPorn/VirtualRealPorn.180.mp4",
    ],
  )

  a = agent()
  res = await a.run(req.model_dump_json())
  want = VRResponse(is_vr=True)
  assert res.output == want


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_vr_agent_with_regular_content():
  setupLogfire()

  req = PlanRequest(
    files=[
      "Regular/ExampleScene.mp4",
      "Regular/AnotherScene.mkv",
      "Regular/StandardContent.avi",
    ],
  )

  a = agent()
  res = await a.run(req.model_dump_json())
  want = VRResponse(is_vr=False)
  assert res.output == want


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_vr_agent_mixed_content():
  setupLogfire()

  req = PlanRequest(
    files=[
      "VRPorn/ExampleScene.180.sbs.mp4",
      "Regular/ExampleScene.mp4",
      "VRBangers/TestScene.360.mp4",
    ],
  )

  a = agent()
  res = await a.run(req.model_dump_json())
  want = VRResponse(is_vr=True)
  assert res.output == want


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_vr_agent_no_vr_indicators():
  setupLogfire()

  req = PlanRequest(
    files=[
      "content/scene1.mp4",
      "content/scene2.mkv",
      "content/scene3.avi",
    ],
  )

  a = agent()
  res = await a.run(req.model_dump_json())
  want = VRResponse(is_vr=False)
  assert res.output == want
