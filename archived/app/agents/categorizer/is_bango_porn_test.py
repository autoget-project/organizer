import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_bango_porn import is_bango_porn
from .models import SimpleAgentResponseResult


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_bango_porn_jav_yes():
  """Test case: jav(IPZZ-123) yes - file with JAV bango code."""
  setupLogfire()

  req = PlanRequest(files=["IPZZ-123.mp4"], metadata={})

  result, usage = await is_bango_porn(req, metadataMcp())
  assert result.is_bango_porn == SimpleAgentResponseResult.yes
  assert "IPZZ-123.mp4" in result.porns
  assert result.porns["IPZZ-123.mp4"].is_bango_porn == SimpleAgentResponseResult.yes
  assert usage is not None


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_bango_porn_fc2_yes():
  """Test case: fc2(FC2-PPV-4784877) yes - file with FC2 PPV code."""
  setupLogfire()

  req = PlanRequest(files=["FC2-PPV-4784877.mp4"], metadata={})

  result, usage = await is_bango_porn(req, metadataMcp())
  assert result.is_bango_porn == SimpleAgentResponseResult.yes
  assert "FC2-PPV-4784877.mp4" in result.porns
  assert result.porns["FC2-PPV-4784877.mp4"].from_fc2 == SimpleAgentResponseResult.yes
  assert usage is not None


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_bango_porn_madou_yes():
  """Test case: madou(MDCM-0005) yes - file with Madou bango code."""
  setupLogfire()

  req = PlanRequest(files=["MDCM-0005.mp4"], metadata={})

  result, usage = await is_bango_porn(req, metadataMcp())
  assert result.is_bango_porn == SimpleAgentResponseResult.yes
  assert "MDCM-0005.mp4" in result.porns
  assert result.porns["MDCM-0005.mp4"].from_madou == SimpleAgentResponseResult.yes
  assert usage is not None


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_bango_porn_jav_vr_yes():
  """Test case: jav vr(IPVR-002) yes - file with JAV VR bango code."""
  setupLogfire()

  req = PlanRequest(files=["IPVR-002.mp4"], metadata={})

  result, usage = await is_bango_porn(req, metadataMcp())
  assert result.is_bango_porn == SimpleAgentResponseResult.yes
  assert "IPVR-002.mp4" in result.porns
  assert result.porns["IPVR-002.mp4"].is_vr == SimpleAgentResponseResult.yes
  assert usage is not None


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_bango_porn_porn_no():
  """Test case: porn(Long Con) no - regular porn without bango code."""
  setupLogfire()

  req = PlanRequest(files=["Long Con 1.mp4"], metadata={})

  result, usage = await is_bango_porn(req, metadataMcp())
  assert result.is_bango_porn == SimpleAgentResponseResult.no
  assert "Long Con 1.mp4" in result.porns
  assert result.porns["Long Con 1.mp4"].is_bango_porn == SimpleAgentResponseResult.no
  assert usage is not None
