import os
from os import path
from typing import Tuple

from pydantic import BaseModel
from pydantic_ai import Agent, ToolOutput
from pydantic_ai.usage import RunUsage

from ..ai import model
from ..models import MoverResponse


class SubtitleFileWithContent(BaseModel):
  name: str
  content: str


class SubtitleFiles(BaseModel):
  files: list[SubtitleFileWithContent]
  video_move_plan: MoverResponse


def read_subtitle_file_start(file_path: str) -> str:
  """Read the first 30 lines of the given subtitle file to help detect language."""
  try:
    full_path = path.join(os.getenv("DOWNLOAD_COMPLETED_DIR"), file_path)
    with open(full_path, "r", encoding="utf-8", errors="ignore") as f:
      content_lines = []
      for i, line in enumerate(f):
        if i >= 30:
          break
        content_lines.append(line.strip())
      return "\n".join(content_lines)
  except Exception as e:
    return f"Error reading file {file_path}: {str(e)}"


_INSTRUCTION = """\
Task: You are an AI system that organizes subtitle files to match their corresponding video files
that have already been planned for movement.

You must analyze subtitle files along with the provided video movement plan, and produce a JSON
plan describing how each subtitle file should be renamed and moved to match its video counterpart.

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - JSON object containing:
     - "files": array of subtitle file objects, each with:
       - "name": subtitle file path (.srt, .ass, .sub, etc.)
       - "content": subtitle file content (first 30 lines)
     - "video_move_plan": the movement plan for video files with their target locations
2. Analyze:
   - Use the provided content field in each subtitle file object to examine subtitle content and determine language
   - Match subtitle files to their corresponding video files by episode/season numbers
   - Extract language information from subtitle content (Chinese characters, English text, etc.)
   - If content shows an error or is empty, consider the file corrupted or unreadable
3. Construct matching subtitle names:
   - Find the matching video file from the video_move_plan
   - Use the same base name as the video file
   - Add appropriate language suffix:
     - Chinese: `<VideoName>.简体中文.chi.ext`
     - English: `<VideoName>.English.eng.ext`
     - Japanese: `<VideoName>.日本語.jpn.ext`
     - Unknown: `<VideoName>.ext`
4. Target location:
   - Place subtitles in the same folder as their matching video files
   - Use the exact same target directory path as the video file
5. Edge cases:
   - If no matching video file found, set "action": "skip"
   - If subtitle file content shows error or is empty, set "action": "skip"
   - Only process subtitle files (.srt, .ass, .sub, .vtt)
"""


def agent() -> Agent:
  return Agent(
    name="subtitle_mover",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(MoverResponse),
  )


async def move(
  dir: str, files: list[str], video_move_plan: MoverResponse
) -> Tuple[MoverResponse, RunUsage]:
  """Generate subtitle movement plan based on video movement plan."""
  # Read subtitle files contents and create SubtitleFileWithContent objects
  subtitle_files_with_content = []
  for file_path in files:
    content = read_subtitle_file_start(path.join(dir, file_path))
    subtitle_files_with_content.append(SubtitleFileWithContent(name=file_path, content=content))

  subtitle_files = SubtitleFiles(files=subtitle_files_with_content, video_move_plan=video_move_plan)

  a = agent()
  res = await a.run(subtitle_files.model_dump_json())
  return res.output, res.usage()


if __name__ == "__main__":
  import asyncio

  from ..ai import model, setupLogfire
  from ..models import PlanAction

  if model():
    setupLogfire()

    # add a test srt
    text = """\
1
00:01:10,952 --> 00:01:12,351
妖怪來了！

2
00:01:16,524 --> 00:01:17,582
大俠

3
00:01:19,527 --> 00:01:20,653
大俠！

4
00:01:28,870 --> 00:01:30,064
般若波羅密

5
00:01:34,809 --> 00:01:35,867
到了！

6
00:01:42,283 --> 00:01:43,580
我就要收服你
"""
    downloaded_dir = os.getenv("DOWNLOAD_COMPLETED_DIR")
    os.makedirs(path.join(downloaded_dir, "movie1"), exist_ok=True)

    with open(path.join(downloaded_dir, "movie1/sub.srt"), "w", encoding="utf-8") as f:
      f.write(text)

    video_move_plan = MoverResponse(
      plan=[
        PlanAction(
          file="movie1/movie1.mkv",
          target="movie/movie1 (2000)/movie1 (2000).mkv",
          action="move",
        ),
      ]
    )

    res, usage = asyncio.run(move(["movie1/sub.srt"], video_move_plan))
    print(f"output: {res}")
    print(f"usage: {usage}")
