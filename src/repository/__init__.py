from fastapi import Request

from repository.base import TaskRepository
from repository.file import FileTaskRepository

__all__ = ["FileTaskRepository", "TaskRepository", "get_task_repository"]


def get_task_repository(request: Request) -> TaskRepository:
    return request.app.state.repository