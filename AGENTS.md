# AGENTS.md

## Repo state

V1 is in progress: the FastAPI task API, Redis queue and Go worker are implemented with PostgreSQL storage. `MVP.md` describes the target architecture (`FastAPI → Redis → Go workers → PostgreSQL`); PostgreSQL is implemented via SQLAlchemy ORM (Python) and GORM (Go). The queue stores only task IDs; task state lives in PostgreSQL.

- `MVP.md` — authoritative V1 spec (task model, statuses, HTTP API, scope). Read before writing feature code.
- `ROADMAP.md` — V2+ plan (retry, visibility timeout, DLQ, heartbeat, etc.). Its "already implemented" list lags reality — trust the code, not it.
- `COMMIT-RULES.MD` — binding commit convention (see below).
- `README.md` — current run instructions and API usage (English).

## Layout & packaging gotchas

- `src/` holds top-level modules `app.py`, `config.py`, `main.py` (imported as `from app import app`) plus packages `api/`, `db/`, `repository/` and `taskqueue/`.
- New top-level modules must be added to `py-modules` in `pyproject.toml`; packages are auto-discovered via `include = ["api*", "db*", "repository*", "taskqueue*"]`.
- The queue package is **`taskqueue`, not `queue`** — `queue` collides with the Python stdlib module that `redis` imports.
- Entry point `distributed-task-queue = "main:main"` runs uvicorn on `0.0.0.0:8000` (hardcoded, not configurable).
- Python pinned to 3.14 in `.python-version` (project requires >=3.12).

## Storage & queue

- Persistence is `PostgresTaskRepository` (`src/db/postgres.py`): SQLAlchemy ORM with asyncpg driver. `DATABASE_URL` env var (required) sets the connection; `DB_AUTO_CREATE` controls table creation.
- It implements the `TaskRepository` ABC (`src/repository/base.py`). The API obtains the repo via the `get_task_repository` FastAPI dependency.
- Task queue is `TaskQueue` ABC (`src/taskqueue/base.py`); backends: `RedisTaskQueue` (`LPUSH`/`LREM` on `QUEUE_KEY`) and `InMemoryQueue`. Selected via `create_queue()` (`src/taskqueue/factory.py`) by `QUEUE_TYPE` env (`redis`|`memory`); obtained via the `get_task_queue` FastAPI dependency; created in the FastAPI lifespan. New backends plug into the factory.
- Go worker consumes IDs via a pluggable `Queue` interface (`worker/internal/queue/queue.go`): `RedisQueue` (BRPOP) and `DirQueue` (polling, fallback). Backend selected by `--queue` / `QUEUE_TYPE` (default `redis`) through `queue.New(QueueConfig{...})`.
- Go worker reads/writes task state via `PostgresStore` (`worker/internal/store/postgres.go`): GORM with lib/pq driver. `DATABASE_URL` and `DB_AUTO_CREATE` flags control connection.
- Config on both sides reads `.env` (root `.env.example`). Env vars: `DATABASE_URL`, `DB_AUTO_CREATE`, `QUEUE_TYPE`, `REDIS_URL`, `QUEUE_KEY`, `LOG_LEVEL` (Python + Go), plus worker-only `CONCURRENCY`, `POLL_INTERVAL_MS`.
- Tests override `app.dependency_overrides[get_task_repository]` and `app.dependency_overrides[get_task_queue]`. Pure-API tests use `api_client` (in-memory queue); integration tests use `client` (real Redis + real PostgreSQL + real worker, skip if unavailable). Test `QUEUE_TYPE` comes from env — the worker fixture and client build the backend through config, not hardcoding.

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

- Docs in English; commit subjects/bodies in English.
- No comments in code — anywhere.
- Use uv for everything (no pip/venv directly).

## Commands

- Install deps: `uv sync --extra dev`
- Start Redis: `docker compose up -d redis`
- Start PostgreSQL: `docker compose up -d postgres`
- Run API: `DATABASE_URL=postgresql+asyncpg://dtq:dtq@localhost:5432/dtq uv run distributed-task-queue` (http://0.0.0.0:8000, Swagger at `/docs`)
- Build worker: `cd worker && go build -o worker ./cmd/worker`
- Run Go tests: `cd worker && go test ./...`
- Run all tests: `uv run pytest`