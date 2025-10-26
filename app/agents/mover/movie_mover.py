from os import path

from pydantic_ai import Agent, ToolOutput

from ..ai import model
from ..categorizer.models import PlanRequestWithCategory
from ..models import Category, Language, MoverResponse, SimpleAgentResponseResult, TargetDir


def _build_instruction(target_dir: TargetDir, language: Language) -> str:
  return """\
Task: You are an AI system that organizes movie downloads into Jellyfin's preferred folder and file
naming conventions.

You must analyze a list of downloaded files along with provided movie metadata, and produce a JSON
plan describing how each file should be renamed and moved.

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - JSON object containing:
     - "request.files": array of file paths.
     - "request.metadata": optional fields like title, description, etc.
     - "movie": object containing movie metadata from previous AI analysis, including:
       - "movie_name": the movie title
       - "movie_name_in_chinese": Simplified Chinese title (简体中文) if available
       - "release_year": the release year
       - "language": the movie language
       - "is_anim": whether it's an animation
2. Analyze:
   - Use the provided movie metadata instead of searching.
   - Only keep video (.mp4, .mkv, .avi) and subtitle (.srt, .ass, .sub) files.
   - Exclude non-media (.jpg, .png, .nfo, etc.).
3. Use provided movie information:
   - Use the movie_name_in_chinese (Simplified Chinese) if available, otherwise use movie_name.
   - Use the provided release_year for the year in folder and filenames.
4. Construct new Jellyfin-compatible names:
   - Folder: `$PATH$/<Movie Name (Year)>`
   - Video: `<Movie Name (Year)>.ext`
   - Subtitle: `<Movie Name (Year)>.HumanLang.LangCode.ext`
   - Subtitle:
     - If language is identifiable: `<Movie Name (Year)>.<Language>.<ISO639-2>.ext`
       - Example: `泰坦尼克号 (2000).简体中文.chi.srt`
     - If language cannot be determined: `<Movie Name (Year)>.ext`
5. Edge cases:
   - If the file doesn't match the provided movie or is an extra, `"action": "skip"`.
   - If multiple movies detected, separate into logical groups.
```
""".replace("$PATH$", path.join(target_dir.name, language.name))


def agent(target_dir: TargetDir, language: Language) -> Agent:
  return Agent(
    name="movie_mover",
    model=model(),
    instructions=_build_instruction(target_dir, language),
    output_type=ToolOutput(MoverResponse),
  )


async def move(req: PlanRequestWithCategory) -> MoverResponse:
  targer_dir = (
    TargetDir.anim_movie if req.movie.is_anim == SimpleAgentResponseResult.yes else TargetDir.movie
  )
  Language = req.movie.language
  a = agent(targer_dir, Language)
  res = await a.run(req.model_dump_json())
  return res.output


if __name__ == "__main__":
  import asyncio

  from ..ai import setupLogfire
  from ..categorizer.models import IsMovieResponse
  from ..models import PlanRequest

  if model():
    setupLogfire()

    req = PlanRequestWithCategory(
      request=PlanRequest(
        files=[
          "The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv",
          "The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass",
          "The.Mad.Phoenix.1997/cover.jpg",
          "The.Mad.Phoenix.1997/behind the scenes.mp4.part",
        ]
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

    res = asyncio.run(move(req))
    print(f"output: ${res}")
