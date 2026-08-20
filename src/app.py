from contextlib import asynccontextmanager

from fastapi import FastAPI

from api.routes import tasks
from config import TASKS_DIR
from repository import FileTaskRepository
from taskqueue import create_queue


@asynccontextmanager
async def lifespan(app: FastAPI):
    app.state.repository = FileTaskRepository(TASKS_DIR)
    app.state.queue = create_queue()
    try:
        yield
    finally:
        app.state.queue.close()


app = FastAPI(lifespan=lifespan)
app.include_router(tasks.router)