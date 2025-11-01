import pytest

from ..ai import model
from ..models import Category, PlanRequest
from .manual_categorizer import categorize_by_file_name, categorize_by_metadata_hints


class TestManualCategorizer:
  """Test cases for the manual categorizer function."""

  def test_video_extensions_movie_categories(self):
    """Test that video files return all movie/TV related categories."""
    req = PlanRequest(files=["movie.mp4"])
    result = categorize_by_file_name(req)

    expected_categories = {
      Category.movie,
      Category.tv_series,
      Category.porn,
      Category.bango_porn,
      Category.music_video,
    }
    assert set(result.possible_categories) == expected_categories

  def test_book_extensions(self):
    """Test that book files return book category."""
    req = PlanRequest(files=["ebook.pdf", "novel.epub", "document.txt"])
    result = categorize_by_file_name(req)

    expected = {Category.book}
    assert set(result.possible_categories) == expected

  def test_audio_extensions_music_categories(self):
    """Test that audio files return music and audio book categories."""
    req = PlanRequest(files=["song.mp3"])
    result = categorize_by_file_name(req)

    expected_categories = {Category.music, Category.audio_book}
    assert set(result.possible_categories) == expected_categories

  def test_picture_extensions_photobook(self):
    """Test that picture files return photobook category."""
    req = PlanRequest(files=["photo.jpg", "image.jpeg", "picture.png"])
    result = categorize_by_file_name(req)

    expected = {Category.photobook}
    assert set(result.possible_categories) == expected

  def test_mixed_file_types_multiple_categories(self):
    """Test that mixed file types return multiple possible categories."""
    req = PlanRequest(files=["movie.mp4", "ebook.pdf", "song.mp3", "photo.jpg"])
    result = categorize_by_file_name(req)

    # Should include categories from all file types
    expected_categories = {
      Category.movie,
      Category.book,
      Category.music,
      Category.photobook,
      Category.audio_book,
      Category.porn,  # from video file
      Category.music_video,  # from video file
      Category.bango_porn,
      Category.tv_series,
    }
    assert set(result.possible_categories) == expected_categories

  def test_multiple_video_files_no_duplicates(self):
    """Test that multiple video files don't create duplicate categories."""
    req = PlanRequest(files=["movie1.mp4", "movie2.mkv", "movie3.avi"])
    result = categorize_by_file_name(req)

    # Should still return each category only once
    expected_categories = {
      Category.movie,
      Category.tv_series,
      Category.porn,
      Category.bango_porn,
      Category.music_video,
    }
    assert set(result.possible_categories) == expected_categories

  def test_multiple_book_files_no_duplicates(self):
    """Test that multiple book files don't create duplicate categories."""
    req = PlanRequest(files=["book1.pdf", "book2.epub", "book3.txt"])
    result = categorize_by_file_name(req)

    # Should return book category only once
    expected = {Category.book}
    assert set(result.possible_categories) == expected

  def test_empty_file_list(self):
    """Test that empty file list returns empty categories."""
    req = PlanRequest(files=[])
    result = categorize_by_file_name(req)

    assert set(result.possible_categories) == set()

  def test_unknown_file_extension(self):
    """Test that unknown file extensions return no categories."""
    req = PlanRequest(files=["unknown.xyz", "noextension"])
    result = categorize_by_file_name(req)

    assert set(result.possible_categories) == set()

  def test_files_without_extensions(self):
    """Test files without extensions return no categories."""
    req = PlanRequest(files=["movie", "document"])
    result = categorize_by_file_name(req)

    assert set(result.possible_categories) == set()

  def test_complex_file_paths(self):
    """Test that complex file paths are handled correctly."""
    req = PlanRequest(
      files=["/path/to/movie.mp4", "deep/nested/path/ebook.pdf", "relative/path/song.mp3"]
    )
    result = categorize_by_file_name(req)

    expected_categories = {
      Category.movie,
      Category.book,
      Category.music,
      Category.audio_book,
      Category.tv_series,
      Category.porn,
      Category.bango_porn,
      Category.music_video,
    }
    assert set(result.possible_categories) == expected_categories

  def test_all_supported_video_extensions(self):
    """Test all supported video extensions."""
    extensions = [".mp4", ".mkv", ".avi", ".mov", ".wmv", ".ts"]
    expected_categories = {
      Category.movie,
      Category.tv_series,
      Category.porn,
      Category.bango_porn,
      Category.music_video,
    }

    for ext in extensions:
      req = PlanRequest(files=[f"test{ext}"])
      result = categorize_by_file_name(req)

      assert set(result.possible_categories) == expected_categories

  def test_all_supported_book_extensions(self):
    """Test all supported book extensions."""
    extensions = [".pdf", ".epub", ".mobi", ".txt"]
    expected = {Category.book}

    for ext in extensions:
      req = PlanRequest(files=[f"test{ext}"])
      result = categorize_by_file_name(req)

      assert set(result.possible_categories) == expected

  def test_all_supported_audio_extensions(self):
    """Test all supported audio extensions."""
    extensions = [".mp3", ".wav", ".flac", ".ogg", ".m4a"]
    expected_categories = {Category.music, Category.audio_book}

    for ext in extensions:
      req = PlanRequest(files=[f"test{ext}"])
      result = categorize_by_file_name(req)

      assert set(result.possible_categories) == expected_categories

  def test_all_supported_picture_extensions(self):
    """Test all supported picture extensions."""
    extensions = [".jpg", ".jpeg", ".png"]
    expected = {Category.photobook}

    for ext in extensions:
      req = PlanRequest(files=[f"test{ext}"])
      result = categorize_by_file_name(req)

      assert set(result.possible_categories) == expected

  def test_edge_case_single_dot_in_filename(self):
    """Test filenames with single dots but no extension."""
    req = PlanRequest(files=["movie", "ebook"])
    result = categorize_by_file_name(req)

    assert set(result.possible_categories) == set()

  def test_edge_case_multiple_dots_in_filename(self):
    """Test filenames with multiple dots - should use last extension."""
    req = PlanRequest(files=["my.movie.2024.1080p.mp4", "my.ebook.chapter1.pdf"])
    result = categorize_by_file_name(req)

    expected_categories = {
      Category.movie,
      Category.book,
      Category.tv_series,  # from video file
      Category.porn,
      Category.bango_porn,
      Category.music_video,
    }
    assert set(result.possible_categories) == expected_categories

  def test_tv_series_pattern_sXXeYY(self):
    """Test TV series pattern sXXeYY is detected as highly possible."""
    req = PlanRequest(files=["Game.of.Thrones.S01E01.mp4", "Breaking.Bad.S5E08.mkv"])
    result = categorize_by_file_name(req)

    # Should have tv_series in highly_possible_categories
    assert Category.tv_series in result.highly_possible_categories
    # Should still have all possible video categories
    assert Category.tv_series in result.possible_categories

  def test_tv_series_pattern_case_insensitive(self):
    """Test TV series pattern works case-insensitively."""
    req = PlanRequest(files=["show.s01e01.mp4", "series.S12E05.mkv", "SHOW.s1e1.avi"])
    result = categorize_by_file_name(req)

    assert Category.tv_series in result.highly_possible_categories

  def test_tv_series_pattern_single_digits(self):
    """Test TV series pattern with single digits."""
    req = PlanRequest(files=["show.s1e1.mp4", "series.s2e3.mkv"])
    result = categorize_by_file_name(req)

    assert Category.tv_series in result.highly_possible_categories

  def test_bango_porn_pattern_3char_3digit(self):
    """Test bango porn pattern with 3 chars and 3 digits."""
    req = PlanRequest(files=["ABC-123.mp4", "XYZ-456.mkv"])
    result = categorize_by_file_name(req)

    assert Category.bango_porn in result.highly_possible_categories

  def test_bango_porn_pattern_4char_4digit(self):
    """Test bango porn pattern with 4 chars and 4 digits."""
    req = PlanRequest(files=["ABCD-1234.mp4", "WXYZ-5678.mkv"])
    result = categorize_by_file_name(req)

    assert Category.bango_porn in result.highly_possible_categories

  def test_bango_porn_pattern_mixed_lengths(self):
    """Test bango porn pattern with mixed lengths."""
    req = PlanRequest(files=["ABC-1234.mp4", "WXYZ-567.mk4"])
    result = categorize_by_file_name(req)

    assert Category.bango_porn in result.highly_possible_categories

  def test_bango_porn_pattern_FC2(self):
    """Test bango porn pattern with FC2-."""
    req = PlanRequest(files=["FC2-123456.mp4", "FC2-789012.mkv"])
    result = categorize_by_file_name(req)

    assert Category.bango_porn in result.highly_possible_categories

  def test_bango_porn_pattern_FC2PPV(self):
    """Test bango porn pattern with FC2PPV-."""
    req = PlanRequest(files=["FC2PPV-123456.mp4", "FC2PPV-789012.mkv"])
    result = categorize_by_file_name(req)

    assert Category.bango_porn in result.highly_possible_categories

  def test_bango_porn_pattern_FC2_case_insensitive(self):
    """Test bango porn pattern with FC2 is case-insensitive."""
    req = PlanRequest(files=["fc2-123456.mp4", "FC2ppv-789012.mkv", "Fc2PpV-123.avi"])
    result = categorize_by_file_name(req)

    assert Category.bango_porn in result.highly_possible_categories

  def test_non_video_files_no_highly_possible(self):
    """Test that non-video files don't trigger highly possible categories."""
    req = PlanRequest(files=["S01E01.txt", "ABC-123.pdf", "FC2-123456.epub"])
    result = categorize_by_file_name(req)

    # Should not have any highly possible categories for non-video files
    assert len(result.highly_possible_categories) == 0

  def test_multiple_highly_possible_categories(self):
    """Test multiple highly possible categories can be detected."""
    req = PlanRequest(files=["S01E01.mp4", "ABC-123.mkv"])
    result = categorize_by_file_name(req)

    # Should have both tv_series and bango_porn in highly_possible_categories
    assert Category.tv_series in result.highly_possible_categories
    assert Category.bango_porn in result.highly_possible_categories

  def test_regex_patterns_in_complex_filenames(self):
    """Test regex patterns work in complex filenames."""
    req = PlanRequest(
      files=[
        "My.Favorite.Show.S02E15.1080p.WEB-DL.x264.mp4",
        "JAV.ABC-123.HD.mp4",
        "Asian.FC2PPV-456789.uncensored.mkv",
      ]
    )
    result = categorize_by_file_name(req)

    assert Category.tv_series in result.highly_possible_categories
    assert Category.bango_porn in result.highly_possible_categories


class TestCategorizeByMetadataHints:
  """Test cases for the categorize_by_metadata_hints function."""

  @pytest.mark.asyncio
  @pytest.mark.skipif(model() is None, reason="No env var for ai model")
  async def test_dmm_id_returns_bango_porn(self):
    """Test that metadata with dmm_id returns bango_porn category."""
    from ..ai import metadataMcp

    req = PlanRequest(files=["video.mp4"], metadata={"dmm_id": "pred00374"})

    mcp = metadataMcp()
    result = await categorize_by_metadata_hints(req, mcp)

    assert result == Category.bango_porn
    # Check that search result was stored in metadata
    assert "search_japanese_porn_result" in req.metadata

  @pytest.mark.asyncio
  @pytest.mark.skipif(model() is None, reason="No env var for ai model")
  async def test_imdb_id_tv_series_returns_tv_series(self):
    """Test that metadata with imdb_id for TV series returns tv_series category."""
    from ..ai import metadataMcp

    req = PlanRequest(
      files=["video.mp4"],
      metadata={"imdb_id": "tt0944947"},  # Game of Thrones
    )

    mcp = metadataMcp()
    result = await categorize_by_metadata_hints(req, mcp)

    assert result == [Category.tv_series]
    # Check that search result was stored in metadata
    assert "find_by_imdb_id_result" in req.metadata

  @pytest.mark.asyncio
  @pytest.mark.skipif(model() is None, reason="No env var for ai model")
  async def test_imdb_id_movie_returns_movie(self):
    """Test that metadata with imdb_id for movie returns movie category."""
    from ..ai import metadataMcp

    req = PlanRequest(
      files=["video.mp4"],
      metadata={"imdb_id": "tt0468569"},  # The Dark Knight
    )

    mcp = metadataMcp()
    result = await categorize_by_metadata_hints(req, mcp)

    assert result == [Category.movie]
    # Check that search result was stored in metadata
    assert "find_by_imdb_id_result" in req.metadata

  @pytest.mark.asyncio
  @pytest.mark.skipif(model() is None, reason="No env var for ai model")
  async def test_imdb_id_unknown_returns_both(self):
    """Test that metadata with unknown imdb_id returns both movie and tv_series."""
    from ..ai import metadataMcp

    req = PlanRequest(
      files=["video.mp4"],
      metadata={"imdb_id": "tt0000000"},  # Non-existent ID
    )

    mcp = metadataMcp()
    result = await categorize_by_metadata_hints(req, mcp)

    assert set(result) == {Category.movie, Category.tv_series}
    # Check that search result was stored in metadata
    assert "find_by_imdb_id_result" in req.metadata

  def test_organizer_category_returns_specified_categories(self):
    """Test that metadata with organizer_category returns the specified categories."""
    import asyncio

    req = PlanRequest(
      files=["video.mp4"], metadata={"organizer_category": [Category.movie, Category.book]}
    )

    # Create a mock MCP server since we won't actually call it
    class MockMCP:
      async def direct_call_tool(self, tool_name, params):
        pytest.fail("Should not be called")
        return {}

    mcp = MockMCP()
    result = asyncio.run(categorize_by_metadata_hints(req, mcp))

    assert result == [Category.movie, Category.book]

  def test_empty_metadata_returns_empty_list(self):
    """Test that empty metadata returns empty list."""
    import asyncio

    req = PlanRequest(files=["video.mp4"], metadata={})

    # Create a mock MCP server since we won't actually call it
    class MockMCP:
      async def direct_call_tool(self, tool_name, params):
        pytest.fail("Should not be called")
        return {}

    mcp = MockMCP()
    result = asyncio.run(categorize_by_metadata_hints(req, mcp))

    assert result == []

  def test_no_metadata_key_returns_empty_list(self):
    """Test that request without metadata key returns empty list."""
    import asyncio

    req = PlanRequest(
      files=["video.mp4"]
      # No metadata key at all
    )

    # Create a mock MCP server since we won't actually call it
    class MockMCP:
      async def direct_call_tool(self, tool_name, params):
        pytest.fail("Should not be called")
        return {}

    mcp = MockMCP()
    result = asyncio.run(categorize_by_metadata_hints(req, mcp))

    assert result == []

  def test_single_organizer_category(self):
    """Test that single organizer_category is returned as list."""
    import asyncio

    req = PlanRequest(files=["video.mp4"], metadata={"organizer_category": Category.book})

    # Create a mock MCP server since we won't actually call it
    class MockMCP:
      async def direct_call_tool(self, tool_name, params):
        pytest.fail("Should not be called")
        return {}

    mcp = MockMCP()
    result = asyncio.run(categorize_by_metadata_hints(req, mcp))

    assert result == [Category.book]
