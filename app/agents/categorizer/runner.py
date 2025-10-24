from pydantic_ai.mcp import MCPServer

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
) -> Category | None:
  match category:
    # categorize_by_file_name does not return anim_ categories
    # case Category.anim_tv_series:
    # case Category.anim_movie:
    case Category.movie:
      a = is_movie_agent(mcp)
      res = await a.run(req_json)
      context.is_movie = res.output
      if res.output.is_movie == SimpleAgentResponseResult.yes:
        if res.output.is_anim == SimpleAgentResponseResult.yes:
          return Category.anim_movie
        return Category.movie

    case Category.tv_series:
      a = is_tv_series_agent(mcp)
      res = await a.run(req_json)
      context.is_tv_series = res.output
      if res.output.is_tv_series == SimpleAgentResponseResult.yes:
        if res.output.is_anim == SimpleAgentResponseResult.yes:
          return Category.anim_tv_series
        return Category.tv_series

    case Category.photobook:
      a = is_photobook_agent(mcp)
      res = await a.run(req_json)
      context.is_photobook = res.output
      if res.output.is_photobook == SimpleAgentResponseResult.yes:
        return Category.photobook

    case Category.porn:
      a = is_porn_agent(mcp)
      res = await a.run(req_json)
      context.is_porn = res.output
      if res.output.is_porn == SimpleAgentResponseResult.yes:
        return Category.porn

    case Category.bango_porn:
      context.is_bango_porn = await is_bango_porn(req, mcp)
      if context.is_bango_porn.is_bango_porn == SimpleAgentResponseResult.yes:
        return Category.bango_porn

    case Category.audio_book:
      a = is_audio_book_agent(mcp)
      res = await a.run(req_json)
      context.is_audio_book = res.output
      if res.output.is_audio_book == SimpleAgentResponseResult.yes:
        return Category.audio_book

    case Category.book:
      a = is_book_agent(mcp)
      res = await a.run(req_json)
      context.is_book = res.output
      if res.output.is_book == SimpleAgentResponseResult.yes:
        return Category.book

    case Category.music:
      a = is_music_agent(mcp)
      res = await a.run(req_json)
      context.is_music = res.output
      if res.output.is_music == SimpleAgentResponseResult.yes:
        return Category.music

    case Category.music_video:
      a = is_music_video_agent(mcp)
      res = await a.run(req_json)
      context.is_music_video = res.output
      if res.output.is_music_video == SimpleAgentResponseResult.yes:
        return Category.music_video

  return None


async def categorizer(req: PlanRequest, mcp: MCPServer) -> PlanRequestWithCategory:
  possible_categories = categorize_by_file_name(req)
  categorizer_context = CategorizerContext(request=req)
  req_json = req.model_dump_json()

  for cat in possible_categories.highly_possible_categories:
    res = await per_category_checker(req_json, cat, mcp, categorizer_context)
    if res:
      return categorizer_context.to_plan_request_with_category(res)

  for cat in possible_categories.possible_categories:
    if cat in possible_categories.highly_possible_categories:
      continue
    res = await per_category_checker(req_json, cat, mcp, categorizer_context)
    if res:
      return categorizer_context.to_plan_request_with_category(res)

  # run until here is likely unknown
  a = decision_maker_agent()
  res = await a.run(req_json)
  if res.output.category != Category.unknown:
    return categorizer_context.to_plan_request_with_category(res.output.category)

  return categorizer_context.to_plan_request_with_category_unknown(
    Category.unknown, res.output.reason
  )
