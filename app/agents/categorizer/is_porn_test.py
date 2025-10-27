import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_porn import is_porn
from .models import GroupIsPornResponse, SimpleAgentResponseResult


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_porn_yes():
  """Test case: porn("Long Con 1") yes - regular porn content."""
  setupLogfire()

  req = PlanRequest(files=["Long Con 1.mp4"], metadata={})

  res, usage = await is_porn(req, metadataMcp())

  assert isinstance(res, GroupIsPornResponse)
  assert res.is_porn == SimpleAgentResponseResult.yes
  assert "Long Con 1.mp4" in res.porns
  assert res.porns["Long Con 1.mp4"].is_porn == SimpleAgentResponseResult.yes
  assert "porn" in res.porns["Long Con 1.mp4"].reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_porn_vr_yes():
  """Test case: porn vr("SLROriginals - Stepsister's Intimate Desires") yes - VR porn content."""
  setupLogfire()

  req = PlanRequest(files=["SLROriginals - Stepsister's Intimate Desires.mp4"], metadata={})

  res, usage = await is_porn(req, metadataMcp())

  assert isinstance(res, GroupIsPornResponse)
  assert res.is_porn == SimpleAgentResponseResult.yes
  assert "SLROriginals - Stepsister's Intimate Desires.mp4" in res.porns
  assert (
    res.porns["SLROriginals - Stepsister's Intimate Desires.mp4"].is_porn
    == SimpleAgentResponseResult.yes
  )
  assert (
    res.porns["SLROriginals - Stepsister's Intimate Desires.mp4"].is_vr
    == SimpleAgentResponseResult.yes
  )
  assert (
    "vr" in res.porns["SLROriginals - Stepsister's Intimate Desires.mp4"].reason.lower()
    or "virtual" in res.porns["SLROriginals - Stepsister's Intimate Desires.mp4"].reason.lower()
  )


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_porn_jav_no():
  """Test case: jav(IPZZ-123) no - JAV content (should be categorized as bango_porn not porn)."""
  setupLogfire()

  req = PlanRequest(files=["IPZZ-123.mp4"], metadata={})

  res, usage = await is_porn(req, metadataMcp())

  assert isinstance(res, GroupIsPornResponse)
  assert res.is_porn == SimpleAgentResponseResult.no
  assert "IPZZ-123.mp4" in res.porns
  assert res.porns["IPZZ-123.mp4"].is_porn == SimpleAgentResponseResult.no
  assert (
    "jav" in res.porns["IPZZ-123.mp4"].reason.lower()
    or "bango" in res.porns["IPZZ-123.mp4"].reason.lower()
  )
