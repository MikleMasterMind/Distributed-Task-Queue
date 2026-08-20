import logging
import uuid
from datetime import datetime, timezone

from fastapi import APIRouter, Depends, HTTPException, Query, Response

from api.schemas.tasks import (
    TaskCreate,
    TaskCreated,
    TaskFilter,
    TaskOut,
    TaskPage,
    TaskStatus,
    TaskType,
)
from taskqueue import TaskQueue, get_task_queue
from repository import TaskRepository, get_task_repository
from repository.base import TaskNotFoundError, TaskNotPendingError

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/tasks", tags=["tasks"])


@router.post(
    "",
    response_model=TaskCreated,
    status_code=201,
    summary="Create a task",
    description="Create a new task with the given type and payload.",
)
def create_task(
    body: TaskCreate,
    repo: TaskRepository = Depends(get_task_repository),
    queue: TaskQueue = Depends(get_task_queue),
) -> TaskCreated:
    task = TaskOut(
        id=str(uuid.uuid4()),
        type=body.type,
        payload=body.payload,
        status=TaskStatus.PENDING,
        result=None,
        error=None,
        created_at=datetime.now(timezone.utc),
    )
    repo.create(task)
    try:
        queue.push(task.id)
    except Exception:
        logger.exception("failed to enqueue task", extra={"task_id": task.id})
        try:
            repo.delete(task.id)
        except Exception:
            pass
        raise HTTPException(status_code=503, detail="task queue unavailable")
    logger.info("task enqueued", extra={"task_id": task.id})
    return TaskCreated(id=task.id, status=task.status)


@router.get(
    "",
    response_model=TaskPage,
    summary="List tasks",
    description="Return tasks with pagination and filters.",
)
def list_tasks(
    status: TaskStatus | None = Query(default=None),
    type: TaskType | None = Query(default=None),
    created_after: datetime | None = Query(default=None),
    created_before: datetime | None = Query(default=None),
    started_after: datetime | None = Query(default=None),
    started_before: datetime | None = Query(default=None),
    finished_after: datetime | None = Query(default=None),
    finished_before: datetime | None = Query(default=None),
    limit: int = Query(default=20, ge=1, le=100),
    next_token: str | None = Query(default=None, alias="next-token"),
    repo: TaskRepository = Depends(get_task_repository),
) -> TaskPage:
    filter = TaskFilter(
        status=status,
        type=type,
        created_after=created_after,
        created_before=created_before,
        started_after=started_after,
        started_before=started_before,
        finished_after=finished_after,
        finished_before=finished_before,
    )
    return repo.list_tasks(filter=filter, limit=limit, cursor=next_token)


@router.get(
    "/{task_id}",
    response_model=TaskOut,
    summary="Get a task",
    description="Return the task with the given id.",
)
def get_task(
    task_id: str,
    repo: TaskRepository = Depends(get_task_repository),
) -> TaskOut:
    task = repo.get(task_id)
    if task is None:
        raise HTTPException(status_code=404, detail="task not found")
    return task


@router.delete(
    "/{task_id}",
    status_code=204,
    summary="Delete a task",
    description="Delete a pending task. Tasks that have already started cannot be deleted.",
)
def delete_task(
    task_id: str,
    repo: TaskRepository = Depends(get_task_repository),
    queue: TaskQueue = Depends(get_task_queue),
) -> Response:
    try:
        repo.delete(task_id)
    except TaskNotFoundError:
        raise HTTPException(status_code=404, detail="task not found")
    except TaskNotPendingError:
        raise HTTPException(status_code=409, detail="task already started")
    try:
        queue.remove(task_id)
    except Exception:
        logger.exception("failed to remove task from queue", extra={"task_id": task_id})
    return Response(status_code=204)