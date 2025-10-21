from fastapi.testclient import TestClient

from .main import app

client = TestClient(app)


def test_execute_plan():
  response = client.post(
    "/v1/execute",
    json={
      "plan": [{"file": "/tmp/from/file1.txt", "action": "move", "target": "/tmp/to/file1.txt"}]
    },
  )
  assert response.status_code == 200
