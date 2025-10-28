import pytest

from ..ai import model, setupLogfire
from ..categorizer.models import IsTVSeriesResponse, PlanRequestWithCategory
from ..models import (
  Category,
  Language,
  MoverResponse,
  PlanAction,
  PlanRequest,
  SimpleAgentResponseResult,
)
from .tv_series_mover import move


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_tv_series_mover_agent():
  setupLogfire()

  req = PlanRequestWithCategory(
    request=PlanRequest(
      files=[
        "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv",
        "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP2.mkv",
        "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass",
        "My.Date.with.a.Vampire.Season.02.2000/cover.jpg",
        "My.Date.with.a.Vampire.Season.02.2000/behind the scenes.mp4.part",
      ],
    ),
    category=Category.tv_series,
    tv_series=IsTVSeriesResponse(
      is_tv_series=SimpleAgentResponseResult.yes,
      is_anim=SimpleAgentResponseResult.no,
      tv_series_name="My Date with a Vampire",
      tv_series_name_in_chinese="我和僵尸有个约会",
      the_first_season_release_year=1998,
      language=Language.Chinese,
      reason="metadata from tmdb",
    ),
  )

  res, usage = await move(req)
  want = MoverResponse(
    plan=[
      PlanAction(
        file="My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv",
        action="move",
        target="tv_series/chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E01.mkv",
      ),
      PlanAction(
        file="My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP2.mkv",
        action="move",
        target="tv_series/chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E02.mkv",
      ),
      PlanAction(
        file="My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass",
        action="move",
        target="tv_series/chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E01.English.eng.ass",
      ),
      PlanAction(
        file="My.Date.with.a.Vampire.Season.02.2000/cover.jpg", action="skip", target=None
      ),
      PlanAction(
        file="My.Date.with.a.Vampire.Season.02.2000/behind the scenes.mp4.part",
        action="skip",
        target=None,
      ),
    ]
  )
  assert res == want
  assert usage is not None
