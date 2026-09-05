from ..categorizer.models import PlanRequestWithCategory
from ..models import Category, PlanAction, PlanRequest, PlanResponse
from .simple_mover import simple_move_plan


def test_simple_move_plan_single_file():
  category = Category.movie
  files = ["torrent_hash/movie.mp4"]
  expected_plan = PlanResponse(
    plan=[PlanAction(file="torrent_hash/movie.mp4", action="move", target="movie/movie.mp4")]
  )
  req = PlanRequestWithCategory(category=category, request=PlanRequest(files=files))
  assert simple_move_plan(req) == expected_plan


def test_simple_move_plan_multiple_files_under_hash_dir():
  category = Category.tv_series
  files = ["torrent_hash/episode1.mp4", "torrent_hash/episode2.mp4"]
  expected_plan = PlanResponse(
    plan=[PlanAction(file="torrent_hash", action="move", target="tv_series/torrent_hash")]
  )
  req = PlanRequestWithCategory(category=category, request=PlanRequest(files=files))
  assert simple_move_plan(req) == expected_plan


def test_simple_move_plan_multiple_files_in_subdirs():
  category = Category.book
  files = [
    "torrent_hash/chapter1/page1.pdf",
    "torrent_hash/chapter1/page2.pdf",
    "torrent_hash/chapter2/page1.pdf",
  ]
  expected_plan = PlanResponse(
    plan=[
      PlanAction(file="torrent_hash/chapter1", action="move", target="book/chapter1"),
      PlanAction(file="torrent_hash/chapter2", action="move", target="book/chapter2"),
    ]
  )
  # The order of actions might vary due to set iteration, so we compare sets of actions
  req = PlanRequestWithCategory(category=category, request=PlanRequest(files=files))
  actual_plan = simple_move_plan(req)
  assert len(actual_plan.plan) == len(expected_plan.plan)
  assert set(actual_plan.plan) == set(expected_plan.plan)


def test_simple_move_plan_mixed_files_and_dirs_under_hash_dir():
  category = Category.music
  files = ["torrent_hash/song.mp3", "torrent_hash/album_art/cover.jpg"]
  expected_plan = PlanResponse(
    plan=[PlanAction(file="torrent_hash", action="move", target="music/torrent_hash")]
  )
  req = PlanRequestWithCategory(category=category, request=PlanRequest(files=files))
  assert simple_move_plan(req) == expected_plan
