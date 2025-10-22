from pydantic import BaseModel
from pydantic_ai import Agent, ToolOutput
from pydantic_ai.mcp import MCPServer

from .ai import allowedTools, metadataMcp, model, setupLogfire
from .models import PlanRequest


class JAVResponse(BaseModel):
  metadata: str


def _get_instruction() -> str:
  return """\
Task: You are an AI agent specialized in retrieving JAV (Japanese Adult Video) metadata using the search_japanese_porn MCP tool.

Your goal is to extract JAV bango (番号) from filenames and search for detailed metadata.

Please repeat the prompt back as you understand it.

Specifics:
1. Input:
   - JSON object containing:
     - "files": array of file paths.
     - "metadata": optional fields like title, description, etc.
2. **JAV Bango Extraction**:
   - Extract JAV bango (番号) from filenames, which typically follows the pattern:
     - Studio/Label Prefix (usually 3-4 letters) + dash (-) + number
     - Examples: "IPZZ-002", "SSIS-698", "MIDE-947", "ABP-123"
   - Look for common JAV studio prefixes: IPZZ, SSIS, MIDE, ABP, STAR, JU, MEYD, etc.
3. **Metadata Search**:
   - Use the search_japanese_porn MCP tool with the extracted bango
   - The tool will return detailed metadata including:
     - Title, actress names, release date, studio, genre tags
     - Cover images and sample screenshots
     - Description and plot summary
4. **Output**:
   - Return the retrieved metadata as a string
   - If no bango is found or search fails, indicate that in the response
   - For multiple files, process each JAV bango found
5. **Edge Cases**:
   - If multiple JAV bango are found in filenames, process each one
   - Handle cases where metadata search returns incomplete information
   - If uncertain about extracted bango, still attempt search but note uncertainty
"""


def agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="jav_metadata",
    model=model(),
    instructions=_get_instruction(),
    output_type=ToolOutput(JAVResponse),
    toolsets=[mcp],
    prepare_tools=allowedTools(["search_japanese_porn"]),
  )


if __name__ == "__main__":
  if model():
    setupLogfire()

    req = PlanRequest(
      files=[
        "IPZZ-002.mp4",
        "SSIS-698.mkv",
      ],
    )

    a = agent(metadataMcp())
    res = a.run_sync(req.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
