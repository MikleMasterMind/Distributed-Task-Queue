from datetime import datetime
from enum import Enum
from typing import Any

from pydantic import BaseModel

from api.schemas.task_create import TaskType


class TaskStatus(str, Enum):
    PENDING = "PENDING"
    RUNNING = "RUNNING"
    SUCCESS = "SUCCESS"
    FAILED = "FAILED"


class TaskExecution(BaseModel):
    started_at: datetime
    finished_at: datetime | None = None


class TaskOut(BaseModel):
    id: str
    type: TaskType
    payload: dict[str, Any]
    status: TaskStatus
    result: dict[str, Any] | None = None
    error: str | None = None
    created_at: datetime
    executions: list[TaskExecution] = []


class TaskCreated(BaseModel):
    id: str
    status: TaskStatus