from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsBookResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents a book and detecting its language based solely on filenames, directory paths, and available metadata. Return an IsBookResponse object with "is_book" (yes/no/maybe), "language", and "reason".

Please repeat the prompt back as you understand it.

Classification Rules for is_book field:
- "yes": Files are clearly books (text-based formats, book-like metadata, book-specific naming patterns)
- "no": Files are clearly NOT books (audio files, videos, music, movies, software, etc.)
- "maybe": Files could be books but lack sufficient information for definitive classification

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent a book.

2. Return "no" when:
   - Files are audio formats (.mp3, .m4a, .flac, .wav, .aac, .ogg, .wma, .opus) - these are NOT books
   - Files are video formats (.mp4, .mkv, .avi, .mov, .wmv, .flv, .webm) - these are NOT books
   - Files are music files with artist - song title patterns
   - Files are software executables or archives (.exe, .dmg, .pkg, .deb, .rpm)
   - Files are clearly movies, TV shows, or other multimedia content

3. Return "yes" when:
   - Book file extensions: .pdf, .epub, .mobi, .txt, .doc, .docx, .rtf, .chm, .lit, .azw, .azw3
   - Metadata fields: title, author, ISBN, publisher, pages, edition, book-like tags
   - Filename patterns: "Author - Title", "Title - Author", ISBN patterns (10 or 13 digits), edition tokens ("2nd", "Revised Edition"), publisher names
   - Directory keywords: "book", "ebook", "novel", "textbook", "manual", "guide", or author/series names

4. Detection rules & signals (priority order):
   - Prefer metadata over filename cues; consider directory structure and dominant patterns across the whole set
   - Check that all files in the group are consistent with being books
   - Look for multiple files that represent parts of the same book (chapters, volumes, etc.)

5. Web search:
   - Use `web_search` with title + author (optional) for more detail and verification
   - Use search results to confirm if content is actually a book or something else

6. Language detection:
   - Original language, or title or metadata indicated it is translated edition
   - Use content analysis and search results to determine language
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="is_book_detector",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(IsBookResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["web_search"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "Stephen King - The Shining.pdf",
      ],
      metadata={
        "title": "The Shining",
        "author": "Stephen King",
        "description": "A horror novel by Stephen King",
      },
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
