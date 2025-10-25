from pydantic_ai import Agent

from ..ai import model
from .models import CategorizerContext, DecisionMakerResponse

_INSTRUCTION = """\
Task: You are an AI agent specialized in making the final categorization decision based on the analysis results from multiple specialized categorizer agents.

Please repeat the prompt back as you understand it.

Specifics (each bullet contains specifics about the task):

1. Input:
   - A CategorizerContext object containing:
     - "request": Original file and metadata information
     - Analysis results from multiple specialized agents:
       - is_audio_book: Audio book detection results
       - is_bango_porn: Bango porn detection results  
       - is_book: Book detection results
       - is_movie: Movie detection results
       - is_music_video: Music video detection results
       - is_music: Music detection results
       - is_photobook: Photobook detection results
       - is_porn: Porn detection results
       - is_tv_series: TV series detection results

2. Decision making criteria:
   - Priority order (high to low):
     2. Handle multiple "maybe" responses by choosing the most likely category
     3. If all agents return "no", return "unknown"
     4. Use metadata and file patterns as tie-breakers

3. Multiple "maybe" handling strategy:
   - Analyze the reasoning from each "maybe" response
   - Look for stronger indicators in one agent's reasoning vs others
   - Consider file extensions and directory structure patterns
   - Use metadata to validate or contradict agent decisions
   - Prioritize categories based on file type patterns:
     * .mp3, .m4a, .flac audio files → music, audio_book
     * .mp4, .mkv, .avi video files → movie, tv_series, music_video, porn
     * .jpg, .png, .pdf image files → photobook, book
     * Mixed file types need careful analysis

4. Special considerations for "maybe" cases:
   - Audio content: differentiate music vs audio_book by structure (chapters vs tracks)
   - Video content: differentiate movie vs tv_series by patterns (single vs multiple episodes)
   - Image content: differentiate photobook vs other media by context
   - Check for conflicting signals between agents' reasoning
   - When multiple "maybe" have equal strength, prefer the most specific category
   - If truly uncertain after analysis, return Category.unknown with explanation

5. Output:
   - Return the most appropriate Category enum value
   - Provide detailed reasoning showing how you resolved "maybe" conflicts
   - If uncertain, return Category.unknown with clear explanation
"""


def agent() -> Agent:
  return Agent(
    name="decision_maker",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=DecisionMakerResponse,
  )


if __name__ == "__main__":
  from ..ai import setupLogfire

  if model():
    setupLogfire()

    # Example test case with multiple "maybe" responses
    from ..models import PlanRequest
    from .models import IsAudioBookResponse, IsMusicResponse, SimpleAgentResponseResult

    context = CategorizerContext(
      request=PlanRequest(
        files=[
          "audio_file_01.mp3",
          "audio_file_02.mp3",
        ],
        metadata={"title": "Some Audio Content"},
      ),
      is_audio_book=IsAudioBookResponse(
        is_audio_book=SimpleAgentResponseResult.maybe,
        reason="Could be audiobook but lacks clear chapter structure",
      ),
      is_music=IsMusicResponse(
        is_music=SimpleAgentResponseResult.maybe,
        language="",
        reason="Could be music but track numbering is unclear",
      ),
      is_bango_porn=None,
      is_book=None,
      is_movie=None,
      is_music_video=None,
      is_photobook=None,
      is_porn=None,
      is_tv_series=None,
    )

    a = agent()
    res = a.run_sync(context.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
