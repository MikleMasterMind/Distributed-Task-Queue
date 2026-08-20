import os
from pathlib import Path

from dotenv import load_dotenv

load_dotenv()

PROJECT_ROOT = Path(__file__).resolve().parents[1]
DATA_DIR = Path(os.getenv("DATA_DIR", PROJECT_ROOT / "data"))
TASKS_DIR = DATA_DIR / "tasks"
QUEUE_TYPE = os.getenv("QUEUE_TYPE", "redis")
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")
QUEUE_KEY = os.getenv("QUEUE_KEY", "dtq:tasks")
LOG_LEVEL = os.getenv("LOG_LEVEL", "info")