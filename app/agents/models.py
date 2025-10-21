from enum import Enum, auto
from typing import List, Literal, Optional

from pydantic import BaseModel


class Category(Enum):
  movie = auto()
  tv_series = auto()
  anim_tv_series = auto()
  anim_movie = auto()
  photobook = auto()
  porn = auto()
  audio_book = auto()
  book = auto()
  music = auto()
  music_video = auto()


category_list: List[str] = [
  Category.movie.name,
  Category.tv_series.name,
  Category.anim_tv_series.name,
  Category.anim_movie.name,
  Category.photobook.name,
  Category.porn.name,
  Category.audio_book.name,
  Category.book.name,
  Category.music.name,
  Category.music_video.name,
]

simple_move_categories = [
  Category.photobook,
  Category.audio_book,
  Category.book,
  Category.music,
  Category.music_video,
]


class PlanRequest(BaseModel):
  files: List[str]
  metadata: dict[str, str] = None


class PlanAction(BaseModel):
  file: str
  action: Literal["move", "skip"]
  target: Optional[str] = None

  def __hash__(self) -> int:
    return hash(self.file)


class PlanResponse(BaseModel):
  plan: List[PlanAction]


class ExecuteRequest(BaseModel):
  plan: List[PlanAction]
