from typing import Tuple

from pydantic_ai.mcp import MCPServer
from pydantic_ai.usage import RunUsage

from ..models import Category, PlanRequest, SimpleAgentResponseResult
from .decision_maker import agent as decision_maker_agent
from .is_audio_book import agent as is_audio_book_agent
from .is_bango_porn import is_bango_porn
from .is_book import agent as is_book_agent
from .is_movie import agent as is_movie_agent
from .is_music import agent as is_music_agent
from .is_music_video import agent as is_music_video_agent
from .is_photobook import agent as is_photobook_agent
from .is_porn import agent as is_porn_agent
from .is_tv_series import agent as is_tv_series_agent
from .manual_categorizer import categorize_by_file_name
from .models import CategorizerContext, PlanRequestWithCategory


async def per_category_checker(
  req: PlanRequest, req_json: str, category: Category, mcp: MCPServer, context: CategorizerContext
) -> Tuple[Category | None, RunUsage]:
  match category:
    case Category.movie:
      a = is_movie_agent(mcp)
      res = await a.run(req_json)
      context.is_movie = res.output
      if res.output.is_movie == SimpleAgentResponseResult.yes:
        return Category.movie, res.usage()

    case Category.tv_series:
      a = is_tv_series_agent(mcp)
      res = await a.run(req_json)
      context.is_tv_series = res.output
      if res.output.is_tv_series == SimpleAgentResponseResult.yes:
        return Category.tv_series, res.usage()

    case Category.photobook:
      a = is_photobook_agent(mcp)
      res = await a.run(req_json)
      context.is_photobook = res.output
      if res.output.is_photobook == SimpleAgentResponseResult.yes:
        return Category.photobook, res.usage()

    case Category.porn:
      a = is_porn_agent(mcp)
      res = await a.run(req_json)
      context.is_porn = res.output
      if res.output.is_porn == SimpleAgentResponseResult.yes:
        return Category.porn, res.usage()

    case Category.bango_porn:
      context.is_bango_porn, bango_usage = await is_bango_porn(req, mcp)
      if context.is_bango_porn.is_bango_porn == SimpleAgentResponseResult.yes:
        return Category.bango_porn, bango_usage

    case Category.audio_book:
      a = is_audio_book_agent(mcp)
      res = await a.run(req_json)
      context.is_audio_book = res.output
      if res.output.is_audio_book == SimpleAgentResponseResult.yes:
        return Category.audio_book, res.usage()

    case Category.book:
      a = is_book_agent(mcp)
      res = await a.run(req_json)
      context.is_book = res.output
      if res.output.is_book == SimpleAgentResponseResult.yes:
        return Category.book, res.usage()

    case Category.music:
      a = is_music_agent(mcp)
      res = await a.run(req_json)
      context.is_music = res.output
      if res.output.is_music == SimpleAgentResponseResult.yes:
        return Category.music, res.usage()

    case Category.music_video:
      a = is_music_video_agent(mcp)
      res = await a.run(req_json)
      context.is_music_video = res.output
      if res.output.is_music_video == SimpleAgentResponseResult.yes:
        return Category.music_video, res.usage()

  return None, RunUsage()


async def run_categorizer(
  req: PlanRequest, mcp: MCPServer
) -> Tuple[PlanRequestWithCategory, RunUsage]:
  possible_categories = categorize_by_file_name(req)
  categorizer_context = CategorizerContext(request=req)
  req_json = req.model_dump_json()

  usage = RunUsage()

  for cat in possible_categories.highly_possible_categories:
    res, u = await per_category_checker(req, req_json, cat, mcp, categorizer_context)
    usage.incr(u)
    if res:
      return categorizer_context.to_plan_request_with_category(res), usage

  for cat in possible_categories.possible_categories:
    if cat in possible_categories.highly_possible_categories:
      continue
    res, u = await per_category_checker(req, req_json, cat, mcp, categorizer_context)
    usage.incr(u)
    if res:
      return categorizer_context.to_plan_request_with_category(res), usage

  # run until here is likely unknown
  a = decision_maker_agent()
  res = await a.run(req_json)
  usage.incr(res.usage())
  if res.output.category != Category.unknown:
    return categorizer_context.to_plan_request_with_category(res.output.category), usage

  return categorizer_context.to_plan_request_with_category_unknown(res.output.reason), usage


if __name__ == "__main__":
  import asyncio

  from ..ai import metadataMcp, model, setupLogfire

  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "SSIS-456.mp4",
      ],
    )

    res, usage = asyncio.run(run_categorizer(req, metadataMcp()))
    print(f"output: {res}")
    print(f"usage: {usage}")
