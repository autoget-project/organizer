import os

from fastapi.testclient import TestClient

from .main import app

client = TestClient(app)


def test_execute_plan(tmp_path):
  # Setup: Create temporary directories and files using tmp_path
  download_dir = tmp_path / "download_completed_dir"
  target_dir = tmp_path / "target_dir"
  download_dir.mkdir()
  (target_dir / "documents").mkdir(parents=True)

  file_name = "test_file.txt"
  file_path_in_download = download_dir / file_name
  file_path_in_download.write_text("test content")

  # Set environment variables for the test client
  os.environ["DOWNLOAD_COMPLETED_DIR"] = str(download_dir)
  os.environ["TARGET_DIR"] = str(target_dir)

  response = client.post(
    "/v1/execute",
    json={"plan": [{"file": file_name, "action": "move", "target": f"documents/{file_name}"}]},
  )
  assert response.status_code == 200
  assert (target_dir / "documents" / file_name).exists()
  assert not file_path_in_download.exists()

  # Unset environment variables
  del os.environ["DOWNLOAD_COMPLETED_DIR"]
  del os.environ["TARGET_DIR"]


def test_execute_plan_failed(tmp_path):
  # Setup: Create temporary directories using tmp_path
  download_dir = tmp_path / "download_completed_dir_failed"
  target_dir = tmp_path / "target_dir_failed"
  download_dir.mkdir()
  (target_dir / "documents").mkdir(parents=True)

  # File that does not exist in the download directory
  non_existent_file = "non_existent_file.txt"

  # Set environment variables for the test client
  os.environ["DOWNLOAD_COMPLETED_DIR"] = str(download_dir)
  os.environ["TARGET_DIR"] = str(target_dir)

  response = client.post(
    "/v1/execute",
    json={
      "plan": [
        {"file": non_existent_file, "action": "move", "target": f"documents/{non_existent_file}"}
      ]
    },
  )
  assert response.status_code == 400
  assert "file not found" in response.json()["failed_move"][0]["reason"]
  assert not (target_dir / "documents" / non_existent_file).exists()

  # Unset environment variables
  del os.environ["DOWNLOAD_COMPLETED_DIR"]
  del os.environ["TARGET_DIR"]
