from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsBookResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents a book and detecting its language based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent a book.
2. Detection rules & signals (priority order):
   - Metadata fields (title, author, ISBN, publisher, pages, edition, book-like tags).
   - File extensions in { .pdf, .epub, .mobi, .txt } → positive signal for book.
   - Filename patterns: "Author - Title" or "Title - Author", ISBN patterns (10 or 13 digits), edition tokens ("2nd", "Revised Edition"), publisher names.
   - Directory keywords: folder names containing "book", "ebook", "novel", "textbook", "manual", "guide", or author/series names.
   - Prefer metadata over filename cues; consider directory structure and dominant patterns across the whole set.
3. Web search:
   - Use `web_search` with title + author (optional) for more detail.
4. Language detection:
   - Original langauge, or title or metadata indicated it is translated edition.
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
