from contextlib import asynccontextmanager

from fastapi import FastAPI

from api.routes import tasks
from config import DATABASE_URL, DB_AUTO_CREATE
from db import PostgresTaskRepository
from repository import TaskRepository
from taskqueue import create_queue

REPOSITORY_TYPES = {
    "postgres": lambda url, auto_create: PostgresTaskRepository(url, auto_create),
}


def create_repository(db_type: str = "postgres") -> TaskRepository:
    if db_type not in REPOSITORY_TYPES:
        raise ValueError(f"Unsupported database type: {db_type}")
    return REPOSITORY_TYPES[db_type](DATABASE_URL, DB_AUTO_CREATE)


@asynccontextmanager
async def lifespan(app: FastAPI):
    app.state.repository = create_repository()
    app.state.queue = create_queue()
    try:
        yield
    finally:
        app.state.repository.dispose()
        app.state.queue.close()


app = FastAPI(lifespan=lifespan)
app.include_router(tasks.router)