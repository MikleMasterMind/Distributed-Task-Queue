import uuid
from datetime import datetime, timezone
from pathlib import Path

from api.schemas.tasks import TaskOut
from repository import FileTaskRepository


def make_task() -> TaskOut:
    return TaskOut(
        id=str(uuid.uuid4()),
        type="echo",
        payload={"message": "hello"},
        status="PENDING",
        created_at=datetime.now(timezone.utc),
    )


def test_create_and_get(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    task = make_task()
    repo.create(task)
    assert repo.get(task.id) == task
    assert (tmp_path / f"{task.id}.json").exists()


def test_get_missing(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    assert repo.get("missing") is None