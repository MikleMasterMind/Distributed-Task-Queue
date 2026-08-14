from abc import ABC, abstractmethod

from api.schemas.tasks import TaskFilter, TaskOut, TaskPage


class TaskRepository(ABC):
    @abstractmethod
    def create(self, task: TaskOut) -> TaskOut:
        raise NotImplementedError

    @abstractmethod
    def get(self, task_id: str) -> TaskOut | None:
        raise NotImplementedError

    @abstractmethod
    def list_tasks(
        self, filter: TaskFilter, limit: int, cursor: str | None
    ) -> TaskPage:
        raise NotImplementedError