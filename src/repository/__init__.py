from fastapi import Request

from repository.base import TaskRepository

__all__ = ["TaskRepository", "get_task_repository"]


def get_task_repository(request: Request) -> TaskRepository:
    return request.app.state.repository