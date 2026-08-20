from config import QUEUE_KEY, QUEUE_TYPE, REDIS_URL
from taskqueue.base import TaskQueue
from taskqueue.memory import InMemoryQueue
from taskqueue.redis import RedisTaskQueue


def create_queue(kind: str | None = None, *, url: str | None = None, key: str | None = None) -> TaskQueue:
    selected = kind or QUEUE_TYPE
    if selected == "redis":
        return RedisTaskQueue(url or REDIS_URL, key or QUEUE_KEY)
    if selected == "memory":
        return InMemoryQueue()
    raise ValueError(f"unknown queue type {selected!r}")