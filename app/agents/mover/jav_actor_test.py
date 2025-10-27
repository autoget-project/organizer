import json
import os

import pytest
from pydantic_ai import RunUsage

from ..ai import metadataMcp, model, setupLogfire
from .jav_actor import (
  ActorAlias,
  add_actor_alias,
  find_a_dir_for_list_of_actor_name,
  read_actor_alias,
  search_alias,
)


def test_read_actor_alias(tmp_path):
  """Test reading actor aliases from JSON file"""
  # Create test data
  test_data = {"actor1": ["name1", "alias1"], "actor2": ["name2", "alias2", "alias3"]}

  # Write test data to temp file
  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump(test_data, f)

  # Set environment variable
  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Test reading
  result = read_actor_alias()

  # Verify structure
  assert isinstance(result, ActorAlias)
  assert result.dir_to_alias == test_data

  # Verify name_to_dir mapping
  expected_name_to_dir = {
    "name1": "actor1",
    "alias1": "actor1",
    "name2": "actor2",
    "alias2": "actor2",
    "alias3": "actor2",
  }
  assert result.name_to_dir == expected_name_to_dir


def test_read_actor_alias_empty_file(tmp_path):
  """Test reading from empty JSON file"""
  test_file = tmp_path / "empty_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump({}, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  result = read_actor_alias()

  assert isinstance(result, ActorAlias)
  assert result.dir_to_alias == {}
  assert result.name_to_dir == {}


def test_find_dir(tmp_path):
  """Test finding actor directory by alias"""
  test_data = {
    "波多野结衣": ["Yui Hatano", "波多野结衣", "波多野結衣"],
    "小泽玛利亚": ["Maria Ozawa", "小澤マリア", "小泽玛利亚"],
  }

  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump(test_data, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  actor_alias = read_actor_alias()

  # Test finding by various aliases
  assert actor_alias.find_dir("Yui Hatano") == "波多野结衣"
  assert actor_alias.find_dir("波多野結衣") == "波多野结衣"
  assert actor_alias.find_dir("小澤マリア") == "小泽玛利亚"
  assert actor_alias.find_dir("Maria Ozawa") == "小泽玛利亚"

  # Test non-existent alias
  assert actor_alias.find_dir("NonExistent") is None


def test_add_actor_alias_new_actor(tmp_path):
  """Test adding aliases for a new actor"""
  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump({}, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Add new actor
  result = add_actor_alias("new_actor", ["name1", "alias1", "alias2"])

  # Verify file was updated
  with open(test_file, "r", encoding="utf-8") as f:
    saved_data = json.load(f)

  expected = {"new_actor": ["name1", "alias1", "alias2"]}
  assert saved_data == expected

  # Verify return value is the actor name (new actor case)
  assert result == "new_actor"


def test_add_actor_alias_existing_actor_merge(tmp_path):
  """Test adding aliases to existing actor (merge)"""
  test_data = {"existing_actor": ["old_name", "old_alias"]}

  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump(test_data, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Add more aliases to existing actor (one new, one duplicate)
  result = add_actor_alias("existing_actor", ["old_alias", "new_alias1", "new_alias2"])

  # Verify file was updated
  with open(test_file, "r", encoding="utf-8") as f:
    saved_data = json.load(f)

  # Should contain original aliases plus new ones (no duplicates)
  expected_aliases = ["old_name", "old_alias", "new_alias1", "new_alias2"]
  assert set(saved_data["existing_actor"]) == set(expected_aliases)

  # Verify return value is the existing directory name (existing actor case)
  assert result == "existing_actor"


def test_add_actor_alias_duplicate_detection(tmp_path):
  """Test that duplicate aliases are not added"""
  test_data = {"actor1": ["name1", "alias1"]}

  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump(test_data, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Try to add actor with existing alias (should merge to existing actor)
  result = add_actor_alias("different_name", ["name1", "new_alias"])

  # Verify file was updated
  with open(test_file, "r", encoding="utf-8") as f:
    saved_data = json.load(f)

  # Should have merged with existing actor1, not created different_name
  assert "different_name" not in saved_data
  assert "actor1" in saved_data
  assert "new_alias" in saved_data["actor1"]

  # Verify return value is the existing directory name that was merged into
  assert result == "actor1"


def test_add_actor_alias_unicode(tmp_path):
  """Test handling of unicode characters in names and aliases"""
  test_file = tmp_path / "unicode_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump({}, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Add actor with unicode characters
  result = add_actor_alias("波多野结衣", ["Yui Hatano", "波多野结衣", "波多野結衣"])

  # Verify file was updated with proper encoding
  with open(test_file, "r", encoding="utf-8") as f:
    saved_data = json.load(f)

  expected = {"波多野结衣": ["Yui Hatano", "波多野结衣", "波多野結衣"]}
  assert saved_data == expected

  # Verify return value is the actor name (new actor case with unicode)
  assert result == "波多野结衣"


def test_add_actor_alias_empty_alias_list(tmp_path):
  """Test adding actor with empty alias list"""
  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump({}, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Add actor with empty aliases
  result = add_actor_alias("empty_actor", [])

  # Verify file was updated
  with open(test_file, "r", encoding="utf-8") as f:
    saved_data = json.load(f)

  expected = {"empty_actor": []}
  assert saved_data == expected

  # Verify return value is the actor name (new actor case)
  assert result == "empty_actor"


def test_search_alias():
  """Test searching for aliases of actor 藤森里穂 and verifying 井上遥香 is included"""
  aliases = search_alias("藤森里穂")

  # Verify that the result contains some aliases
  assert len(aliases) > 0

  # Verify that "井上遥香" is in the aliases list
  assert "井上遥香" in aliases

  # The original name should also be in the aliases
  assert "藤森里穂" in aliases


@pytest.fixture
def cleanup_env():
  """Clean up environment variable after tests"""
  original_value = os.environ.get("JAV_ACTOR_FILE")
  yield
  if original_value is not None:
    os.environ["JAV_ACTOR_FILE"] = original_value
  elif "JAV_ACTOR_FILE" in os.environ:
    del os.environ["JAV_ACTOR_FILE"]


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_find_a_dir_for_list_of_actor_name_existing_actor(tmp_path):
  """Test find_a_dir_for_list_of_actor_name when actor already exists in alias database"""
  setupLogfire()

  # Setup test data with existing actor
  test_data = {
    "波多野结衣": ["Yui Hatano", "波多野结衣", "波多野結衣", "はたの ゆい"],
    "三上悠亚": ["三上悠亚", "三上悠亜", "Yua Mikami"],
  }

  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump(test_data, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Create ActorAlias instance
  actor_alias = read_actor_alias()

  # Setup MCP server
  mcp = metadataMcp()

  # Test finding existing actor by various names
  actor_names = ["Yui Hatano", "波多野結衣", "Yui Hatano"]
  result_dir, usage = await find_a_dir_for_list_of_actor_name(actor_alias, mcp, actor_names)

  # Should find existing directory immediately without calling AI
  assert result_dir == "波多野结衣"
  assert isinstance(usage, RunUsage)
  assert usage.requests == 0  # No AI calls needed for existing actor

  # Test another existing actor
  actor_names = ["三上悠亚", "波多野結衣"]
  result_dir, usage = await find_a_dir_for_list_of_actor_name(actor_alias, mcp, actor_names)

  assert result_dir == "三上悠亚"
  assert isinstance(usage, RunUsage)
  assert usage.requests == 0


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_find_a_dir_for_list_of_actor_name_new_actor(tmp_path):
  """Test find_a_dir_for_list_of_actor_name when actor is new and needs AI search"""
  setupLogfire()

  # Setup empty test database
  test_data = {}
  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump(test_data, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Create ActorAlias instance
  actor_alias = read_actor_alias()

  # Setup MCP server
  mcp = metadataMcp()

  # Test finding new actor (should trigger AI search)
  actor_names = ["藤森里穂"]
  result_dir, usage = await find_a_dir_for_list_of_actor_name(actor_alias, mcp, actor_names)

  # Should create new directory with first actor name
  assert result_dir == "藤森里穂"
  assert isinstance(usage, RunUsage)
  assert usage.requests > 0  # Should have made AI calls

  # Verify the actor was added to the database
  updated_actor_alias = read_actor_alias()
  assert "藤森里穂" in updated_actor_alias.dir_to_alias

  # Verify multiple aliases were found including Chinese translations
  aliases = updated_actor_alias.dir_to_alias["藤森里穂"]
  assert len(aliases) >= 1  # Should have original name plus any found aliases
  assert "藤森里穂" in aliases


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_find_a_dir_for_list_of_actor_name_multiple_new_actors(tmp_path):
  """Test find_a_dir_for_list_of_actor_name with multiple new actor names"""
  setupLogfire()

  # Setup empty test database
  test_data = {}
  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump(test_data, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Create ActorAlias instance
  actor_alias = read_actor_alias()

  # Setup MCP server
  mcp = metadataMcp()

  # Test with multiple new actor names (should use first one)
  actor_names = ["葵伊吹", "Jane Doe"]
  result_dir, usage = await find_a_dir_for_list_of_actor_name(actor_alias, mcp, actor_names)

  # Should create new directory with first actor name
  assert result_dir == "葵伊吹"
  assert isinstance(usage, RunUsage)
  assert usage.requests > 0  # Should have made AI calls

  # Verify the actor was added to the database
  updated_actor_alias = read_actor_alias()
  assert "葵伊吹" in updated_actor_alias.dir_to_alias

  # Verify aliases include the input names and any additional ones found
  aliases = updated_actor_alias.dir_to_alias["葵伊吹"]
  assert "葵伊吹" in aliases
  # Should also include Chinese translations found by AI
  assert len(aliases) >= 1


@pytest.mark.skipif(model() is None, reason="No env var for ai model")
@pytest.mark.asyncio
async def test_find_a_dir_for_list_of_actor_name_existing_actor_got_new_name(tmp_path):
  """Test find_a_dir_for_list_of_actor_name with an existing got new names"""
  setupLogfire()

  # Setup test data with one existing actor
  test_data = {
    "山岸逢花": ["山岸逢花"],
  }

  test_file = tmp_path / "test_actors.json"
  with open(test_file, "w", encoding="utf-8") as f:
    json.dump(test_data, f)

  os.environ["JAV_ACTOR_FILE"] = str(test_file)

  # Create ActorAlias instance
  actor_alias = read_actor_alias()

  # Setup MCP server
  mcp = metadataMcp()

  # Test with mix where some names match existing actor
  actor_names = ["山岸あや花"]
  result_dir, usage = await find_a_dir_for_list_of_actor_name(actor_alias, mcp, actor_names)

  # Should find existing actor and not create new one
  assert result_dir == "山岸逢花"
  assert isinstance(usage, RunUsage)
  assert usage.requests > 0
