from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, model
from ..models import PlanRequest
from .models import IsAudioBookResponse

_INSTRUCTION = """\
Task: You are an AI agent that determines whether a provided group of files (filenames + directory paths + optional metadata) represents an Audiobook (有声书), a Radio Drama (广播剧 / audio drama), or Other (music/podcast/lecture/etc.). Use web_search as an essential verification step for every classification. Return an IsAudioBookResponse object with "is_audio_book" (yes/no/maybe), "reason", and other relevant fields.

Please repeat the prompt back as you understand it.

Classification Rules for is_audio_book field:
- "yes": Files are clearly audiobooks or radio dramas (audio format + book/narrative content)
- "no": Files are clearly NOT audiobooks (non-audio formats, obvious music, text files, etc.)
- "maybe": Files could be audiobooks but lack sufficient information for definitive classification

Specifics (each bullet contains specifics about the task):
1. Input: A single JSON object:
   - "files": array of file path strings (each may include folders and filenames)
   - "metadata" (optional): object with fields like "title", "description", "tags", "duration", "author", "narrator", "cast", "publisher", etc.
2. Analysis order: metadata → filenames → directory structure → web_search (required verification).
3. Recognized audio file types: .mp3, .m4a, .flac, .wav, .aac, .ogg, .wma, .opus.
4. Return "no" when:
   - Files are non-audio formats (.pdf, .epub, .txt, etc.)
   - Files are clearly music (artist - song title patterns, album structures)
   - Files are clearly podcasts or lectures with no book content
   - Web search confirms the content is music/movies/other non-book content
5. Audiobook cues (return "yes" when found):
   - Filenames: chapter/track numbers, "Book Title - Author", keywords ("audiobook", "有声书", "narrated").
   - Metadata: author/narrator/book fields, book-related tags.
   - Directory: "Audiobooks", "Audio Books", "有声书", "Author/Book Title/".
6. Radio drama cues (return "yes" when found):
   - Filenames: episode numbers ("Ep01", "第01集"), keywords ("radio drama", "audio drama", "广播剧").
   - Metadata: cast, director, studio, radio station, or production company fields.
   - Directory: "广播剧" folders or season/episode subfolders.
7. Chinese-specific cues: naming patterns ("书名 - 作者", "第X集 - 剧名"), Chinese keywords, narrator/voice actor fields in Chinese.
8. Required web_search behavior:
   - Always perform a search using key extracted entities (e.g., title, author, or drama name).
   - Integrate web_search results as part of reasoning: confirm or correct classification.
   - If search results are ambiguous, note this in "reason" and consider returning "maybe".
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
  from ..ai import metadataMcp, setupLogfire

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
