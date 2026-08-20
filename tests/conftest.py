import os
import shutil
import subprocess
import time
import uuid
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

from app import app
from repository import FileTaskRepository, get_task_repository
from taskqueue import create_queue, get_task_queue

PROJECT_ROOT = Path(__file__).resolve().parents[1]
WORKER_DIR = PROJECT_ROOT / "worker"

REQUIRE_GO = pytest.mark.skipif(
    shutil.which("go") is None, reason="go toolchain is required"
)

QUEUE_TYPE = os.getenv("QUEUE_TYPE", "redis")
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")


def redis_reachable() -> bool:
    try:
        import redis as redis_client

        redis_client.Redis.from_url(REDIS_URL, socket_connect_timeout=0.5).ping()
        return True
    except Exception:
        return False


def _ensure_redis() -> bool:
    if redis_reachable():
        return True
    if shutil.which("docker") is None:
        return False
    subprocess.run(
        ["docker", "compose", "up", "-d", "redis"],
        cwd=PROJECT_ROOT,
        capture_output=True,
    )
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        if redis_reachable():
            return True
        time.sleep(0.5)
    return False


REQUIRE_REDIS = pytest.mark.skipif(
    QUEUE_TYPE == "redis" and not _ensure_redis(), reason="redis is not available"
)


@pytest.fixture()
def queue_key() -> str:
    return f"dtq:tasks:{uuid.uuid4().hex}"


@pytest.fixture(scope="session")
def worker_binary(tmp_path_factory: pytest.TempPathFactory) -> Path:
    binary = tmp_path_factory.mktemp("worker-build") / "dtq-worker"
    subprocess.run(
        ["go", "build", "-o", str(binary), "./cmd/worker"],
        cwd=WORKER_DIR,
        check=True,
        capture_output=True,
    )
    return binary


@pytest.fixture()
def worker(worker_binary: Path, tmp_path: Path, queue_key: str):
    log_file = tmp_path / "worker.log"
    args = [
        str(worker_binary),
        f"--tasks-dir={tmp_path}",
        f"--queue={QUEUE_TYPE}",
        "--concurrency=2",
        "--log-level=debug",
    ]
    if QUEUE_TYPE == "redis":
        args += [f"--redis-url={REDIS_URL}", f"--queue-key={queue_key}"]
    proc = subprocess.Popen(
        args,
        stdout=log_file.open("wb"),
        stderr=subprocess.STDOUT,
    )
    try:
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            if "worker started" in log_file.read_text():
                break
            if proc.poll() is not None:
                raise RuntimeError(f"worker exited early:\n{log_file.read_text()}")
            time.sleep(0.05)
        else:
            raise RuntimeError("worker did not start in time")
        yield proc
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=5)


@pytest.fixture()
def client(tmp_path: Path, worker: subprocess.Popen, queue_key: str):
    repo = FileTaskRepository(tmp_path)
    queue = create_queue(QUEUE_TYPE, url=REDIS_URL, key=queue_key)
    app.dependency_overrides[get_task_repository] = lambda: repo
    app.dependency_overrides[get_task_queue] = lambda: queue
    with TestClient(app) as c:
        yield c
    app.dependency_overrides.clear()
    queue.close()


@pytest.fixture()
def api_client(tmp_path: Path):
    repo = FileTaskRepository(tmp_path)
    queue = create_queue("memory")
    app.dependency_overrides[get_task_repository] = lambda: repo
    app.dependency_overrides[get_task_queue] = lambda: queue
    with TestClient(app) as c:
        yield c
    app.dependency_overrides.clear()