from typing import Dict

from pydantic import BaseModel, Field
from pydantic_ai import RunUsage

from ..models import Category, Language, PlanRequest, SimpleAgentResponseResult


class FilenameBasedPreCatergoizerResult(BaseModel):
  # file name based regex matcher. we use this to control the order to call ai agent.
  highly_possible_categories: list[Category]
  # file extension based matcher.
  possible_categories: list[Category]


class IsAudioBookResponse(BaseModel):
  is_audio_book: SimpleAgentResponseResult
  reason: str = Field(description="provide brief reasoning for your decision")


class IsBangoPornResponse(BaseModel):
  is_bango_porn: SimpleAgentResponseResult
  is_vr: SimpleAgentResponseResult
  from_madou: SimpleAgentResponseResult = Field(description="是否麻豆出品")
  from_fc2: SimpleAgentResponseResult
  bango: str | None
  actors: list[str]
  language: Language
  reason: str = Field(description="provide brief reasoning for your decision")


class GroupIsBangoPornResponse(BaseModel):
  is_bango_porn: SimpleAgentResponseResult
  porns: Dict[str, IsBangoPornResponse] = Field(description="file to details information")


class IsBookResponse(BaseModel):
  is_book: SimpleAgentResponseResult
  language: Language
  reason: str = Field(description="provide brief reasoning for your decision")


class IsMovieResponse(BaseModel):
  is_movie: SimpleAgentResponseResult
  is_anim: SimpleAgentResponseResult
  movie_name: str | None
  movie_name_in_chinese: str | None = Field(
    description="Name of Movie in Simpilfied Chinese 简体中文"
  )
  release_year: int | None
  language: Language
  reason: str = Field(description="provide brief reasoning for your decision")


class IsMusicVideoResponse(BaseModel):
  is_music_video: SimpleAgentResponseResult
  language: Language
  reason: str = Field(description="provide brief reasoning for your decision")


class IsMusicResponse(BaseModel):
  is_music: SimpleAgentResponseResult
  language: Language
  reason: str = Field(description="provide brief reasoning for your decision")


class IsPhotobookResponse(BaseModel):
  is_photobook: SimpleAgentResponseResult
  reason: str = Field(description="provide brief reasoning for your decision")


class IsPornResponse(BaseModel):
  id: str | None = Field(description="the id from mcp tool")
  is_porn: SimpleAgentResponseResult
  is_vr: SimpleAgentResponseResult
  from_onlyfans: SimpleAgentResponseResult
  name: str | None
  actors: list[str]
  language: Language
  reason: str = Field(description="provide brief reasoning for your decision")


class GroupIsPornResponse(BaseModel):
  is_porn: SimpleAgentResponseResult
  porns: Dict[str, IsPornResponse] = Field(description="file to details information")


class IsTVSeriesResponse(BaseModel):
  is_tv_series: SimpleAgentResponseResult
  is_anim: SimpleAgentResponseResult
  tv_series_name: str | None
  tv_series_name_in_chinese: str | None = Field(
    description="Name of TV Series in Simpilfied Chinese 简体中文"
  )
  the_first_season_release_year: int | None
  language: Language
  reason: str = Field(description="provide brief reasoning for your decision")


class DecisionMakerResponse(BaseModel):
  category: Category
  reason: str = Field(description="provide brief reasoning for your decision")


class PlanRequestWithCategory(BaseModel):
  request: PlanRequest
  category: Category
  audio_book: IsAudioBookResponse | None = None
  bango_porn: GroupIsBangoPornResponse | None = None
  book: IsBookResponse | None = None
  movie: IsMovieResponse | None = None
  music_video: IsMusicVideoResponse | None = None
  music: IsMusicResponse | None = None
  photobook: IsPhotobookResponse | None = None
  porn: GroupIsPornResponse | None = None
  tv_series: IsTVSeriesResponse | None = None
  unknown_reason: str | None = None


class CategorizerContext(BaseModel):
  request: PlanRequest
  usage: RunUsage = RunUsage()
  is_audio_book: IsAudioBookResponse | None = None
  is_bango_porn: GroupIsBangoPornResponse | None = None
  is_book: IsBookResponse | None = None
  is_movie: IsMovieResponse | None = None
  is_music_video: IsMusicVideoResponse | None = None
  is_music: IsMusicResponse | None = None
  is_photobook: IsPhotobookResponse | None = None
  is_porn: GroupIsPornResponse | None = None
  is_tv_series: IsTVSeriesResponse | None = None

  def to_plan_request_with_category(self, category: Category) -> PlanRequestWithCategory:
    return PlanRequestWithCategory(
      request=self.request,
      category=category,
      audio_book=self.is_audio_book if category == Category.audio_book else None,
      bango_porn=self.is_bango_porn if category == Category.bango_porn else None,
      book=self.is_book if category == Category.book else None,
      movie=self.is_movie if category == Category.movie else None,
      music_video=self.is_music_video if category == Category.music_video else None,
      music=self.is_music if category == Category.music else None,
      photobook=self.is_photobook if category == Category.photobook else None,
      porn=self.is_porn if category == Category.porn else None,
      tv_series=self.is_tv_series if category == Category.tv_series else None,
      unknown_reason=None,
    )

  def to_plan_request_with_category_unknown(self, reason: str) -> PlanRequestWithCategory:
    return PlanRequestWithCategory(
      request=self.request,
      category=Category.unknown,
      unknown_reason=reason,
    )
