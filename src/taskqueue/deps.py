from fastapi import Request

from taskqueue.base import TaskQueue


def get_task_queue(request: Request) -> TaskQueue:
    return request.app.state.queue