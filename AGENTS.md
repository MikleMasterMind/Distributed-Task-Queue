# AGENTS.md

## Repo state

V1 is in progress: the FastAPI task API is implemented, but the distributed parts are not. `MVP.md` describes the target architecture (`FastAPI → Redis → Go workers → PostgreSQL`); current code is only the Python API layer, persisted to JSON files via `FileTaskRepository`. There is **no** `worker/`, `docker-compose.yml`, Redis, PostgreSQL, or CI yet — do not assume they exist.

- `MVP.md` — authoritative V1 spec (task model, statuses, HTTP API, scope). Read before writing feature code.
- `ROADMAP.md` — V2+ plan (retry, visibility timeout, DLQ, heartbeat, etc.). Its "already implemented" list lags reality — trust the code, not it.
- `COMMIT-RULES.MD` — binding commit convention (see below).
- `README.md` — current run instructions and API usage (Russian).

## Layout & packaging gotchas

- `src/` holds top-level modules `app.py`, `config.py`, `main.py` (imported as `from app import app`) plus packages `api/` and `repository/`.
- New top-level modules must be added to `py-modules` in `pyproject.toml` or they won't be importable; `api*`/`repository*` packages are auto-discovered.
- Entry point `distributed-task-queue = "main:main"` runs uvicorn on `0.0.0.0:8000` (hardcoded, not configurable).
- Python pinned to 3.14 in `.python-version` (project requires >=3.12).

## Storage

- Persistence is `FileTaskRepository` (`src/repository/file.py`): one JSON file per task in `data/tasks/` (gitignored). `DATA_DIR` env var overrides the base directory.
- It implements the `TaskRepository` ABC (`src/repository/base.py`); a future Postgres implementation plugs in there. The API obtains the repo via the `get_task_repository` FastAPI dependency.
- Tests never touch real storage: they override with `app.dependency_overrides[get_task_repository] = lambda: FileTaskRepository(tmp_path)`.

## Domain rules to respect

- Task types restricted to `echo | sleep | fibonacci` with strict per-type payload validation (`TaskCreate.validate_payload`).
- Statuses: `PENDING → RUNNING → SUCCESS | FAILED`. DELETE only allowed on `PENDING` (404 if missing, 409 otherwise).
- `GET /tasks` pagination: cursor `next-token` is base64 of `created_at|id`; tasks sorted by `(created_at, id)` ascending.

## MVP scope discipline

`MVP.md` §14 defers: retries, priority/multiple queues, scheduled/cron, heartbeat, worker discovery, visibility timeout, DLQ, distributed locks, auth, dashboard, metrics/tracing, custom broker, Kubernetes. Do not add these in V1.

## Git rules

- Never run git-mutating commands (`add`, `commit`, `push`, `rebase`, `merge`, etc.) unless the user explicitly asks.
- Commits: Conventional Commits `<type>(<scope>): <description>`, English, imperative mood (e.g. `feat(worker): add graceful shutdown`). Types: `feat`, `fix`, `refactor`, `test`, `docs`, `perf`, `build`, `ci`, `chore`, `revert`. Scopes: `api`, `worker`, `queue`, `scheduler`, `db`, `config`, `observability`, `auth`, `docs`, `ci`. One logical change per commit. Breaking: `!` in subject + `BREAKING CHANGE:` in body. (Full rules in COMMIT-RULES.MD.)

## Conventions

- Docs in Russian; commit subjects/bodies in English.
- No comments in code — anywhere.
- Use uv for everything (no pip/venv directly).

## Commands

- Install deps: `uv sync --extra dev`
- Run API: `uv run distributed-task-queue` (http://0.0.0.0:8000, Swagger at `/docs`)
- Run all tests: `uv run pytest`