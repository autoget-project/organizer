from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsAudioBookResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents audiobook or radio drama content (有声书, 广播剧) based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

Specifics (each bullet contains specifics about the task):

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent audiobook content.

2. Audiobook detection criteria:
   - File types: Look for .mp3, .m4a, .flac, .wav, .aac, .ogg, .wma, .opus audio files
   - Filename patterns:
     - Chapter/track numbering (e.g., "Chapter 01", "Chapter 1", "01 - Chapter Title")
     - Book title with author names (e.g., "Book Title - Author Name")
     - Part/section numbering (e.g., "Part 1", "Section 01")
     - Audiobook-specific keywords: "audiobook", "audio book", "narrated", "unabridged", "abridged"
   - Directory structure:
     - Book folders with author names (e.g., "Author Name/Book Title/")
     - Audiobook folders like "Audiobooks", "Audio Books", "有声书"
     - Series organization (e.g., "Series Name/Book 1 Title/")
   - Metadata indicators:
     - "author", "narrator", "book", "audiobook" in metadata fields
     - Book-related tags like "fiction", "non-fiction", "biography", "mystery"
     - Duration information (typical audiobook length)
     - Publisher information

3. Radio drama detection criteria (广播剧):
   - Filename patterns:
     - Episode numbering (e.g., "Episode 01", "Ep 01", "第01集")
     - Radio drama keywords: "radio drama", "广播剧", "audio drama", "drama"
     - Cast/character names in filenames
     - Scene/act numbering
   - Directory structure:
     - Radio drama folders with series names
     -广播剧 folders organization
     - Season/episode organization
   - Metadata indicators:
     - "radio drama", "广播剧", "audio drama" in metadata
     - Cast information, director, sound engineer
     - Radio station or production company information

4. Chinese audiobook/radio drama specific patterns:
   - Chinese characters in filenames and metadata
   - Chinese audiobook naming: "书名 - 作者", "有声书 - 书名"
   - Chinese radio drama: "广播剧 - 剧名", "第X集 - 剧名"
   - Narrator/voice actor information in Chinese
   - Publisher information in Chinese

5. Non-audiobook exclusions:
   - Music albums or songs (should be categorized as music)
   - Podcasts (different structure, usually topical)
   - Regular audiobooks (should be categorized as book)
   - Sound effects or samples
   - Voice recordings or memos
   - Educational lectures (unless part of an audiobook)

6. Analysis approach:
   - Analyze the entire file set as one logical unit
   - Prefer metadata over filename cues
   - Consider directory structure and dominant patterns
   - Use web_search to verify uncertain cases - search for book titles, authors, radio dramas
   - Look for consistent chapter/episode numbering across files
   - When uncertain, select the closest matching response based on strongest evidence
   - Provide brief reasoning for your decision
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="is_audio_book_detector",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(IsAudioBookResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["web_search"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "有声书 - 三体 - 第01章.mp3",
        "有声书 - 三体 - 第02章.mp3",
        "有声书 - 三体 - 第03章.mp3",
      ],
      metadata={"title": "三体", "author": "刘慈欣", "narrator": "张磊", "type": "有声书"},
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
