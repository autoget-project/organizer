import os

from google.adk.agents import Agent
from google.adk.models.lite_llm import LiteLlm
from pydantic import BaseModel, Field

from .mcp_tools import mcp_search_tool
from .models import category_list

llm_model = os.getenv("MODEL")

INSTRUCTION: str = (
  """\
You are an AI agent that categorizes a set of downloaded files based on their paths and optional
metadata from the download source. The user will provide input in this JSON format:

```
{
    "files": [
        "path/to/file1.ext",
        ...
    ],
    "metadata": {
        "title": "Example Title",
        "description": "Optional description",
        ...
    }
}
```

- "files": An array of file paths from a single download request. Use these to infer the content
  type (e.g., file extensions like .mp4 for video, .epub for books) and extract potential titles,
  names, or keywords from the paths.
- "metadata": Optional object with details like title, description, or other info from the origin.
  If missing, rely solely on file paths.

Your task:
- Determine the best-fitting category for the entire set of files (treat them as a cohesive group
  from one download).
- Only use one of these exact categories: $CATEGORY_LIST$
- If the content doesn't clearly fit any category, default to the closest match based on evidence;
  do not invent new categories.
- Also detect the primary language of the content (e.g., from titles, descriptions, or file names).
  Use broad labels like: Chinese, Japanese, English, Korean. If uncertain or mixed, choose the
  dominant one. No need to specify variants like Simplified/Traditional Chinese.

To ensure accuracy:
- Use DuckDuckGoWebSearch to verify titles, keywords, or inferred names from files/metadata. For
  example, search for a title to confirm if it's a movie, TV series, or book.
- Prioritize metadata if present; otherwise, parse file paths for clues (e.g., episode numbers
  suggest tv_series).
- Handle common file types: video files (.mp4, .mkv) for movie/tv_series/anim/porn/music_video;
  audio (.mp3, .m4a) for music/audio_book; images (.jpg, .pdf) for photobook/book.

Respond only with valid JSON, no explanations, additional text, or markdown. Use this exact
structure:

{
  "category": "one_of_the_given_categories",
  "language": "detected_language"
}
"""
).replace("$CATEGORY_LIST$", ", ".join(category_list))


class CategoryResponse(BaseModel):
  category: str = Field(description="The detected category of the download.")
  language: str = Field(description="The detected language of the download.")


def agent() -> Agent:
  return Agent(
    name="categorizer",
    model=LiteLlm(model=llm_model),
    description="This agent catgorize the download",
    instruction=INSTRUCTION,
    output_schema=CategoryResponse,
    output_key="category",
    disallow_transfer_to_peers=True,  # incompatible with output_schema
    disallow_transfer_to_parent=True,  # incompatible with output_schema
    tools=[mcp_search_tool()],
  )
