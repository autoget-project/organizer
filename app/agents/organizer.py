from typing import override, AsyncGenerator

from google.adk.events import Event
from google.adk.agents import BaseAgent, Agent, InvocationContext
from google.adk.agents.callback_context import CallbackContext

from .categorizer import agent as categorizer_agent, CategoryResponse
from .models import PlanRequest, Category, category_list
from .utils.utils import simple_move_plan_event
from .tv_series_mover import agent as tv_series_mover_agent
from .movie_mover import agent as movie_mover_agent


simple_move_categories = [
  Category.photobook.name,
  Category.audio_book.name,
  Category.book.name,
  Category.music.name,
  Category.music_video.name,
]


def ensure_state_files_exist(callback_context: CallbackContext):
  # the caller should put files to state
  if "file" not in callback_context.state:
    # to allow run with adk web, parse files from user_content.
    if (
      callback_context.user_content
      and callback_context.user_content.parts
      and callback_context.user_content.parts[0].text
    ):
      req = PlanRequest.model_validate_json(callback_context.user_content.parts[0].text)
      callback_context.state["files"] = req.files


class OrganizerAgent(BaseAgent):
  categorizer: Agent
  tv_series_mover: Agent
  movie_mover: Agent

  def __init__(self):
    categorizer_agent_ = categorizer_agent()
    tv_series_mover_agent_ = tv_series_mover_agent()
    movie_mover_agent_ = movie_mover_agent()

    sub_agents_list = [
      categorizer_agent_,
      tv_series_mover_agent_,
      movie_mover_agent_,
    ]

    super().__init__(
      name="organizer",
      description="this agent creates the organization plan",
      categorizer=categorizer_agent_,
      tv_series_mover=tv_series_mover_agent_,
      movie_mover=movie_mover_agent_,
      sub_agents=sub_agents_list,
      before_agent_callback=ensure_state_files_exist,
    )

  @override
  async def _run_async_impl(self, ctx: InvocationContext) -> AsyncGenerator[Event, None]:
    async for event in self.categorizer.run_async(ctx):
      yield event

    cat = CategoryResponse.model_validate(ctx.session.state["category"])
    if cat.category not in category_list:
      raise Exception(f"Unknown category: {cat.category}")

    if cat.category in simple_move_categories:
      event = simple_move_plan_event(self.name, Category[cat.category], ctx.session.state["files"])
      yield event
      return

    if cat.category == Category.tv_series.name or cat.category == Category.anim_tv_series.name:
      async for event in self.tv_series_mover.run_async(ctx):
        yield event
      return

    if cat.category == Category.movie.name or cat.category == Category.anim_movie.name:
      async for event in self.movie_mover.run_async(ctx):
        yield event
      return

    return
