from enum import Enum, auto
from typing import Any, List, Literal, Optional

from iso639 import Lang
from pydantic import BaseModel, Field


class Category(Enum):
  unknown = auto()
  movie = auto()
  tv_series = auto()
  photobook = auto()
  porn = auto()
  # Japanese and Taiwan Porn are use bango (番号) system for naming.
  bango_porn = auto()
  audio_book = auto()
  book = auto()
  music = auto()
  music_video = auto()


category_list: List[str] = [name for name, _ in Category.__members__.items()]

simple_move_categories = [
  Category.photobook,
  Category.audio_book,
  Category.book,
  Category.music,
  Category.music_video,
]


class PlanRequest(BaseModel):
  files: List[str]
  metadata: dict[str, Any] | None = None


class PlanAction(BaseModel):
  file: str = Field(description="Exact original path")
  action: Literal["move", "skip"]
  target: Optional[str] = None

  def __hash__(self) -> int:
    return hash(self.file)


class MoverResponse(BaseModel):
  plan: List[PlanAction] = []


class PlanResponse(BaseModel):
  plan: List[PlanAction] = []
  error: str | None = None


class ExecuteRequest(BaseModel):
  plan: List[PlanAction]


class APIPlanRequest(BaseModel):
  dir: str
  files: List[str]
  metadata: dict[str, Any] | None = None


class APIExecuteRequest(BaseModel):
  dir: str
  plan: List[PlanAction]


class APIReplanRequest(BaseModel):
  files: List[str]
  metadata: dict[str, Any] | None = None
  previous_response: PlanResponse
  user_hint: str


class PlanFailed(PlanAction):
  reason: str


class ExecuteResponse(BaseModel):
  failed_move: List[PlanFailed]


class SimpleAgentResponseResult(Enum):
  yes = auto()
  no = auto()
  maybe = auto()


class Language(Enum):
  Chinese = auto()
  English = auto()
  Japanese = auto()
  Korean = auto()
  Others = auto()


class TargetDir(Enum):
  audio_book = auto()
  book = auto()
  music = auto()
  music_video = auto()
  photobook = auto()
  movie = auto()
  anim_movie = auto()
  tv_series = auto()
  anim_tv_series = auto()
  porn = auto()
  porn_vr = auto()
  jav = auto()
  jav_vr = auto()
  madou = auto()


VIDEO_EXT = {".mp4", ".mkv", ".avi", ".mov", ".wmv", ".ts"}
SUB_EXT = {".srt", ".sub", ".ass", ".ssa", ".vtt"}


def iso639_to_lang_enum(iso639_code: str) -> Language:
  """Convert ISO 639-1 language code to Language enum."""
  # Map common invalid codes to valid ISO 639-1 codes
  code_mapping = {
    "cn": "zh",  # Chinese
  }

  # Convert invalid codes to valid ones
  iso639_code = code_mapping.get(iso639_code, iso639_code)

  try:
    lang = Lang(iso639_code)
    if lang.name == "English":
      return Language.English
    elif lang.name == "Chinese":
      return Language.Chinese
    elif lang.name == "Japanese":
      return Language.Japanese
    elif lang.name == "Korean":
      return Language.Korean
    else:
      return Language.Others
  except (ValueError, TypeError, AttributeError):
    return Language.Others
