import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_movie import is_movie
from .models import SimpleAgentResponseResult


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_movie_yes():
  """Test case: movie yes - files clearly represent movie content."""
  setupLogfire()

  req = PlanRequest(files=["Inception.2010.1080p.BluRay.x264.mkv"], metadata={})

  res, _ = await is_movie(req, metadataMcp())

  assert res.is_movie == SimpleAgentResponseResult.yes
  assert "movie" in res.reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_movie_anim_movie_yes():
  """Test case: anim-movie yes - files clearly represent anime movie content."""
  setupLogfire()

  req = PlanRequest(files=["Your.Name.2016.1080p.BluRay.x264.mkv"], metadata={})

  res, _ = await is_movie(req, metadataMcp())

  assert res.is_movie == SimpleAgentResponseResult.yes
  assert res.is_anim == SimpleAgentResponseResult.yes
  assert "anime" in res.reason.lower() or "animation" in res.reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_movie_jav_no():
  """Test case: jav no - JAV content should not be categorized as movie."""
  setupLogfire()

  req = PlanRequest(files=["IPZZ-123.mp4"], metadata={})

  res, _ = await is_movie(req, metadataMcp())

  assert res.is_movie == SimpleAgentResponseResult.no


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_movie_tv_series_no():
  """Test case: tv series no - TV series content should not be categorized as movie."""
  setupLogfire()

  req = PlanRequest(files=["Game.of.Thrones.S01E01.mp4"], metadata={})

  res, _ = await is_movie(req, metadataMcp())

  assert res.is_movie == SimpleAgentResponseResult.no
  assert (
    "tv" in res.reason.lower() or "series" in res.reason.lower() or "episode" in res.reason.lower()
  )


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_movie_porn_no():
  """Test case: porn no - porn content should not be categorized as movie."""
  setupLogfire()

  req = PlanRequest(files=["Long Con 1.mp4"], metadata={})

  res, _ = await is_movie(req, metadataMcp())

  assert res.is_movie == SimpleAgentResponseResult.no
