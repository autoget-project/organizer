from os import path

from pydantic_ai import Agent, ToolOutput

from ..ai import model, setupLogfire
from ..categorizer.models import PlanRequestWithCategory
from ..models import Category, Language, MoverResponse, SimpleAgentResponseResult, TargetDir


def _build_instruction(target_dir: TargetDir, language: Language) -> str:
  return """\
Task: You are an AI system that organizes TV series downloads into Jellyfin's preferred folder and file
naming conventions.

You must analyze a list of downloaded files along with provided TV series metadata, and produce a JSON
plan describing how each file should be renamed and moved.

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - JSON object containing:
     - "request.files": array of file paths.
     - "request.metadata": optional fields like title, description, etc.
     - "tv_series": object containing TV series metadata from previous AI analysis, including:
       - "tv_series_name": the TV series title
       - "tv_series_name_in_chinese": Simplified Chinese title (简体中文) if available
       - "the_first_season_release_year": the year of the first season
       - "language": the TV series language
       - "is_anim": whether it's an animation
2. Analyze:
   - Use the provided TV series metadata instead of searching.
   - Parse filenames to extract season and episode numbers.
   - Only keep video (.mp4, .mkv, .avi) and subtitle (.srt, .ass, .sub) files.
   - Exclude non-media (.jpg, .png, .nfo, etc.).
3. Use provided TV series information:
   - Use the tv_series_name_in_chinese (Simplified Chinese) if available, otherwise use tv_series_name.
   - Use the provided the_first_season_release_year for the year in folder and filenames.
4. Construct new Jellyfin-compatible names:
   - Folder: `$PATH$/<Series Name (Year)>/Season XX`
   - Video: `<Series Name (Year)> SXXEYY.ext`
   - Subtitle:
     - If language is identifiable: `<Series Name (Year)> SXXEYY.<Language>.<ISO639-2>.ext`
       - Example: `我和僵尸有个约会 (2000) S02E01.简体中文.chi.srt`
     - If language cannot be determined: `<Series Name (Year)> SXXEYY.ext`
   - Place subtitles in the same season folder as their matching video and pair by episode number.
5. Edge cases:
   - If the file doesn't match the provided TV series or is an extra, `"action": "skip"`.
   - If multiple series detected, separate into logical groups.
```
""".replace("$PATH$", path.join(target_dir.name, language.name))


def agent(target_dir: TargetDir, language: Language) -> Agent:
  return Agent(
    name="tv_series_mover",
    model=model(),
    instructions=_build_instruction(target_dir, language),
    output_type=ToolOutput(MoverResponse),
  )


async def move(req: PlanRequestWithCategory) -> MoverResponse:
  target_dir = (
    TargetDir.anim_tv_series
    if req.tv_series.is_anim == SimpleAgentResponseResult.yes
    else TargetDir.tv_series
  )
  language = req.tv_series.language
  a = agent(target_dir, language)
  res = await a.run(req.model_dump_json())
  return res.output


if __name__ == "__main__":
  import asyncio

  from ..categorizer.models import IsTVSeriesResponse
  from ..models import PlanRequest

  if model():
    setupLogfire()

    req = PlanRequestWithCategory(
      request=PlanRequest(
        files=[
          "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv",
          "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP2.mkv",
          "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass",
          "My.Date.with.a.Vampire.Season.02.2000/cover.jpg",
          "My.Date.with.a.Vampire.Season.02.2000/behind the scenes.mp4.part",
        ]
      ),
      category=Category.tv_series,
      tv_series=IsTVSeriesResponse(
        is_tv_series=SimpleAgentResponseResult.yes,
        is_anim=SimpleAgentResponseResult.no,
        tv_series_name="My Date with a Vampire",
        tv_series_name_in_chinese="我和僵尸有个约会",
        the_first_season_release_year=1998,
        language=Language.chinese,
        reason="metadata from tmdb",
      ),
    )

    res = asyncio.run(move(req))
    print(f"output: ${res}")
