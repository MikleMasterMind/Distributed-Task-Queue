import base64
import json
from datetime import datetime
from pathlib import Path

from api.schemas.tasks import TaskFilter, TaskOut, TaskPage, TaskStatus
from repository.base import TaskNotFoundError, TaskNotPendingError, TaskRepository


class FileTaskRepository(TaskRepository):
    def __init__(self, tasks_dir: Path) -> None:
        self.tasks_dir = tasks_dir

    def create(self, task: TaskOut) -> TaskOut:
        self.tasks_dir.mkdir(parents=True, exist_ok=True)
        path = self.tasks_dir / f"{task.id}.json"
        path.write_text(task.model_dump_json(), encoding="utf-8")
        return task

    def get(self, task_id: str) -> TaskOut | None:
        path = self.tasks_dir / f"{task_id}.json"
        if not path.exists():
            return None
        with path.open(encoding="utf-8") as f:
            return TaskOut.model_validate(json.load(f))

    def delete(self, task_id: str) -> TaskOut:
        task = self.get(task_id)
        if task is None:
            raise TaskNotFoundError(task_id)
        if task.status != TaskStatus.PENDING:
            raise TaskNotPendingError(task_id)
        (self.tasks_dir / f"{task_id}.json").unlink()
        return task

    def list_tasks(
        self, filter: TaskFilter, limit: int, cursor: str | None
    ) -> TaskPage:
        tasks = []
        if self.tasks_dir.exists():
            for path in self.tasks_dir.glob("*.json"):
                with path.open(encoding="utf-8") as f:
                    task = TaskOut.model_validate(json.load(f))
                if self._matches(task, filter):
                    tasks.append(task)
        tasks.sort(key=lambda t: (t.created_at, t.id))
        if cursor:
            cursor_key = self._decode_cursor(cursor)
            tasks = [t for t in tasks if (t.created_at, t.id) > cursor_key]
        page = tasks[: limit + 1]
        next_token = (
            self._encode_cursor(page[:limit][-1]) if len(page) > limit else None
        )
        return TaskPage(items=page[:limit], next_token=next_token, limit=limit)

    def _matches(self, task: TaskOut, filter: TaskFilter) -> bool:
        checks = [
            self._matches_status,
            self._matches_type,
            self._matches_created_range,
            self._matches_started_range,
            self._matches_finished_range,
        ]
        return all(check(task, filter) for check in checks)

    @staticmethod
    def _matches_status(task: TaskOut, filter: TaskFilter) -> bool:
        return filter.status is None or task.status == filter.status

    @staticmethod
    def _matches_type(task: TaskOut, filter: TaskFilter) -> bool:
        return filter.type is None or task.type == filter.type

    @staticmethod
    def _matches_created_range(task: TaskOut, filter: TaskFilter) -> bool:
        return _in_range(task.created_at, filter.created_after, filter.created_before)

    @staticmethod
    def _matches_started_range(task: TaskOut, filter: TaskFilter) -> bool:
        if task.started_at is None:
            return filter.started_after is None and filter.started_before is None
        return _in_range(task.started_at, filter.started_after, filter.started_before)

    @staticmethod
    def _matches_finished_range(task: TaskOut, filter: TaskFilter) -> bool:
        if task.finished_at is None:
            return filter.finished_after is None and filter.finished_before is None
        return _in_range(task.finished_at, filter.finished_after, filter.finished_before)

    @staticmethod
    def _encode_cursor(task: TaskOut) -> str:
        raw = f"{task.created_at.isoformat()}|{task.id}"
        return base64.urlsafe_b64encode(raw.encode()).decode()

    @staticmethod
    def _decode_cursor(cursor: str) -> tuple[datetime, str]:
        raw = base64.urlsafe_b64decode(cursor.encode()).decode()
        created_at, task_id = raw.split("|", 1)
        return datetime.fromisoformat(created_at), task_id


def _in_range(value: datetime, after: datetime | None, before: datetime | None) -> bool:
    if after is not None and value < after:
        return False
    if before is not None and value > before:
        return False
    return True