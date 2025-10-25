import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_music_video import agent
from .models import SimpleAgentResponseResult


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_music_video_yes():
  """Test case: music_video yes - files clearly represent music video content."""
  setupLogfire()

  req = PlanRequest(
    files=["Taylor Swift - Shake It Off.mp4", "Taylor Swift - Bad Blood.mp4"], metadata={}
  )

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_music_video == SimpleAgentResponseResult.yes
  assert "music video" in res.output.reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_music_video_music_no():
  """Test case: music no - files represent music not music video."""
  setupLogfire()

  req = PlanRequest(
    files=[
      "The Beatles - Come Together.mp3",
      "The Beatles - Something.mp3",
      "The Beatles - Here Comes the Sun.mp3",
    ],
    metadata={},
  )

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_music_video == SimpleAgentResponseResult.no
  assert "music" in res.output.reason.lower() and (
    "audio" in res.output.reason.lower() or "mp3" in res.output.reason.lower()
  )
