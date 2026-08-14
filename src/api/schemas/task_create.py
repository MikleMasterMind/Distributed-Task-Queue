from typing import Any, Literal

from pydantic import BaseModel, model_validator

TaskType = Literal["echo", "sleep", "fibonacci"]


class TaskCreate(BaseModel):
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