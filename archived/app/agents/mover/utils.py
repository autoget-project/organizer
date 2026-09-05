from os import path
from typing import Tuple

from ..models import SUB_EXT, VIDEO_EXT


def filter_video_files_sub_files_and_others(
  files: list[str],
) -> Tuple[list[str], list[str], list[str]]:
  video_files = []
  sub_files = []
  others = []
  for file in files:
    ext = path.splitext(file)[1]
    if ext in VIDEO_EXT:
      video_files.append(file)
    elif ext in SUB_EXT:
      sub_files.append(file)
    else:
      others.append(file)
  return video_files, sub_files, others
