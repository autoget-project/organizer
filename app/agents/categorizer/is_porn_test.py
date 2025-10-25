import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_porn import agent
from .models import SimpleAgentResponseResult


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_porn_yes():
  """Test case: porn("Long Con 1") yes - regular porn content."""
  setupLogfire()

  req = PlanRequest(files=["Long Con 1.mp4"], metadata={})

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_porn == SimpleAgentResponseResult.yes
  assert "porn" in res.output.reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_porn_vr_yes():
  """Test case: porn vr("SLROriginals - Stepsister's Intimate Desires") yes - VR porn content."""
  setupLogfire()

  req = PlanRequest(files=["SLROriginals - Stepsister's Intimate Desires.mp4"], metadata={})

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_porn == SimpleAgentResponseResult.yes
  assert res.output.is_vr == SimpleAgentResponseResult.yes
  assert "vr" in res.output.reason.lower() or "virtual" in res.output.reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_porn_jav_no():
  """Test case: jav(IPZZ-123) no - JAV content (should be categorized as bango_porn not porn)."""
  setupLogfire()

  req = PlanRequest(files=["IPZZ-123.mp4"], metadata={})

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_porn == SimpleAgentResponseResult.no
  assert "jav" in res.output.reason.lower() or "bango" in res.output.reason.lower()
