import pytest

from ..ai import model, setupLogfire
from ..categorizer.models import IsMovieResponse, PlanRequestWithCategory
from ..models import (
  Category,
  Language,
  MoverResponse,
  PlanAction,
  PlanRequest,
  SimpleAgentResponseResult,
)
from .movie_mover import move


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_movie_mover_agent():
  setupLogfire()

  req = PlanRequestWithCategory(
    request=PlanRequest(
      files=[
        "The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv",
        "The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass",
        "The.Mad.Phoenix.1997/cover.jpg",
        "The.Mad.Phoenix.1997/behind the scenes.mp4.part",
      ],
    ),
    category=Category.movie,
    movie=IsMovieResponse(
      is_movie=SimpleAgentResponseResult.yes,
      is_anim=SimpleAgentResponseResult.no,
      movie_name="The Mad Phoenix",
      movie_name_in_chinese="南海十三郎",
      release_year=1997,
      language=Language.chinese,
      reason="metadata from tmdb",
    ),
  )

  res, usage = await move(req)
  want = MoverResponse(
    plan=[
      PlanAction(
        file="The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv",
        action="move",
        target="movie/chinese/南海十三郎 (1997)/南海十三郎 (1997).mkv",
      ),
      PlanAction(
        file="The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass",
        action="move",
        target="movie/chinese/南海十三郎 (1997)/南海十三郎 (1997).English.eng.ass",
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

  assert res == want
  assert usage is not None
