import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path

from api.schemas.tasks import TaskFilter, TaskOut, TaskStatus
from repository import FileTaskRepository


def make_task(
    *,
    type: str = "echo",
    payload: dict | None = None,
    status: TaskStatus = TaskStatus.PENDING,
    created_at: datetime | None = None,
    started_at: datetime | None = None,
    finished_at: datetime | None = None,
) -> TaskOut:
    return TaskOut(
        id=str(uuid.uuid4()),
        type=type,
        payload=payload or {"message": "hello"},
        status=status,
        created_at=created_at or datetime.now(timezone.utc),
        started_at=started_at,
        finished_at=finished_at,
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


def test_list_empty(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    page = repo.list_tasks(TaskFilter(), limit=20, cursor=None)
    assert page.items == []
    assert page.next_token is None
    assert page.limit == 20


def test_list_sorted_asc(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    base = datetime(2026, 1, 1, tzinfo=timezone.utc)
    for i in range(3):
        repo.create(make_task(created_at=base + timedelta(hours=i)))
    page = repo.list_tasks(TaskFilter(), limit=20, cursor=None)
    created = [t.created_at for t in page.items]
    assert len(created) == 3
    assert created == sorted(created)


def test_list_pagination(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    base = datetime(2026, 1, 1, tzinfo=timezone.utc)
    for i in range(5):
        repo.create(make_task(created_at=base + timedelta(hours=i)))

    first = repo.list_tasks(TaskFilter(), limit=2, cursor=None)
    assert len(first.items) == 2
    assert first.next_token is not None

    second = repo.list_tasks(TaskFilter(), limit=2, cursor=first.next_token)
    assert len(second.items) == 2
    assert second.next_token is not None

    third = repo.list_tasks(TaskFilter(), limit=2, cursor=second.next_token)
    assert len(third.items) == 1
    assert third.next_token is None

    ids = [t.id for t in first.items + second.items + third.items]
    assert len(set(ids)) == 5

    all_tasks = first.items + second.items + third.items
    created = [t.created_at for t in all_tasks]
    assert created == sorted(created)


def test_list_filter_by_status(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    repo.create(make_task(status=TaskStatus.PENDING))
    repo.create(make_task(status=TaskStatus.SUCCESS))
    page = repo.list_tasks(TaskFilter(status=TaskStatus.SUCCESS), limit=20, cursor=None)
    assert [t.status for t in page.items] == [TaskStatus.SUCCESS]


def test_list_filter_by_type(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    repo.create(make_task(type="echo"))
    repo.create(make_task(type="sleep", payload={"seconds": 5}))
    page = repo.list_tasks(TaskFilter(type="sleep"), limit=20, cursor=None)
    assert [t.type for t in page.items] == ["sleep"]


def test_list_filter_by_created_range(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    base = datetime(2026, 1, 1, tzinfo=timezone.utc)
    repo.create(make_task(created_at=base))
    repo.create(make_task(created_at=base + timedelta(days=1)))
    repo.create(make_task(created_at=base + timedelta(days=2)))
    filter = TaskFilter(created_after=base, created_before=base + timedelta(days=1))
    page = repo.list_tasks(filter, limit=20, cursor=None)
    assert len(page.items) == 2


def test_list_filter_by_started_range(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    started = datetime(2026, 1, 1, 12, 0, tzinfo=timezone.utc)
    repo.create(make_task(started_at=started, finished_at=started))
    repo.create(make_task())
    filter = TaskFilter(
        started_after=datetime(2026, 1, 1, 11, 0, tzinfo=timezone.utc),
        started_before=datetime(2026, 1, 1, 13, 0, tzinfo=timezone.utc),
    )
    page = repo.list_tasks(filter, limit=20, cursor=None)
    assert len(page.items) == 1
    assert page.items[0].started_at == started


def test_list_filter_by_finished_range(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    started = datetime(2026, 1, 1, 12, 0, tzinfo=timezone.utc)
    finished = datetime(2026, 1, 1, 12, 5, tzinfo=timezone.utc)
    repo.create(make_task(started_at=started, finished_at=finished))
    repo.create(make_task())
    filter = TaskFilter(
        finished_after=datetime(2026, 1, 1, 12, 4, tzinfo=timezone.utc),
        finished_before=datetime(2026, 1, 1, 12, 6, tzinfo=timezone.utc),
    )
    page = repo.list_tasks(filter, limit=20, cursor=None)
    assert len(page.items) == 1
    assert page.items[0].finished_at == finished