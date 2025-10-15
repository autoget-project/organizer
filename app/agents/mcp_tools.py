import os

from google.adk.tools.mcp_tool.mcp_toolset import McpToolset, StreamableHTTPConnectionParams

search_mcp: str = os.getenv("SEARCH_MCP")


def mcp_search_tool() -> McpToolset:
  return McpToolset(
    connection_params=StreamableHTTPConnectionParams(
      url=search_mcp,
    )
  )
