import json
from pathlib import Path

from api.schemas.task_out import TaskOut
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