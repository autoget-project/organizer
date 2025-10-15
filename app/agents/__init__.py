import os

if os.getenv("XAI_API_KEY") or os.getenv("LM_STUDIO_API_BASE"):
  from .organizer import OrganizerAgent

  root_agent = OrganizerAgent()
