import uuid
from datetime import datetime, timezone

from fastapi import APIRouter, Depends, HTTPException

from api.schemas.tasks import TaskCreate, TaskCreated, TaskOut, TaskStatus
from repository import TaskRepository, get_task_repository

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
) -> TaskCreated:
    task = TaskOut(
        id=str(uuid.uuid4()),
        type=body.type,
        payload=body.payload,
        status=TaskStatus.PENDING,
        result=None,
        error=None,
        created_at=datetime.now(timezone.utc),
        executions=[],
    )
    repo.create(task)
    return TaskCreated(id=task.id, status=task.status)


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