import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_music import agent
from .models import SimpleAgentResponseResult


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_music_yes():
  """Test case: music yes - files clearly represent music content."""
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

  assert res.output.is_music == SimpleAgentResponseResult.yes
  assert "music" in res.output.reason.lower()


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_music_audio_book_no():
  """Test case: audio_book no - files represent audiobook not music."""
  setupLogfire()

  req = PlanRequest(
    files=[
      "Audiobook - The Great Gatsby - Chapter 1.mp3",
      "Audiobook - The Great Gatsby - Chapter 2.mp3",
      "Audiobook - The Great Gatsby - Chapter 3.mp3",
    ],
    metadata={},
  )

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_music == SimpleAgentResponseResult.no
  assert "audiobook" in res.output.reason.lower() or "chapter" in res.output.reason.lower()
