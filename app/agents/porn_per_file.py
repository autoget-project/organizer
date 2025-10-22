from typing import Literal

from pydantic import BaseModel
from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from .ai import allowedTools, metadataMcp, model, setupLogfire
from .models import PlanRequest


class PerPornMetadata(BaseModel):
  filename: str
  porn_type: Literal["jav", "porn"]
  metadata: dict[str, str]


class PornMetadataResponse(BaseModel):
  metadata: list[PerPornMetadata]


_INSTRUCTION = """\
Task: You are an AI agent specialized in retrieving adult content metadata using both search_porn and search_japanese_porn MCP tools.

Your goal is to identify the type of adult content from filenames and search for detailed metadata using the appropriate tool(s).

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - JSON object containing:
     - "files": array of file paths.
     - "metadata": optional fields like title, description, etc.
2. **Content Type Detection**:
   - Analyze filenames to determine if content is JAV (Japanese Adult Video) or Western/other adult content
   - JAV indicators:
     * Japanese text in filename
     * JAV bango (番号) pattern: Studio/Label Prefix (3-4 letters) + dash (-) + number
     * Common JAV studio prefixes: IPZZ, SSIS, MIDE, ABP, STAR, JU, MEYD, etc.
   - Western/other content indicators:
     * English studio names
     * Actress names in English
     * Scene titles in English
     * No JAV bango pattern
3. **Metadata Search Strategy**:
   - For JAV content: Use search_japanese_porn MCP tool with extracted bango
   - For Western/other content: Use search_porn MCP tool with relevant search terms
   - For ambiguous content: Try both tools and return the best result
4. **Search Parameters**:
   - JAV: Extract bango (e.g., "IPZZ-002", "SSIS-698") and use as search term
   - Western: Use studio name, actress names, or scene title from filename
   - Remove file extensions, quality indicators, and irrelevant text
5. **Output**:
   - Return the retrieved metadata as a key-value map
   - Include source information to porn_type (jav or porn)
   - If search fails for one tool, try the other tool
   - For multiple files, process each content item found
6. **Edge Cases**:
   - If content type is unclear, attempt both searches
   - Handle cases where metadata search returns incomplete information
   - Prioritize JAV tool when JAV bango is detected
   - If both tools return results, combine or choose the most comprehensive one
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="porn_metadata",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(PornMetadataResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["search_porn", "search_japanese_porn"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=["IPZZ-002.mp4", "Long Con1.mp4"],
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
