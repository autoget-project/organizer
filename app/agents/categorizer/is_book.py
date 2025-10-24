from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, metadataMcp, model, setupLogfire
from ..models import PlanRequest
from .models import IsBookResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in determining if a group of files represents a book and detecting its language based solely on filenames, directory paths, and available metadata.

Please repeat the prompt back as you understand it.

Specifics (each bullet contains specifics about the task):

1. Input:
   - A single JSON object containing:
     - "files": array of file path strings (each may include folders and filenames)
     - "metadata" (optional): object with fields like "title", "description", "tags", etc.
   - Treat all files as a single group to determine if they represent a book.

2. Book detection criteria:
   - File types: Look for .pdf, .epub, .mobi, .txt files
   - Filename patterns: 
     - Book titles with author names (e.g., "Author Name - Book Title.pdf")
     - ISBN numbers in filenames
     - Publisher names
     - Edition information (e.g., "2nd Edition", "Revised Edition")
   - Directory structure:
     - Folder names containing "book", "ebook", "novel", "textbook", "manual", "guide"
     - Author names or book series names as folder names
   - Metadata indicators:
     - "book", "author", "publisher", "ISBN", "pages", "edition" in metadata fields
     - Book-related tags like "fiction", "non-fiction", "textbook", "novel"

4. Language detection criteria:
   - Japanese: Hiragana/Katakana/Kanji characters in filenames or metadata
   - Chinese: Chinese characters (简体/繁體) in filenames or metadata
   - Korean: Hangul characters in filenames or metadata
   - English: Latin script with English words; absence of East Asian scripts
   - Other: if none apply clearly

5. Analysis approach:
   - Analyze the entire file set as one logical unit
   - Prefer metadata over filename cues
   - Consider directory structure and dominant patterns
   - Use web_search mcp tool to verify uncertain cases - search for book titles, authors, or ISBNs
   - When uncertain, select the closest matching response based on strongest evidence
   - Provide brief reasoning for your decision
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
        "Stephen King - The Shining.epub",
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
