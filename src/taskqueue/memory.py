from taskqueue.base import TaskQueue


class InMemoryQueue(TaskQueue):
    def __init__(self) -> None:
        self.items: list[str] = []

    def push(self, task_id: str) -> None:
        self.items.append(task_id)

    def remove(self, task_id: str) -> None:
        self.items = [i for i in self.items if i != task_id]