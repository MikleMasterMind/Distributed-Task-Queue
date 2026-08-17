from datetime import datetime
from enum import Enum
from typing import Any, Literal

from pydantic import AliasChoices, BaseModel, ConfigDict, Field, model_validator

TaskType = Literal["echo", "sleep", "fibonacci"]


class TaskStatus(str, Enum):
    PENDING = "PENDING"
    RUNNING = "RUNNING"
    SUCCESS = "SUCCESS"
    FAILED = "FAILED"


class TaskCreate(BaseModel):
    model_config = ConfigDict(
        json_schema_extra={
            "examples": [
                {"type": "echo", "payload": {"message": "hello"}},
                {"type": "sleep", "payload": {"seconds": 5}},
                {"type": "fibonacci", "payload": {"n": 40}},
            ]
        }
    )

    type: TaskType
    payload: dict[str, Any]

    @model_validator(mode="after")
    def validate_payload(self) -> "TaskCreate":
        if self.type == "echo":
            self._require_string("message")
        elif self.type == "sleep":
            self._require_non_negative_int("seconds")
        elif self.type == "fibonacci":
            self._require_non_negative_int("n")
        return self

    def _require_string(self, field: str) -> None:
        if not isinstance(self.payload.get(field), str):
            raise ValueError(f"payload.{field} must be a string")

    def _require_non_negative_int(self, field: str) -> None:
        value = self.payload.get(field)
        if isinstance(value, bool) or not isinstance(value, int) or value < 0:
            raise ValueError(f"payload.{field} must be a non-negative integer")


class TaskOut(BaseModel):
    id: str
    type: TaskType
    payload: dict[str, Any]
    status: TaskStatus
    result: dict[str, Any] | None = None
    error: str | None = None
    created_at: datetime
    started_at: datetime | None = None
    finished_at: datetime | None = None


class TaskCreated(BaseModel):
    id: str
    status: TaskStatus


class TaskFilter(BaseModel):
    status: TaskStatus | None = None
    type: TaskType | None = None
    created_after: datetime | None = None
    created_before: datetime | None = None
    started_after: datetime | None = None
    started_before: datetime | None = None
    finished_after: datetime | None = None
    finished_before: datetime | None = None


class TaskPage(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    items: list[TaskOut]
    next_token: str | None = Field(
        default=None,
        validation_alias=AliasChoices("next-token"),
        serialization_alias="next-token",
    )
    limit: int