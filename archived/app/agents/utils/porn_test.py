from ..models import PlanRequest
from .porn import filter_unrelated_files


def test_filter_unrelated_files_mixed_content():
  """Test filtering mixed content with both media and non-media files."""
  req = PlanRequest(
    files=[
      "movie.mp4",
      "subtitle.srt",
      "cover.jpg",
      "info.txt",
      "episode.mkv",
      "sample.avi",
      "poster.png",
      "scene.ass",
    ],
    metadata={"language": "English"},
  )

  filtered_req, skip_res, subtitle_files = filter_unrelated_files(req)

  # Check that only video files are in the filtered request (subtitles excluded)
  expected_video_files = ["movie.mp4", "episode.mkv", "sample.avi"]
  assert filtered_req.files == expected_video_files
  assert filtered_req.metadata == req.metadata

  # Check that subtitle files are separated
  expected_subtitle_files = ["subtitle.srt", "scene.ass"]
  assert subtitle_files == expected_subtitle_files

  # Check that non-media files are in skip response
  assert len(skip_res.plan) == 3
  skip_files = [action.file for action in skip_res.plan]
  expected_skip_files = ["cover.jpg", "info.txt", "poster.png"]
  assert skip_files == expected_skip_files

  # Check that all skip actions have action="skip"
  for action in skip_res.plan:
    assert action.action == "skip"


def test_filter_unrelated_files_only_media():
  """Test when all files are media files."""
  req = PlanRequest(
    files=["video.mp4", "subtitle.srt", "movie.mkv"], metadata={"language": "Chinese"}
  )

  filtered_req, skip_res, subtitle_files = filter_unrelated_files(req)

  # Only video files should be in filtered request
  expected_video_files = ["video.mp4", "movie.mkv"]
  assert filtered_req.files == expected_video_files
  assert filtered_req.metadata == req.metadata

  # Subtitle files should be separated
  assert subtitle_files == ["subtitle.srt"]

  # No files should be skipped
  assert len(skip_res.plan) == 0


def test_filter_unrelated_files_only_non_media():
  """Test when all files are non-media files."""
  req = PlanRequest(
    files=["image.jpg", "document.pdf", "info.txt"], metadata={"language": "Japanese"}
  )

  filtered_req, skip_res, subtitle_files = filter_unrelated_files(req)

  # No files should be in filtered request
  assert len(filtered_req.files) == 0
  assert filtered_req.metadata == req.metadata

  # No subtitle files
  assert len(subtitle_files) == 0

  # All files should be skipped
  assert len(skip_res.plan) == 3
  for action in skip_res.plan:
    assert action.action == "skip"


def test_filter_unrelated_files_case_insensitive():
  """Test that file extension checking is case insensitive."""
  req = PlanRequest(files=["VIDEO.MP4", "Subtitle.SRT", "Movie.MKV", "image.JPG"], metadata={})

  filtered_req, skip_res, subtitle_files = filter_unrelated_files(req)

  # Video files should be included regardless of case
  expected_video_files = ["VIDEO.MP4", "Movie.MKV"]
  assert filtered_req.files == expected_video_files

  # Subtitle files should be separated
  assert subtitle_files == ["Subtitle.SRT"]

  # Non-media files should be skipped
  assert len(skip_res.plan) == 1
  assert skip_res.plan[0].file == "image.JPG"


def test_filter_unrelated_files_empty_request():
  """Test handling of empty file list."""
  req = PlanRequest(files=[], metadata={})

  filtered_req, skip_res, subtitle_files = filter_unrelated_files(req)

  # All should be empty
  assert len(filtered_req.files) == 0
  assert len(skip_res.plan) == 0
  assert len(subtitle_files) == 0


def test_filter_unrelated_files_only_subtitles():
  """Test when all files are subtitle files."""
  req = PlanRequest(files=["subtitle1.srt", "subtitle2.ass", "subtitle3.sub"], metadata={})

  filtered_req, skip_res, subtitle_files = filter_unrelated_files(req)

  # No video files should be in filtered request
  assert len(filtered_req.files) == 0
  assert filtered_req.metadata == req.metadata

  # All files should be in subtitle list
  expected_subtitle_files = ["subtitle1.srt", "subtitle2.ass", "subtitle3.sub"]
  assert subtitle_files == expected_subtitle_files

  # No files should be skipped
  assert len(skip_res.plan) == 0
