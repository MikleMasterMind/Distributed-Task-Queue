import shutil
import time
import uuid

import pytest

from conftest import REQUIRE_REDIS

pytestmark = [
    pytest.mark.skipif(
        shutil.which("go") is None, reason="go toolchain is required for integration tests"
    ),
    REQUIRE_REDIS,
]


def wait_for(client, task_id: str, target: str, timeout: float = 10.0, interval: float = 0.05) -> dict:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        body = client.get(f"/tasks/{task_id}").json()
        if body["status"] == target:
            return body
        time.sleep(interval)
    raise AssertionError(
        f"task {task_id} did not reach {target}, last status: {body['status']}"
    )


def test_end_to_end_echo(client):
    created = client.post(
        "/tasks", json={"type": "echo", "payload": {"message": "hello"}}
    ).json()
    task = wait_for(client, created["id"], "SUCCESS")
    assert task["result"] == {"message": "hello"}
    assert task["error"] is None
    assert task["started_at"]
    assert task["finished_at"]


def test_end_to_end_fibonacci(client):
    created = client.post(
        "/tasks", json={"type": "fibonacci", "payload": {"n": 10}}
    ).json()
    task = wait_for(client, created["id"], "SUCCESS")
    assert task["result"] == {"n": 10, "value": 55}


def test_end_to_end_sleep(client):
    created = client.post(
        "/tasks", json={"type": "sleep", "payload": {"seconds": 1}}
    ).json()
    task = wait_for(client, created["id"], "SUCCESS")
    assert task["result"] == {"slept": 1}


def test_worker_pool_processes_tasks(client):
    ids = [
        client.post(
            "/tasks", json={"type": "echo", "payload": {"message": f"m{i}"}}
        ).json()["id"]
        for i in range(4)
    ]
    for task_id in ids:
        task = wait_for(client, task_id, "SUCCESS")
        assert task["result"] == {"message": task["payload"]["message"]}


def test_delete_running_task_conflict(client):
    created = client.post(
        "/tasks", json={"type": "sleep", "payload": {"seconds": 2}}
    ).json()
    wait_for(client, created["id"], "RUNNING")
    resp = client.delete(f"/tasks/{created['id']}")
    assert resp.status_code == 409
    task = wait_for(client, created["id"], "SUCCESS")
    assert task["result"] == {"slept": 2}


def test_delete_completed_task_conflict(client):
    created = client.post(
        "/tasks", json={"type": "echo", "payload": {"message": "hello"}}
    ).json()
    wait_for(client, created["id"], "SUCCESS")
    resp = client.delete(f"/tasks/{created['id']}")
    assert resp.status_code == 409
    assert client.get(f"/tasks/{created['id']}").status_code == 200


def test_delete_missing_task(client):
    resp = client.delete(f"/tasks/{uuid.uuid4()}")
    assert resp.status_code == 404


def test_invalid_payload_rejected(client):
    resp = client.post("/tasks", json={"type": "echo", "payload": {"message": 1}})
    assert resp.status_code == 422
    resp = client.post("/tasks", json={"type": "sleep", "payload": {"seconds": "x"}})
    assert resp.status_code == 422
