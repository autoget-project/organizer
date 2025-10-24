import pytest

from .ai import metadataMcp, model, setupLogfire
from .models import Category, PlanAction, PlanRequest, PlanResponse
from .movie_mover import agent
from .oldcategorizer import CategoryResponse


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_movie_mover_agent():
  setupLogfire()

  req = PlanRequest(
    files=[
      "The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv",
      "The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass",
      "The.Mad.Phoenix.1997/cover.jpg",
      "The.Mad.Phoenix.1997/behind the scenes.mp4.part",
    ],
  )

  a = agent(metadataMcp(), CategoryResponse(category=Category.movie, language="Chinese"))
  res = await a.run(req.model_dump_json())
  want = PlanResponse(
    plan=[
      PlanAction(
        file="The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv",
        action="move",
        target="movie/Chinese/南海十三郎 (1997)/南海十三郎 (1997).mkv",
      ),
      PlanAction(
        file="The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass",
        action="move",
        target="movie/Chinese/南海十三郎 (1997)/南海十三郎 (1997).English.eng.ass",
      ),
      PlanAction(
        file="The.Mad.Phoenix.1997/cover.jpg",
        action="skip",
      ),
      PlanAction(
        file="The.Mad.Phoenix.1997/behind the scenes.mp4.part",
        action="skip",
      ),
    ]
  )

  assert res.output == want
