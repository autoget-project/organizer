import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_audio_book import agent
from .models import SimpleAgentResponseResult


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_audio_book_yes():
  """Test case: audio_book yes - files clearly represent audiobook content."""
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

  assert res.output.is_audio_book == SimpleAgentResponseResult.yes
  assert "chapter" in res.output.reason.lower() or "audiobook" in res.output.reason.lower()


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_audio_book_music_no():
  """Test case: music no - files represent music not audiobook."""
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

  assert res.output.is_audio_book == SimpleAgentResponseResult.no
  assert "music" in res.output.reason.lower() or "album" in res.output.reason.lower()


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_audio_book_book_no():
  """Test case: book no - files represent book not audiobook."""
  setupLogfire()

  req = PlanRequest(
    files=[
      "The Great Gatsby.pdf",
      "The Great Gatsby - Chapter 1.pdf",
      "The Great Gatsby - Chapter 2.pdf",
    ],
    metadata={},
  )

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_audio_book == SimpleAgentResponseResult.no
  assert (
    "pdf" in res.output.reason.lower()
    or "ebook" in res.output.reason.lower()
    or "text" in res.output.reason.lower()
  )
