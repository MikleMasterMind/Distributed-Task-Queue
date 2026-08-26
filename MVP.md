# Distributed Task Queue — MVP

## 1. Project Goal

Create a minimal distributed system for enqueueing background tasks and executing them with separate Go workers.

The MVP must support the following scenario:

```text
Client
  │
  │ POST /tasks
  ▼
FastAPI
  │
  ▼
Redis Queue
  │
  ▼
Go Worker
  │
  ▼
Task execution
  │
  ▼
PostgreSQL
```

A user creates a task via HTTP API, the task enters the queue, a Go worker picks it up and executes it, and the result and state are saved in PostgreSQL.

---

## 2. Components

The MVP consists of four main services and infrastructure.

### Python API

Responsible for:

* HTTP API;
* task creation;
* fetching task state;
* storing task metadata.

Technologies:

* Python 3.12+
* FastAPI
* SQLAlchemy
* PostgreSQL
* Pydantic

### Redis

Used as a message queue.

Responsible for:

* storing pending tasks;
* delivering tasks to workers.

At the MVP stage, building a custom message broker is not required.

### Go Worker

Responsible for:

* connecting to Redis;
* fetching tasks;
* executing tasks;
* updating task status;
* graceful shutdown.

Technologies:

* Go
* goroutines
* context
* Redis client

### PostgreSQL

Stores:

* tasks;
* their status;
* result;
* error;
* timestamps.

---

# 3. Task Model

Each task should have approximately this model:

```text
Task
├── id
├── type
├── payload
├── status
├── result
├── error
├── created_at
├── started_at
└── finished_at
```

Each task is executed **only once**: `started_at`/`finished_at` is the single pair of start and finish timestamps.

### Status

The MVP must support the following states:

```text
PENDING
RUNNING
SUCCESS
FAILED
```

Lifecycle:

```text
PENDING
   │
   ▼
RUNNING
   │
   ├──────► SUCCESS
   │
   └──────► FAILED
```

---

# 4. HTTP API

## Create Task

```http
POST /tasks
```

Request:

```json
{
  "type": "echo",
  "payload": {
    "message": "hello"
  }
}
```

Response:

```json
{
  "id": "uuid",
  "status": "PENDING"
}
```

---

## Get Task

```http
GET /tasks/{task_id}
```

Response:

```json
{
  "id": "uuid",
  "type": "echo",
  "status": "SUCCESS",
  "result": {
    "message": "hello"
  },
  "error": null,
  "created_at": "...",
  "started_at": "...",
  "finished_at": "..."
}
```

---

## Delete Task

Only tasks that haven't started yet (status `PENDING`) can be deleted.

```http
DELETE /tasks/{task_id}
```

Success:

```text
204 No Content
```

Errors:

```text
404 — task not found
409 — task already started executing (status != PENDING)
```

---

# 5. Task Types

For the MVP, it's sufficient to implement 2–3 simple task types.

For example:

### Echo

Returns the received payload.

```json
{
  "type": "echo",
  "payload": {
    "message": "hello"
  }
}
```

Result:

```json
{
  "message": "hello"
}
```

### Sleep

Used for testing long-running tasks.

```json
{
  "type": "sleep",
  "payload": {
    "seconds": 5
  }
}
```

The worker should wait 5 seconds and complete the task.

### Fibonacci

Allows testing CPU-bound task execution.

```json
{
  "type": "fibonacci",
  "payload": {
    "n": 40
  }
}
```

---

# 6. Redis Queue

When creating a task:

```text
FastAPI
   │
   ├── create task in PostgreSQL
   │
   └── push task ID to Redis
                  │
                  ▼
               Queue
```

It's preferable to store only the task ID in Redis, not the entire task:

```text
queue:
    task_id_1
    task_id_2
    task_id_3
```

The worker receives the ID:

```text
Redis
  │
  ▼
task_id
  │
  ▼
PostgreSQL
  │
  ▼
Task
```

This allows PostgreSQL to remain the source of truth for task state.

---

# 7. Go Worker

The worker runs separately:

```bash
go-worker
```

After startup, it:

1. connects to Redis;
2. checks the PostgreSQL connection;
3. starts waiting for tasks;
4. receives a task ID;
5. fetches the task from PostgreSQL;
6. changes status to `RUNNING`;
7. executes the task;
8. saves the result;
9. changes status to `SUCCESS` or `FAILED`.

Example:

```text
Redis
  │
  │ task_id
  ▼
Worker
  │
  │ SELECT task
  ▼
PostgreSQL
  │
  ▼
RUNNING
  │
  ▼
Execute
  │
  ├──────► SUCCESS
  │
  └──────► FAILED
```

---

# 8. Worker Pool

Even in the MVP, the worker must support multiple concurrent tasks.

For example:

```text
Worker
  │
  ├── goroutine → Task 1
  ├── goroutine → Task 2
  ├── goroutine → Task 3
  └── goroutine → Task 4
```

The number of concurrent tasks should be configurable:

```yaml
worker:
  concurrency: 4
```

For example:

```bash
go-worker --concurrency=4
```

If 20 tasks arrive simultaneously, the worker only executes 4:

```text
20 tasks
   │
   ▼
┌────────────────────┐
│ Worker             │
│                    │
│ Task 1  RUNNING    │
│ Task 2  RUNNING    │
│ Task 3  RUNNING    │
│ Task 4  RUNNING    │
│                    │
│ Task 5+ PENDING    │
└────────────────────┘
```

---

# 9. Errors

If a task fails:

```text
RUNNING
   │
   ▼
FAILED
```

The error must be saved in PostgreSQL:

```json
{
  "status": "FAILED",
  "error": "division by zero"
}
```

The worker must not crash due to a single task failure.

For example:

```text
Task 1 → FAILED
Task 2 → SUCCESS
Task 3 → SUCCESS
```

The worker continues running.

---

# 10. Graceful Shutdown

Upon receiving `SIGTERM` or `SIGINT`, the worker must:

1. stop accepting new tasks;
2. wait for running tasks to finish;
3. close connections;
4. exit.

For example:

```text
SIGTERM
   │
   ▼
Stop accepting tasks
   │
   ▼
Wait for running tasks
   │
   ▼
Close Redis
   │
   ▼
Exit
```

This is a mandatory part of the MVP because the worker is a long-running service.

---

# 11. Configuration

Configuration must not be hardcoded.

Minimum set:

```yaml
server:
  host: 0.0.0.0
  port: 8000

postgres:
  dsn: ...

redis:
  address: ...

worker:
  concurrency: 4
```

For production-like setups, secrets should be passed via environment variables.

---

# 12. Docker Compose

The entire MVP should start with a single command:

```bash
docker compose up
```

Minimum containers:

```text
┌─────────────────────────────────┐
│         Docker Compose          │
│                                 │
│  ┌─────────┐                    │
│  │ FastAPI │                    │
│  └─────────┘                    │
│                                 │
│  ┌─────────┐                    │
│  │ Worker  │                    │
│  └─────────┘                    │
│                                 │
│  ┌─────────┐  ┌──────────────┐  │
│  │ Redis   │  │ PostgreSQL   │  │
│  └─────────┘  └──────────────┘  │
└─────────────────────────────────┘
```

---

# 13. Minimum Tests

### Python

Verify:

* task creation;
* task fetching;
* payload validation;
* handling of non-existent task IDs.

### Go

Verify:

* handling of each task type;
* successful execution;
* execution failure;
* worker pool;
* graceful shutdown.

### Integration Test

Main scenario:

```text
POST /tasks
      │
      ▼
PostgreSQL
      │
      ▼
Redis
      │
      ▼
Go Worker
      │
      ▼
PostgreSQL
      │
      ▼
GET /tasks/{id}
```

Expected result:

```text
PENDING → RUNNING → SUCCESS
```

---

# 14. What NOT to Include in the MVP

The following features are better left for future versions:

* retries;
* priority queues;
* scheduled tasks;
* cron;
* heartbeat;
* worker discovery;
* visibility timeout;
* dead-letter queue;
* distributed locks;
* authentication;
* web dashboard;
* Prometheus;
* Grafana;
* OpenTelemetry;
* custom message broker;
* Kubernetes.

They are interesting but would significantly increase the project scope.

---

# 15. Remaining Work

## P0 — Blocks Definition of Done

- [ ] Add FastAPI container to `docker-compose.yml`
- [ ] Add Go Worker container to `docker-compose.yml`
- [ ] Verify `docker compose up` starts all 4 services end-to-end

## P1 — Spec Compliance

- [ ] Make FastAPI `host` and `port` configurable via env vars (`HOST`, `PORT`)
- [ ] Add PostgreSQL connection check on Go worker startup (§7, step 2)

## P2 — Test Coverage

- [ ] Add Go test for graceful shutdown (§13 requires it)

## P3 — Documentation

- [ ] Remove or correct `DirQueue` reference in `AGENTS.md` (does not exist in codebase)

---

# 16. Definition of Done

The MVP is considered complete when the following scenario can be executed.

### Step 1

Start the system:

```bash
docker compose up
```

### Step 2

Create a task:

```http
POST /tasks
```

### Step 3

Receive:

```text
task_id = 123
status = PENDING
```

### Step 4

The Go Worker automatically picks up the task:

```text
PENDING
   ↓
RUNNING
```

### Step 5

The worker executes the task:

```text
RUNNING
   ↓
SUCCESS
```

### Step 6

Fetch the result:

```http
GET /tasks/123
```

and see:

```json
{
  "id": "123",
  "status": "SUCCESS",
  "result": {
    "message": "hello"
  }
}
```

### Step 7

Start multiple workers:

```text
Worker 1
Worker 2
Worker 3
```

and verify that tasks are distributed among them.

---

# Final MVP Architecture

```text
                       HTTP
                        │
                        ▼
                ┌──────────────┐
                │   FastAPI    │
                │    Python    │
                └──────┬───────┘
                       │
              ┌────────┴────────┐
              │                 │
              ▼                 ▼
       ┌─────────────┐   ┌─────────────┐
       │ PostgreSQL  │   │    Redis    │
       │             │   │    Queue    │
       │ Task state  │   │             │
       └─────────────┘   └──────┬──────┘
                                │
                       ┌────────┼────────┐
                       │        │        │
                       ▼        ▼        ▼
                   ┌───────┐ ┌───────┐ ┌───────┐
                   │ Go W1 │ │ Go W2 │ │ Go W3 │
                   └───────┘ └───────┘ └───────┘
```

## MVP Core Goal

Not the number of features, but proof that the full pipeline works:

**task creation → persistence → queue → distributed worker → execution → result persistence → result retrieval via API.**

After that, it makes sense to move on to **V2: retries + visibility timeout + heartbeat + DLQ** — that's where the most interesting part of distributed systems begins.
