import os
import re

from pydantic_ai.mcp import MCPServer

from ..models import Category, PlanRequest
from .models import FilenameBasedPreCatergoizerResult

_VIDEO_EXT = {".mp4", ".mkv", ".avi", ".mov", ".wmv", ".ts"}
_BOOK_EXT = {".pdf", ".epub", ".mobi", ".txt"}
_AUDIO_EXT = {".mp3", ".wav", ".flac", ".ogg", ".m4a"}
_PICTURE_EXT = {".jpg", ".jpeg", ".png"}


def categorize_by_file_name(req: PlanRequest) -> FilenameBasedPreCatergoizerResult:
  """by_file_name is like an arena, it check extensions of file for all posibile categories.

  For example:

  - if req only contains .epub file, it is not possible to be a movie.
  - if req has .jpg and .mp4, it is hard to say it is movie, tv_series or photobook (sometime
    a small video as gift).
  """
  possible_categories = set()
  highly_possible_categories = set()

  # Regex patterns for highly possible categories
  tv_series_pattern = re.compile(r"s\d{1,2}e\d{1,4}", re.IGNORECASE)  # sXXeYY pattern
  bango_porn_pattern1 = re.compile(
    r"[a-z]{3,5}-\d{3,4}", re.IGNORECASE
  )  # 3-5 char "-" then 3-4 digit
  bango_porn_pattern2 = re.compile(r"(FC2|FC2PPV)-\d*", re.IGNORECASE)  # FC2- or FC2PPV-

  for file_path in req.files:
    filename = os.path.basename(file_path).lower()
    _, ext = os.path.splitext(filename)

    # Check for highly possible categories using regex
    if ext in _VIDEO_EXT:
      # Check for TV series pattern (sXXeYY)
      if tv_series_pattern.search(filename):
        highly_possible_categories.add(Category.tv_series)

      # Check for bango porn patterns
      if bango_porn_pattern1.search(filename) or bango_porn_pattern2.search(filename):
        highly_possible_categories.add(Category.bango_porn)

      possible_categories.add(Category.movie)
      possible_categories.add(Category.tv_series)
      possible_categories.add(Category.porn)
      possible_categories.add(Category.bango_porn)
      possible_categories.add(Category.music_video)

    if ext in _BOOK_EXT:
      possible_categories.add(Category.book)
    if ext in _AUDIO_EXT:
      possible_categories.add(Category.music)
      possible_categories.add(Category.audio_book)
    if ext in _PICTURE_EXT:
      possible_categories.add(Category.photobook)

  return FilenameBasedPreCatergoizerResult(
    highly_possible_categories=list(highly_possible_categories),
    possible_categories=list(possible_categories),
  )


async def categorize_by_metadata_hints(req: PlanRequest, mcp: MCPServer) -> list[Category]:
  if not req.metadata:
    return []

  if "dmm_id" in req.metadata:
    res = await mcp.direct_call_tool("search_japanese_porn", {"jav_id": req.metadata["dmm_id"]})
    req.metadata["search_japanese_porn_result"] = res
    return Category.bango_porn

  if "imdb_id" in req.metadata:
    res = await mcp.direct_call_tool("find_by_imdb_id", {"imdb_id": req.metadata["imdb_id"]})
    req.metadata["find_by_imdb_id_result"] = res
    if "tv_results" in res:
      return [Category.tv_series]
    if "movie_results" in res:
      return [Category.movie]

    return [Category.tv_series, Category.movie]

  if "organizer_category" in req.metadata:
    return req.metadata["organizer_category"]

  return []


if __name__ == "__main__":
  import asyncio

  from ..ai import metadataMcp, model, setupLogfire

  if model():
    setupLogfire()

    req = PlanRequest(
      files=[],
      # metadata={"imdb_id": "tt0369179"},
      metadata={"dmm_id": "pred00374"},
    )

    mcp = metadataMcp()
    cat = asyncio.run(categorize_by_metadata_hints(req, mcp))
    print(f"category: {cat}")
    print(f"req: {req}")
