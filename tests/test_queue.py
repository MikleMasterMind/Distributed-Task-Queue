import uuid

import pytest

from conftest import REDIS_URL, REQUIRE_REDIS
from taskqueue.redis import RedisTaskQueue

pytestmark = [REQUIRE_REDIS]


@pytest.fixture()
def queue():
    q = RedisTaskQueue(REDIS_URL, f"dtq:tasks:{uuid.uuid4().hex}:test")
    yield q
    q._client.delete(q._key)
    q.close()


def test_push_puts_task_in_queue(queue):
    queue.push("task-1")
    assert queue._client.lrange(queue._key, 0, -1) == [b"task-1"]


def test_remove_deletes_task_from_queue(queue):
    queue.push("task-1")
    queue.push("task-2")
    queue.remove("task-1")
    assert queue._client.lrange(queue._key, 0, -1) == [b"task-2"]


def test_remove_missing_is_noop(queue):
    queue.remove("missing")
    assert queue._client.lrange(queue._key, 0, -1) == []