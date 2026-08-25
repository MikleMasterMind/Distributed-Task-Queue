import os

from dotenv import load_dotenv

load_dotenv()

DATABASE_URL = os.environ["DATABASE_URL"]
DB_AUTO_CREATE = os.getenv("DB_AUTO_CREATE", "true").lower() == "true"
QUEUE_TYPE = os.getenv("QUEUE_TYPE", "redis")
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")
QUEUE_KEY = os.getenv("QUEUE_KEY", "dtq:tasks")
LOG_LEVEL = os.getenv("LOG_LEVEL", "info")