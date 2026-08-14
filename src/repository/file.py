import base64
import json
from datetime import datetime
from pathlib import Path

from api.schemas.tasks import TaskFilter, TaskOut, TaskPage
from repository.base import TaskRepository


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

    @staticmethod
    def _matches(task: TaskOut, filter: TaskFilter) -> bool:
        if filter.status is not None and task.status != filter.status:
            return False
        if filter.type is not None and task.type != filter.type:
            return False
        if filter.created_after is not None and task.created_at < filter.created_after:
            return False
        if filter.created_before is not None and task.created_at > filter.created_before:
            return False
        executions = len(task.executions)
        if filter.min_executions is not None and executions < filter.min_executions:
            return False
        if filter.max_executions is not None and executions > filter.max_executions:
            return False
        if filter.started_after is not None or filter.started_before is not None:
            if not any(
                _in_range(e.started_at, filter.started_after, filter.started_before)
                for e in task.executions
            ):
                return False
        if filter.finished_after is not None or filter.finished_before is not None:
            if not any(
                e.finished_at is not None
                and _in_range(
                    e.finished_at, filter.finished_after, filter.finished_before
                )
                for e in task.executions
            ):
                return False
        return True

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