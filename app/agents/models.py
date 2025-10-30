from enum import Enum, auto
from typing import Any, List, Literal, Optional

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
  plan: List[PlanAction] = None


class PlanResponse(BaseModel):
  plan: List[PlanAction] = None
  error: str | None = None


class ExecuteRequest(BaseModel):
  plan: List[PlanAction]


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
