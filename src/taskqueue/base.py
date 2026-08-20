from abc import ABC, abstractmethod


class TaskQueue(ABC):
    @abstractmethod
    def push(self, task_id: str) -> None:
        raise NotImplementedError

    @abstractmethod
    def remove(self, task_id: str) -> None:
        raise NotImplementedError

    def close(self) -> None:
        pass