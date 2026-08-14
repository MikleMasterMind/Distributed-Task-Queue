# AGENTS.md

## Repo state

This repository currently contains **only planning docs — there is no source code yet** (no `api/`, `worker/`, `docker-compose.yml`, CI, or tests). Do not search for code; it does not exist. All work right now is implementing the plan.

- `MVP.md` — authoritative spec for V1 (architecture, task model, statuses, HTTP API, worker pool, config, tests, Definition of Done). Read it before writing code.
- `ROADMAP.md` — V2+ plan (retry, visibility timeout, DLQ, heartbeat, priority queues, scheduling, etc.).
- `COMMIT-RULES.MD` — binding commit convention (see below).
- `README.md` — empty placeholder.

## Planned architecture (from MVP.md)

```
FastAPI (Python) → Redis Queue → Go Workers → PostgreSQL
```

- **API** (`POST /tasks`, `GET /tasks/{id}`): Python 3.12+, FastAPI, SQLAlchemy, Pydantic.
- **Queue**: Redis holds only task IDs (single list); PostgreSQL is the source of truth for task state.
- **Worker**: Go binary `go-worker` (single binary, no separate control plane), goroutine worker pool with configurable concurrency, graceful shutdown on SIGTERM/SIGINT.
- **Statuses**: `PENDING → RUNNING → SUCCESS | FAILED`.
- Everything must boot with `docker compose up`.

## MVP scope discipline

`MVP.md` §14 explicitly defers: retries, priority/multiple queues, scheduled/cron tasks, heartbeat, worker discovery, visibility timeout, DLQ, distributed locks, auth, dashboard, metrics/tracing, custom broker, Kubernetes. Do not introduce these in V1 even if they seem easy.

## Git restrictions

- **Never** run `git add`, `git commit`, `git push`, `git rebase`, `git merge`, or any other git-mutating commands. Commit only when the user explicitly asks. (For the expected format, see COMMIT-RULES.MD below.)

## Commit rules (from COMMIT-RULES.MD)

- Conventional Commits: `<type>(<scope>): <description>`.
- Description in **English**, imperative mood: `feat(worker): add graceful shutdown` (not "added").
- Allowed types: `feat`, `fix`, `refactor`, `test`, `docs`, `perf`, `build`, `ci`, `chore`, `revert`.
- Scopes: `api`, `worker`, `queue`, `scheduler`, `db`, `config`, `observability`, `auth`, `docs`, `ci`.
- One commit = one logical change; split mixed changes into separate commits.
- Breaking changes: `!` in subject + `BREAKING CHANGE:` in body.

## Conventions

- Project docs are written in **Russian**; commit subjects/bodies are in **English**.
- No comments in code — anywhere.
- Python tooling: use **uv** for dependency management, virtualenvs, and running scripts (no pip/venv directly).

## Commands

- Install deps: `uv sync --extra dev`
- Run API: `uv run distributed-task-queue` (uvicorn on `0.0.0.0:8000`)
- Run tests: `uv run pytest`
