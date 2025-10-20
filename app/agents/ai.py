import os
from typing import Union
from pydantic_ai import RunContext, ToolDefinition
from pydantic_ai.tools import ToolsPrepareFunc
from pydantic_ai.mcp import MCPServerStreamableHTTP, MCPServer
from pydantic_ai.models import Model
from pydantic_ai.models.openai import OpenAIChatModel
from pydantic_ai.providers.openai import OpenAIProvider

metadata_mcp: str = os.getenv("METADATA_MCP")


def metadataMcp() -> MCPServer:
  return MCPServerStreamableHTTP(metadata_mcp)


def allowedTools(names: list[str]) -> ToolsPrepareFunc[None]:
  async def prepareTools(
    ctx: RunContext[None], tool_defs: list[ToolDefinition]
  ) -> Union[list[ToolDefinition], None]:
    return [tool_def for tool_def in tool_defs if tool_def.name in names]

  return prepareTools


def model() -> Model | str | None:
  if os.getenv("GROK_API_KEY"):
    return os.getenv("MODEL")

  if os.getenv("LM_STUDIO_API_BASE"):
    return OpenAIChatModel(
      model_name=os.getenv("MODEL"),
      provider=OpenAIProvider(api_key="key", base_url=os.getenv("LM_STUDIO_API_BASE")),
    )
  return None


def setupLogfire():
  if os.getenv("LOGFIRE_TOKEN"):
    import logfire

    logfire.configure()
    logfire.instrument_pydantic_ai()
