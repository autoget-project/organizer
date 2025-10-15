import uuid

from google.adk.sessions import InMemorySessionService
from google.adk.runners import Runner
from google.genai import types

from .models import PlanRequest, PlanResponse

APP_NAME = "autoget_organizer"
USER_ID = "user"


class OrganizerRunner:
  def __init__(self):
    self.session_service = InMemorySessionService()
    self.runner = Runner(app_name=APP_NAME, session_service=self.session_service)

  async def run(self, request: PlanRequest) -> PlanResponse:
    query = request.model_dump_json()

    content = types.Content(role="user", parts=[types.Part(text=query)])

    final_response_text = "Agent did not produce a final response."

    session_id = uuid.uuid4().hex

    session = await self.session_service.create_session(
      app_name=APP_NAME, user_id=USER_ID, session_id=session_id
    )

    session.state["file"] = request.files

    # Execute the agent and find the final response
    async for event in self.runner.run_async(
      user_id=USER_ID,
      session_id=session_id,
      new_message=content,
    ):
      if event.is_final_response():
        if event.content and event.content.parts:
          final_response_text = event.content.parts[0].text
        break

    await self.session_service.delete_session(
      app_name=APP_NAME, user_id=USER_ID, session_id=session_id
    )

    return PlanResponse.model_validate_json(final_response_text)
