import shutil
import subprocess
import time
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

from app import app
from repository import FileTaskRepository, get_task_repository

PROJECT_ROOT = Path(__file__).resolve().parents[1]
WORKER_DIR = PROJECT_ROOT / "worker"

REQUIRE_GO = pytest.mark.skipif(
    shutil.which("go") is None, reason="go toolchain is required"
)


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
def worker(worker_binary: Path, tmp_path: Path):
    log_file = tmp_path / "worker.log"
    proc = subprocess.Popen(
        [
            str(worker_binary),
            f"--tasks-dir={tmp_path}",
            "--poll-interval=50ms",
            "--concurrency=2",
            "--log-level=debug",
        ],
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
def client(tmp_path: Path, worker: subprocess.Popen):
    repo = FileTaskRepository(tmp_path)
    app.dependency_overrides[get_task_repository] = lambda: repo
    with TestClient(app) as c:
        yield c
    app.dependency_overrides.clear()
