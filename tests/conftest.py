import os
import shutil
import subprocess
import time
import uuid
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

from app import app
from db import PostgresTaskRepository
from repository import TaskRepository, get_task_repository
from taskqueue import create_queue, get_task_queue

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WORKER_DIR = os.path.join(PROJECT_ROOT, "worker")

REQUIRE_GO = pytest.mark.skipif(
    shutil.which("go") is None, reason="go toolchain is required"
)

QUEUE_TYPE = os.getenv("QUEUE_TYPE", "redis")
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")
DATABASE_URL = os.getenv("DATABASE_URL", "")


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


def db_reachable() -> bool:
    if not DATABASE_URL:
        return False
    try:
        import sqlalchemy

        engine = sqlalchemy.create_engine(DATABASE_URL)
        with engine.connect() as conn:
            conn.execute(sqlalchemy.text("SELECT 1"))
        engine.dispose()
        return True
    except Exception:
        return False


def _ensure_db() -> bool:
    if db_reachable():
        return True
    if shutil.which("docker") is None:
        return False
    subprocess.run(
        ["docker", "compose", "up", "-d", "postgres"],
        cwd=PROJECT_ROOT,
        capture_output=True,
    )
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        if db_reachable():
            return True
        time.sleep(0.5)
    return False


REQUIRE_DB = pytest.mark.skipif(
    not DATABASE_URL or not _ensure_db(), reason="postgresql is not available"
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
def test_db_url() -> str:
    import re

    import sqlalchemy

    base = DATABASE_URL.rsplit("/", 1)
    db_name = f"{base[1]}_test_{uuid.uuid4().hex[:8]}"
    admin_url = re.sub(r"\+asyncpg", "", base[0])
    admin_engine = sqlalchemy.create_engine(admin_url)
    with admin_engine.connect().execution_options(isolation_level="AUTOCOMMIT") as conn:
        conn.execute(sqlalchemy.text(f'CREATE DATABASE "{db_name}"'))
    admin_engine.dispose()
    test_url = f"{base[0]}/{db_name}"
    yield test_url
    admin_engine = sqlalchemy.create_engine(admin_url)
    with admin_engine.connect().execution_options(isolation_level="AUTOCOMMIT") as conn:
        conn.execute(sqlalchemy.text(f'DROP DATABASE IF EXISTS "{db_name}"'))
    admin_engine.dispose()


@pytest.fixture()
def repo(test_db_url: str) -> TaskRepository:
    repository = PostgresTaskRepository(test_db_url, auto_create=True)
    yield repository
    repository.dispose()


@pytest.fixture()
def worker(worker_binary, test_db_url: str, queue_key: str):
    log_file = os.path.join(os.path.dirname(test_db_url), "worker.log")
    args = [
        str(worker_binary),
        f"--database-url={test_db_url}",
        f"--queue={QUEUE_TYPE}",
        "--concurrency=2",
        "--log-level=debug",
    ]
    if QUEUE_TYPE == "redis":
        args += [f"--redis-url={REDIS_URL}", f"--queue-key={queue_key}"]
    proc = subprocess.Popen(
        args,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    try:
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            if proc.poll() is not None:
                raise RuntimeError(f"worker exited early")
            time.sleep(0.05)
        yield proc
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=5)


@pytest.fixture()
def client(worker, test_db_url: str, queue_key: str, repo: TaskRepository):
    queue = create_queue(QUEUE_TYPE, url=REDIS_URL, key=queue_key)
    app.dependency_overrides[get_task_repository] = lambda: repo
    app.dependency_overrides[get_task_queue] = lambda: queue
    with TestClient(app) as c:
        yield c
    app.dependency_overrides.clear()
    queue.close()


@pytest.fixture()
def api_client(repo: TaskRepository):
    queue = create_queue("memory")
    app.dependency_overrides[get_task_repository] = lambda: repo
    app.dependency_overrides[get_task_queue] = lambda: queue
    with TestClient(app) as c:
        yield c
    app.dependency_overrides.clear()
