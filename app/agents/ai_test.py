import asyncio
import os

import pytest
from pydantic import BaseModel
from pydantic_ai import Agent, ToolOutput

from .ai import allowedTools, metadataMcp, model, setupLogfire


@pytest.mark.skipif(
  not os.getenv("METADATA_MCP"), reason="METADATA_MCP environment variable not set"
)
def test_metadata_mcp_tool():
  """Test that metadataMcp can be created when METADATA_MCP is set.
  It contains tools.
  """
  mcp = metadataMcp()
  available_tools = asyncio.run(mcp.list_tools())
  tools = [tool.name for tool in available_tools]
  tools.sort()

  want = [
    "fetch",
    "web_search",
    "wikipedia_search",
    "wikipedia_page",
    "search_movies",
    "search_tv_shows",
    "search_porn",
    "search_japanese_porn",
  ]

  want.sort()
  assert tools == want


@pytest.mark.skipif(
  not os.getenv("GROK_API_KEY") and not os.getenv("LM_STUDIO_API_BASE"),
  reason="GROK_API_KEY or LM_STUDIO_API_BASE environment variable not set",
)
@pytest.mark.asyncio
async def test_agent_tool_use():
  """Test that the agent can use tool."""

  setupLogfire()

  class CPUSpec(BaseModel):
    core_count: int
    thread_count: int
    release_year: int

  agent = Agent(
    model=model(),
    name="search",
    instructions="use web_search, answser user's question",
    output_type=ToolOutput(CPUSpec),
    toolsets=[metadataMcp()],
    prepare_tools=allowedTools(["web_search"]),
  )

  got = await agent.run_async("Spec of AMD Ryzen Al Max+ 395")
  assert got.output == CPUSpec(core_count=16, thread_count=32, release_year=2025)
