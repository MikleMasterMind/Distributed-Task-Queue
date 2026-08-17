from contextlib import asynccontextmanager

from fastapi import FastAPI

from api.routes import tasks
from config import TASKS_DIR
from repository import FileTaskRepository


@asynccontextmanager
async def lifespan(app: FastAPI):
    app.state.repository = FileTaskRepository(TASKS_DIR)
    yield


app = FastAPI(lifespan=lifespan)
app.include_router(tasks.router)