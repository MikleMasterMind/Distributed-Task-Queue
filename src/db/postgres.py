import base64
import re
from datetime import datetime

from sqlalchemy import and_, create_engine, or_, select
from sqlalchemy.exc import DataError
from sqlalchemy.orm import Session

from api.schemas.tasks import TaskFilter, TaskOut, TaskPage, TaskStatus
from db.models import Base, TaskModel
from repository.base import TaskNotFoundError, TaskNotPendingError, TaskRepository


def _build_filter_conditions(filter: TaskFilter) -> list:
    conditions = []
    if filter.status is not None:
        conditions.append(TaskModel.status == filter.status.value)
    if filter.type is not None:
        conditions.append(TaskModel.type == filter.type)
    if filter.created_after is not None:
        conditions.append(TaskModel.created_at >= filter.created_after)
    if filter.created_before is not None:
        conditions.append(TaskModel.created_at <= filter.created_before)
    if filter.started_after is not None:
        conditions.append(TaskModel.started_at >= filter.started_after)
    if filter.started_before is not None:
        conditions.append(TaskModel.started_at <= filter.started_before)
    if filter.finished_after is not None:
        conditions.append(TaskModel.finished_at >= filter.finished_after)
    if filter.finished_before is not None:
        conditions.append(TaskModel.finished_at <= filter.finished_before)
    return conditions


class PostgresTaskRepository(TaskRepository):
    def __init__(self, database_url: str, auto_create: bool = True) -> None:
        sync_url = re.sub(r"\+asyncpg", "", database_url)
        self.engine = create_engine(sync_url)
        if auto_create:
            Base.metadata.create_all(self.engine)

    def create(self, task: TaskOut) -> TaskOut:
        with Session(self.engine) as session:
            model = TaskModel(
                id=task.id,
                type=task.type,
                payload=task.payload,
                status=task.status.value,
                result=task.result,
                error=task.error,
                created_at=task.created_at,
                started_at=task.started_at,
                finished_at=task.finished_at,
            )
            session.merge(model)
            session.commit()
        return task

    def get(self, task_id: str) -> TaskOut | None:
        with Session(self.engine) as session:
            try:
                model = session.get(TaskModel, task_id)
            except DataError:
                return None
            if model is None:
                return None
            return self._to_task_out(model)

    def delete(self, task_id: str) -> TaskOut:
        with Session(self.engine) as session:
            try:
                model = session.get(TaskModel, task_id)
            except DataError:
                raise TaskNotFoundError(task_id)
            if model is None:
                raise TaskNotFoundError(task_id)
            if model.status != TaskStatus.PENDING.value:
                raise TaskNotPendingError(task_id)
            task_out = self._to_task_out(model)
            session.delete(model)
            session.commit()
            return task_out

    def list_tasks(
        self, filter: TaskFilter, limit: int, cursor: str | None
    ) -> TaskPage:
        with Session(self.engine) as session:
            stmt = select(TaskModel)
            conditions = _build_filter_conditions(filter)

            if conditions:
                stmt = stmt.where(and_(*conditions))

            if cursor:
                cursor_ts, cursor_id = self._decode_cursor(cursor)
                stmt = stmt.where(
                    or_(
                        TaskModel.created_at > cursor_ts,
                        and_(
                            TaskModel.created_at == cursor_ts,
                            TaskModel.id > cursor_id,
                        ),
                    )
                )

            stmt = stmt.order_by(TaskModel.created_at, TaskModel.id)
            stmt = stmt.limit(limit + 1)

            results = session.execute(stmt).scalars().all()
            tasks = [self._to_task_out(m) for m in results]

            next_token = None
            if len(tasks) > limit:
                last = tasks[limit - 1]
                next_token = self._encode_cursor(last)

            return TaskPage(items=tasks[:limit], next_token=next_token, limit=limit)

    def dispose(self) -> None:
        self.engine.dispose()

    @staticmethod
    def _to_task_out(model: TaskModel) -> TaskOut:
        return TaskOut(
            id=model.id,
            type=model.type,
            payload=model.payload,
            status=TaskStatus(model.status),
            result=model.result,
            error=model.error,
            created_at=model.created_at,
            started_at=model.started_at,
            finished_at=model.finished_at,
        )

    @staticmethod
    def _encode_cursor(task: TaskOut) -> str:
        raw = f"{task.created_at.isoformat()}|{task.id}"
        return base64.urlsafe_b64encode(raw.encode()).decode()

    @staticmethod
    def _decode_cursor(cursor: str) -> tuple[datetime, str]:
        raw = base64.urlsafe_b64decode(cursor.encode()).decode()
        created_at, task_id = raw.split("|", 1)
        return datetime.fromisoformat(created_at), task_id
