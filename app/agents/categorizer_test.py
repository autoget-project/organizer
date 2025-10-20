import pytest
from .models import Category, PlanRequest
from .categorizer import agent
from .ai import metadataMcp, model, setupLogfire


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.parametrize(
  "files,category",
  [
    (["The.Lychee.Road.2025.Ep01.mkv"], Category.tv_series),
    (["The.Lychee.Road.2025.mkv"], Category.movie),
    (["The.Lychee.Road.pdf"], Category.book),
    (["The.Lychee.Road.Podcast.mp3"], Category.audio_book),
    (["SSIS-698.mkv"], Category.porn),
    (["Yamagishi Aika Friday 2022 44p/1.jpg"], Category.photobook),
  ],
)
@pytest.mark.asyncio
async def test_categorizer_agent(files: list[str], category: Category):
  setupLogfire()

  req = PlanRequest(
    files=files,
  )

  a = agent(metadataMcp())
  res = await a.run(req.model_dump_json())
  assert res.output.category == category
