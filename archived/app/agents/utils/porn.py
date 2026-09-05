import os

from ..models import PlanAction, PlanRequest, PlanResponse

# Define video and subtitle file extensions
_VIDEO_EXT = {".mp4", ".mkv", ".avi", ".mov", ".wmv"}
_SUB_EXT = {".srt", ".sub", ".ass", ".ssa", ".vtt"}


def filter_unrelated_files(req: PlanRequest) -> tuple[PlanRequest, PlanResponse, list[str]]:
  """
  Filter out non-video and non-subtitle files from the request.

  Args:
      req: PlanRequest containing list of files to filter

  Returns:
      tuple[PlanRequest, PlanResponse, list[str]]:
          - New PlanRequest with only video files (subtitles excluded)
          - PlanResponse with skip actions for non-media files
          - List of subtitle files
  """

  # Lists to store filtered files, skip actions, and subtitle files
  video_files = []
  subtitle_files = []
  skip_actions = []

  for file_path in req.files:
    # Get file extension
    _, ext = os.path.splitext(file_path.lower())

    if ext in _VIDEO_EXT:
      # This is a video file, keep it in the new PlanRequest
      video_files.append(file_path)
    elif ext in _SUB_EXT:
      # This is a subtitle file, separate it
      subtitle_files.append(file_path)
    else:
      # This is not a media file, add to skip actions
      skip_actions.append(PlanAction(file=file_path, action="skip"))

  # Create new PlanRequest with only video files (subtitles excluded)
  filtered_request = PlanRequest(files=video_files, metadata=req.metadata)

  # Create PlanResponse with skip actions for non-media files
  skip_response = PlanResponse(plan=skip_actions)

  return filtered_request, skip_response, subtitle_files
