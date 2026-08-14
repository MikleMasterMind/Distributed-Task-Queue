from abc import ABC, abstractmethod

from api.schemas.task_out import TaskOut


class TaskRepository(ABC):
    @abstractmethod
    def create(self, task: TaskOut) -> TaskOut:
        raise NotImplementedError

    @abstractmethod
    def get(self, task_id: str) -> TaskOut | None:
        raise NotImplementedError