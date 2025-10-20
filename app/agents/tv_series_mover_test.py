import pytest
from .models import PlanRequest, PlanResponse, PlanAction, Category
from .categorizer import CategoryResponse
from .tv_series_mover import agent
from .ai import metadataMcp, model, setupLogfire


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_tv_series_mover_agent():
  setupLogfire()

  req = PlanRequest(
    files=[
      "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv",
      "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP2.mkv",
      "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass",
      "My.Date.with.a.Vampire.Season.02.2000/cover.jpg",
      "My.Date.with.a.Vampire.Season.02.2000/behind the scenes.mp4.part",
    ],
  )

  a = agent(metadataMcp(), CategoryResponse(category=Category.tv_series, language="Chinese"))
  res = await a.run(req.model_dump_json())
  want = PlanResponse(
    plan=[
      PlanAction(
        file="My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv",
        action="move",
        target="tv_series/Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E01.mkv",
      ),
      PlanAction(
        file="My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP2.mkv",
        action="move",
        target="tv_series/Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E02.mkv",
      ),
      PlanAction(
        file="My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass",
        action="move",
        target="tv_series/Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E01.English.eng.ass",
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
  assert res.output == want
