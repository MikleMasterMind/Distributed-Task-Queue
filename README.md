# Distributed-Task-Queue

Distributed task queue. HTTP API in Python (FastAPI), Redis queue, and Go worker for task execution.

Architecture: `FastAPI → Redis Queue → Go Workers → PostgreSQL`. When a task is created, FastAPI saves its state to PostgreSQL and places the `task_id` into a Redis queue. The Go worker receives the ID from the queue (blocking `BRPOP`), fetches the task, executes it, and writes the result back to PostgreSQL. Task state is stored in a `tasks` table.

## Setup

Requires [uv](https://docs.astral.sh/uv/), Python 3.12+, Go 1.26+, Redis, PostgreSQL (or Docker).

```bash
cp .env.example .env          # configure DATABASE_URL, REDIS_URL, etc.
docker compose up -d redis postgres  # start Redis and PostgreSQL
uv sync --extra dev
```

Starting the API:

```bash
uv run distributed-task-queue
```

Starting the Go worker (in a separate terminal):

```bash
cd worker && go build -o worker ./cmd/worker && ./worker
```

The worker reads configuration from `.env` in the project root (falls back to `worker/.env`). All settings can also be overridden via flags.

The API will be available at `http://0.0.0.0:8000` (Swagger UI at `/docs`). The worker fetches tasks from Redis and executes them, writing results to PostgreSQL.

Configuration is read from `.env` (see `.env.example`):

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | — | PostgreSQL connection URL (required) |
| `DB_AUTO_CREATE` | `true` | Auto-create tables on startup |
| `STORE_TYPE` | `postgres` | Go worker store backend: `postgres` |
| `QUEUE_TYPE` | `redis` | API queue backend: `redis`, `memory` |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection URL |
| `QUEUE_KEY` | `dtq:tasks` | Redis list key for the task queue |
| `LOG_LEVEL` | `info` | API log level (`debug`, `info`, `warning`, `error`, `critical`) |

## Request Examples

Creating a task:

```bash
curl -X POST localhost:8000/tasks \
  -H 'Content-Type: application/json' \
  -d '{"type": "echo", "payload": {"message": "hello"}}'
```

Getting a task:

```bash
curl localhost:8000/tasks/<task_id>
```

Deleting a task (only if it hasn't started yet, `PENDING`):

```bash
curl -X DELETE localhost:8000/tasks/<task_id>
```

Success — `204 No Content`. Errors: `404` (task not found), `409` (task already started executing).

Listing tasks with pagination and filters:

```bash
curl 'localhost:8000/tasks?limit=20&status=PENDING&type=echo'
```

Response:

```json
{
  "items": [ ... ],
  "next-token": "...",
  "limit": 20
}
```

Parameters: `limit` (1–100, default 20), `next-token` (pagination), `status`, `type`, `created_after`/`created_before`, `started_after`/`started_before` (by start date), `finished_after`/`finished_before` (by finish date). Tasks are sorted by creation date (oldest first).

Supported task types: `echo` (`message`), `sleep` (`seconds`), `fibonacci` (`n`).

## Go Worker

The worker receives task IDs from Redis (`BRPOP`) and executes them in parallel (number of goroutines is configured via configuration). One task's failure doesn't affect others.

```bash
cd worker && go build -o worker ./cmd/worker && ./worker
```

Parameters (flag `--name` or environment variable from `.env`):

- `--database-url` (env `DATABASE_URL`) — PostgreSQL connection URL;
- `--db-auto-create` (env `DB_AUTO_CREATE`) — auto-create tables, default `true`;
- `--store` (env `STORE_TYPE`) — store backend: `postgres`;
- `--queue` (env `QUEUE_TYPE`) — queue backend: `redis`;
- `--redis-url` (env `REDIS_URL`) — Redis connection URL, default `redis://localhost:6379/0`;
- `--queue-key` (env `QUEUE_KEY`) — queue key in Redis, default `dtq:tasks`;
- `--concurrency` (env `CONCURRENCY`) — number of concurrent tasks, default `4`;
- `--log-level` (env `LOG_LEVEL`) — log level (`debug`, `info`, `warn`, `error`), default `info`.

The queue backend is pluggable: to add a new one, implement the `Queue` interface (`Pop(ctx) (string, error)`) and register it in `worker/internal/queue/factory.go`.

The worker shuts down on `SIGINT`/`SIGTERM`: stops fetching new tasks, waits for current ones to finish, and exits.

Tests:

```bash
cd worker && go test ./...
```

## Tests

```bash
uv run pytest
```

Integration tests (`tests/test_integration.py`) run the full pipeline: the API creates a task via HTTP, a real Go worker (automatically built to a temporary directory when tests run) executes it through the Redis queue, and the result is read back through the API. Requires Go installed, Redis and PostgreSQL available; if Redis/PostgreSQL aren't running, the tests automatically try to start them via `docker compose up -d redis postgres`. If unavailable, integration tests are skipped.
