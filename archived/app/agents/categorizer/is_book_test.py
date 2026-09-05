import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_book import agent
from .models import SimpleAgentResponseResult


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_book_yes():
  """Test case: book yes - files clearly represent book content."""
  setupLogfire()

  req = PlanRequest(
    files=["Stephen King - The Shining.pdf", "Stephen King - The Shining.epub"], metadata={}
  )

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_book == SimpleAgentResponseResult.yes
  assert "book" in res.output.reason.lower()


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_is_book_audio_book_no():
  """Test case: audio_book no - files represent audiobook not book."""
  setupLogfire()

  req = PlanRequest(
    files=[
      "Stephen King - The Shining - Chapter 1.mp3",
      "Stephen King - The Shining - Chapter 2.mp3",
      "Stephen King - The Shining - Chapter 3.mp3",
    ],
    metadata={},
  )

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_book == SimpleAgentResponseResult.no
  assert "audio" in res.output.reason.lower() or "mp3" in res.output.reason.lower()
