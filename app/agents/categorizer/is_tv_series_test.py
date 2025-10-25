import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_tv_series import agent
from .models import SimpleAgentResponseResult


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_tv_series_yes():
  """Test case: tv yes - files clearly represent TV series content."""
  setupLogfire()

  req = PlanRequest(files=["Game.of.Thrones.S01E01.mp4", "Game.of.Thrones.S01E02.mp4"], metadata={})

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_tv_series == SimpleAgentResponseResult.yes
  assert "tv" in res.output.reason.lower() or "series" in res.output.reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_tv_series_anim_tv_yes():
  """Test case: anim-tv yes - files clearly represent anime TV series content."""
  setupLogfire()

  req = PlanRequest(files=["Attack.on.Titan.S01E01.mp4", "Attack.on.Titan.S01E02.mp4"], metadata={})

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_tv_series == SimpleAgentResponseResult.yes
  assert res.output.is_anim == SimpleAgentResponseResult.yes
  assert "anime" in res.output.reason.lower() or "animation" in res.output.reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_tv_series_jav_no():
  """Test case: jav no - JAV content should not be categorized as TV series."""
  setupLogfire()

  req = PlanRequest(files=["IPZZ-123.mp4"], metadata={})

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_tv_series == SimpleAgentResponseResult.no


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_tv_series_movie_no():
  """Test case: movie no - movie content should not be categorized as TV series."""
  setupLogfire()

  req = PlanRequest(files=["Inception.2010.1080p.BluRay.x264.mkv"], metadata={})

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_tv_series == SimpleAgentResponseResult.no
  assert "movie" in res.output.reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_tv_series_porn_no():
  """Test case: porn no - porn content should not be categorized as TV series."""
  setupLogfire()

  req = PlanRequest(files=["Long Con 1.mp4"], metadata={})

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_tv_series == SimpleAgentResponseResult.no
