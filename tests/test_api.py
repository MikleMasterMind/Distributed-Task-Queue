import pytest

from api.schemas.tasks import TaskStatus
from app import app
from repository import TaskRepository, get_task_repository


@pytest.fixture()
def client(api_client):
    return api_client


def test_create_task(client):
    resp = client.post("/tasks", json={"type": "echo", "payload": {"message": "hello"}})
    assert resp.status_code == 201
    body = resp.json()
    assert body["id"]
    assert body["status"] == "PENDING"


def test_get_task(client):
    created = client.post(
        "/tasks", json={"type": "echo", "payload": {"message": "hello"}}
    ).json()
    resp = client.get(f"/tasks/{created['id']}")
    assert resp.status_code == 200
    body = resp.json()
    assert body["id"] == created["id"]
    assert body["type"] == "echo"
    assert body["payload"] == {"message": "hello"}
    assert body["status"] == "PENDING"
    assert body["result"] is None
    assert body["error"] is None
    assert body["created_at"]
    assert body["started_at"] is None
    assert body["finished_at"] is None


def test_get_missing_task(client):
    resp = client.get("/tasks/missing")
    assert resp.status_code == 404


def test_delete_task(client):
    created = client.post(
        "/tasks", json={"type": "echo", "payload": {"message": "hello"}}
    ).json()
    resp = client.delete(f"/tasks/{created['id']}")
    assert resp.status_code == 204
    assert client.get(f"/tasks/{created['id']}").status_code == 404


def test_delete_missing_task(client):
    resp = client.delete("/tasks/missing")
    assert resp.status_code == 404


@pytest.mark.parametrize("status", [TaskStatus.RUNNING, TaskStatus.SUCCESS, TaskStatus.FAILED])
def test_delete_not_pending_task(client, status: TaskStatus):
    created = client.post(
        "/tasks", json={"type": "echo", "payload": {"message": "hello"}}
    ).json()
    repo: TaskRepository = app.dependency_overrides[get_task_repository]()
    task = repo.get(created["id"])
    repo.create(task.model_copy(update={"status": status}))
    resp = client.delete(f"/tasks/{created['id']}")
    assert resp.status_code == 409
    assert client.get(f"/tasks/{created['id']}").status_code == 200


def test_list_tasks(client):
    created = client.post(
        "/tasks", json={"type": "echo", "payload": {"message": "hello"}}
    ).json()
    resp = client.get("/tasks")
    assert resp.status_code == 200
    body = resp.json()
    assert body["limit"] == 20
    assert body["next-token"] is None
    assert len(body["items"]) == 1
    assert body["items"][0]["id"] == created["id"]


def test_list_tasks_pagination(client):
    for i in range(3):
        client.post("/tasks", json={"type": "echo", "payload": {"message": f"{i}"}})
    first = client.get("/tasks", params={"limit": 2}).json()
    assert len(first["items"]) == 2
    assert first["next-token"]
    second = client.get(
        "/tasks", params={"limit": 2, "next-token": first["next-token"]}
    ).json()
    assert len(second["items"]) == 1
    assert second["next-token"] is None
    ids = [t["id"] for t in first["items"] + second["items"]]
    assert len(set(ids)) == 3


def test_list_tasks_filter_by_status(client):
    client.post("/tasks", json={"type": "echo", "payload": {"message": "a"}})
    client.post("/tasks", json={"type": "echo", "payload": {"message": "b"}})
    body = client.get("/tasks", params={"status": "PENDING"}).json()
    assert all(t["status"] == "PENDING" for t in body["items"])
    assert len(body["items"]) == 2


def test_list_tasks_filter_by_type(client):
    client.post("/tasks", json={"type": "echo", "payload": {"message": "a"}})
    client.post("/tasks", json={"type": "sleep", "payload": {"seconds": 1}})
    body = client.get("/tasks", params={"type": "sleep"}).json()
    assert len(body["items"]) == 1
    assert body["items"][0]["type"] == "sleep"


def test_list_tasks_invalid_limit(client):
    resp = client.get("/tasks", params={"limit": 0})
    assert resp.status_code == 422


@pytest.mark.parametrize(
    ("task_type", "payload"),
    [
        ("echo", {}),
        ("echo", {"seconds": 1}),
        ("sleep", {"message": "x"}),
        ("sleep", {"seconds": -1}),
        ("sleep", {"seconds": "5"}),
        ("sleep", {}),
        ("fibonacci", {}),
        ("fibonacci", {"n": -1}),
        ("fibonacci", {"n": "10"}),
    ],
)
def test_invalid_payload(client, task_type, payload):
    resp = client.post("/tasks", json={"type": task_type, "payload": payload})
    assert resp.status_code == 422


def test_unknown_task_type(client):
    resp = client.post("/tasks", json={"type": "unknown", "payload": {}})
    assert resp.status_code == 422