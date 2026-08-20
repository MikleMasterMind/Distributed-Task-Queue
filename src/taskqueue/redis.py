import redis

from taskqueue.base import TaskQueue


class RedisTaskQueue(TaskQueue):
    def __init__(self, url: str, key: str) -> None:
        self._client = redis.Redis.from_url(url)
        self._key = key

    def push(self, task_id: str) -> None:
        self._client.lpush(self._key, task_id)

    def remove(self, task_id: str) -> None:
        self._client.lrem(self._key, 0, task_id)

    def close(self) -> None:
        self._client.close()