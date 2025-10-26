import pytest

from .ai import model, setupLogfire
from .models import PlanAction, PlanRequest, PlanResponse
from .runner import create_plan


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
@pytest.mark.parametrize(
  "file,target",
  [
    ("Why Nations Fail.epub", "book/Why Nations Fail.epub"),
    (
      "Curse.of.the.Golden.Flower.2006.4K.mp4",
      "movie/chinese/满城尽带黄金甲 (2006)/满城尽带黄金甲 (2006).mp4",
    ),
    (
      "纵横四海.1999.Ep03.mp4",
      "tv_series/chinese/纵横四海 (1999)/Season 01/纵横四海 (1999) S01E03.mp4",
    ),
  ],
  ids=["simple", "movie", "tv"],
)
async def test_create_plan_simple(file: str, target: str):
  setupLogfire()

  got, usage = await create_plan(PlanRequest(files=[file]))
  want = PlanResponse(
    plan=[
      PlanAction(file=file, action="move", target=target),
    ]
  )

  assert got.plan == want.plan
  assert usage is not None
