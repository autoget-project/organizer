import pytest

from ..ai import metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .is_photobook import agent
from .models import SimpleAgentResponseResult


@pytest.mark.asyncio
@pytest.mark.skipif(model() is None, reason="No env var for ai model")
async def test_is_photobook_yes():
  """Test case: photobook yes - files clearly represent photobook content."""
  setupLogfire()

  req = PlanRequest(
    files=["Photobook/001.jpg", "Photobook/002.jpg", "Photobook/003.jpg"], metadata={}
  )

  test_agent = agent(metadataMcp())
  res = await test_agent.run(req.model_dump_json())

  assert res.output.is_photobook == SimpleAgentResponseResult.yes
  assert "photobook" in res.output.reason.lower() or "photo" in res.output.reason.lower()
