from typing import Tuple

from pydantic_ai import Agent, ToolOutput
from pydantic_ai.usage import RunUsage

from ..ai import model
from ..models import MoverResponse, PlanRequest, PlanResponse


def _build_instruction() -> str:
  return """\
Task: You are an AI system that revises file organization plans based on user feedback.

You must analyze the original plan request, the previous AI-generated response, and the user's hint to create an improved plan.

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - Original PlanRequest containing files and metadata
   - Previous PlanResponse with the original plan
   - User hint containing feedback or corrections
2. Analysis:
   - Understand what the user wants to change based on their hint
   - Identify issues with the previous plan
   - Consider the original files and metadata
3. Generate improved plan:
   - Fix issues identified in the user hint
   - Maintain consistency with file organization best practices
   - Ensure all file paths and actions are valid
4. Response format:
   - Return a new MoverResponse with the improved plan
   - Include PlanAction objects for each file
   - Each action should be either "move" or "skip"
   - For "move" actions, include the target path
5. Edge cases:
   - If the user hint is unclear, make reasonable assumptions
   - If files cannot be organized as requested, explain why
   - Always return a valid MoverResponse object
"""


def agent() -> Agent:
  return Agent(
    name="replan_with_hints",
    model=model(),
    instructions=_build_instruction(),
    output_type=ToolOutput(MoverResponse),
  )


async def replan(
  request: PlanRequest, previous_response: PlanResponse, user_hint: str
) -> Tuple[MoverResponse, RunUsage]:
  a = agent()

  # Prepare the input for the agent
  input_data = {
    "original_request": request.model_dump(),
    "previous_response": previous_response.model_dump(),
    "user_hint": user_hint,
  }

  res = await a.run(str(input_data))
  return res.output, res.usage()


if __name__ == "__main__":
  import asyncio

  from ..ai import setupLogfire

  if model():
    setupLogfire()

  # Example usage
  request = PlanRequest(files=["test_file.mp4", "test_file.srt"], metadata={"title": "Test Movie"})

  previous_response = PlanResponse(
    plan=[
      {"file": "test_file.mp4", "action": "move", "target": "movies/Test Movie (2024).mp4"},
      {"file": "test_file.srt", "action": "move", "target": "movies/Test Movie (2024).en.srt"},
    ]
  )

  user_hint = "The year should be 2023, not 2024, and the subtitle should be marked as Chinese"

  res, usage = asyncio.run(replan(request, previous_response, user_hint))
  print(f"output: {res}")
  print(f"usage: {usage}")
