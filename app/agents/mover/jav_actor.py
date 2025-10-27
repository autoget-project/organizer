import json
import logging
import os
import urllib.parse
from typing import Dict, Tuple

import requests
from bs4 import BeautifulSoup
from filelock import FileLock
from pydantic import BaseModel
from pydantic_ai import Agent, RunUsage, ToolOutput
from pydantic_ai.mcp import MCPServer

from ..ai import allowedTools, model

_LOGGER = logging.getLogger(__name__)


class ActorAlias(BaseModel):
  dir_to_alias: Dict[str, list[str]]
  name_to_dir: Dict[str, str]

  def find_dir(self, actor_name: str) -> str | None:
    return self.name_to_dir.get(actor_name)

  def _find_a_dir_for_list_of_actor_name(self, actor_names: list[str]) -> str:
    """
    Find the first actor has dir
    """

    for actor_name in actor_names:
      dir = self.find_dir(actor_name)
      if dir:
        return dir
    return None


def read_actor_alias() -> ActorAlias:
  """
  Read actor aliases from JSON file specified by JAV_ACTOR_FILE environment variable.

  Returns:
      ActorAlias: ActorAlias object with dir_to_alias and name_to_dir mappings

  Raises:
      FileNotFoundError: If the JAV_ACTOR_FILE does not exist
      json.JSONDecodeError: If the file contains invalid JSON
      KeyError: If JAV_ACTOR_FILE environment variable is not set
  """
  jav_actor_file = os.getenv("JAV_ACTOR_FILE")

  with open(jav_actor_file, "r", encoding="utf-8") as f:
    dir_to_alias = json.load(f)

    # Compute name_to_dir mapping
    name_to_dir = {}
    for dir_name, aliases in dir_to_alias.items():
      for alias in aliases:
        name_to_dir[alias] = dir_name

    return ActorAlias(dir_to_alias=dir_to_alias, name_to_dir=name_to_dir)


def add_actor_alias(name: str, alias: list[str]) -> str:
  """
  Add actor aliases to the JAV_ACTOR_FILE JSON file.

  Args:
      name: The directory name for the actor
      alias: List of aliases for the actor
  """
  jav_actor_file = os.getenv("JAV_ACTOR_FILE")
  lock_file = f"{jav_actor_file}.lock"

  # Use file lock to prevent concurrent access
  with FileLock(lock_file, timeout=10):
    # Read existing actor aliases
    aa = read_actor_alias()

    # Check if actor already exists using name_to_dir
    existing_dir = None
    for a in alias:
      if a in aa.name_to_dir:
        existing_dir = aa.name_to_dir[a]
        break

    if existing_dir:
      # Merge existing list and new list
      existing_aliases = set(aa.dir_to_alias[existing_dir])
      new_aliases = [a for a in alias if a not in existing_aliases]
      aa.dir_to_alias[existing_dir].extend(new_aliases)
    else:
      # Add new actor
      aa.dir_to_alias[name] = alias
      for a in alias:
        aa.name_to_dir[a] = name

    # Write back to file
    with open(jav_actor_file, "w", encoding="utf-8") as f:
      json.dump(aa.dir_to_alias, f, ensure_ascii=False, indent=2)

    return existing_dir if existing_dir else name


def search_alias(name: str) -> list[str]:
  """
  Search for actor aliases from JAVDB.

  Args:
      name: The actor name to search for

  Returns:
      List of actor aliases found
  """

  # Construct search URL
  base_url = "https://javdb.com/search?f=actor"
  encoded_name = urllib.parse.quote(name)
  search_url = f"{base_url}&q={encoded_name}"

  headers = {
    "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
  }

  try:
    response = requests.get(search_url, headers=headers)
    response.raise_for_status()

    soup = BeautifulSoup(response.text, "html.parser")

    # Find the first actor box and extract aliases from title attribute
    aliases = []
    actor_box = soup.select_one(".actor-box a")
    if actor_box and actor_box.get("title"):
      title = actor_box.get("title")
      aliases = [alias.strip() for alias in title.split(",")]

    if name not in aliases:
      aliases.append(name)

    return aliases

  except Exception as e:
    _LOGGER.error(f"Error searching for actor aliases: {e}")
    return []


_INSTRUCTION = """\
Task: You are an AI assistant specialized in expanding alias information for JAV (Japanese Adult Video) actors.

Please repeat the prompt back as you understand it.

Specifics:
1. The input JSON already includes all Japanese aliases (e.g., {"aliases": ["波多野結衣", "はたのゆい"]}). Do not add or modify Japanese names.
2. Use the web_search tool to find:
   - Simplified Chinese translations (简体中文名)
   - Traditional Chinese translations (繁体中文名)
3. Preserve all original aliases from the input.
4. Add newly discovered valid Chinese name variations, remove duplicates, and output the final JSON as: {"aliases": [<all unique names>]}.
"""


class AliasType(BaseModel):
  aliases: list[str]


def alias_agent(mcp: MCPServer) -> Agent:
  return Agent(
    name="jav_actor_alias",
    model=model(),
    instructions=_INSTRUCTION,
    output_type=ToolOutput(AliasType),
    toolsets=[mcp],
    prepare_tools=allowedTools(["web_search"]),
  )


async def find_a_dir_for_list_of_actor_name(
  alias: ActorAlias, mcp: MCPServer, actor_names: list[str]
) -> Tuple[str, RunUsage]:
  dir = alias._find_a_dir_for_list_of_actor_name(actor_names)
  if dir:
    return dir, RunUsage()

  # just add the first actor to our list.
  n = actor_names[0]
  new_actor_aliases = search_alias(n)

  a = alias_agent(mcp)
  input = AliasType(aliases=new_actor_aliases)
  res = await a.run(input.model_dump_json())
  new_actor_aliases = res.output.aliases
  dir = add_actor_alias(n, new_actor_aliases)

  return dir, res.usage()


if __name__ == "__main__":
  from ..ai import metadataMcp, setupLogfire

  if model():
    setupLogfire()

    input = AliasType(aliases=["七瀨愛麗絲", "七瀬アリス", "アリスさん", "市島亜美"])
    mcp = metadataMcp()
    a = alias_agent(mcp)
    res = a.run_sync(input.model_dump_json())
    print(f"output: {res.output}")
    print(f"usage: {res.usage()}")
