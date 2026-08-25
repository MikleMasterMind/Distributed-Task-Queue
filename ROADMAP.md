# Distributed Task Queue — Roadmap

## Goal

After completing the MVP, gradually transform the project from a simple task queue into a full-featured distributed background task execution system.

Expansion should happen in stages. Each stage should add a separate architectural capability that can be tested and demonstrated.

---

# V1 — MVP

**Goal:** basic task execution.

Already implemented:

* FastAPI;
* PostgreSQL;
* Redis Queue;
* Go Worker;
* Worker Pool;
* multiple task types;
* statuses `PENDING`, `RUNNING`, `SUCCESS`, `FAILED`;
* error handling;
* graceful shutdown;
* Docker Compose;
* unit and integration tests.

Architecture:

```text
FastAPI
   │
   ├── PostgreSQL
   │
   └── Redis
         │
         ├── Go Worker
         ├── Go Worker
         └── Go Worker
```

---

# V2 — Execution Reliability

**Goal:** the system should work correctly when workers crash.

This is the first stage where real distributed-systems problems appear.

## 2.1 Retry

Add automatic task retry.

For example:

```text
Task
 │
 ├── attempt 1 → FAILED
 │
 ├── wait 1 sec
 │
 ├── attempt 2 → FAILED
 │
 ├── wait 2 sec
 │
 └── attempt 3 → SUCCESS
```

Add:

```yaml
retry:
  max_attempts: 3
  backoff: exponential
```

Support:

* `max_attempts`;
* exponential backoff;
* configurable delay;
* retryable/non-retryable errors.

---

## 2.2 Visibility Timeout

Problem:

```text
Worker A
   │
   └── received Task 123
          │
          X CRASH
```

The task must not be lost.

When received, the task becomes temporarily invisible to other workers.

If the worker doesn't complete it within a specified time:

```text
visibility timeout
        │
        ▼
Task → QUEUED
```

---

## 2.3 Dead Letter Queue

If a task keeps failing:

```text
attempt 1 → FAILED
attempt 2 → FAILED
attempt 3 → FAILED
attempt 4 → FAILED
```

It should go to:

```text
Dead Letter Queue
```

Add API:

```http
GET /dead-letter/tasks
POST /dead-letter/tasks/{id}/retry
```

---

## 2.4 Task Timeout

Add execution time limit:

```yaml
task:
  timeout: 60s
```

If a task runs too long:

```text
RUNNING
   │
   │ 60 sec
   ▼
TIMEOUT
```

---

# V3 — Worker Management

**Goal:** turn a set of workers into a managed cluster.

## 3.1 Worker Registration

On startup, the worker registers:

```text
Worker
   │
   ▼
Control Plane
```

Information:

```json
{
  "worker_id": "worker-123",
  "hostname": "...",
  "version": "1.0.0",
  "concurrency": 8,
  "queues": ["default"]
}
```

---

## 3.2 Heartbeat

The worker regularly reports:

```text
Worker
   │
   ├── heartbeat
   ├── heartbeat
   ├── heartbeat
   └── heartbeat
```

If heartbeats stop coming:

```text
Worker
   │
   X
   │
   ▼
DEAD
```

This allows detecting worker crashes independently of task timeout.

---

## 3.3 Worker Status

Add states:

```text
STARTING
IDLE
BUSY
DRAINING
DEAD
```

For example:

```http
GET /workers
```

Response:

```json
[
  {
    "id": "worker-1",
    "status": "BUSY",
    "running_tasks": 4,
    "concurrency": 8
  }
]
```

---

## 3.4 Worker Graceful Drain

Add the ability to take a worker out of service:

```http
POST /workers/{id}/drain
```

Worker:

```text
DRAINING
   │
   ├── stops accepting new tasks
   │
   ├── finishes current ones
   │
   ▼
STOPPED
```

This allows safely updating workers.

---

# V4 — Advanced Queues

**Goal:** make the queue more feature-rich.

## 4.1 Priority Queue

Add priority:

```text
priority = 1
priority = 10
priority = 100
```

The worker first receives high-priority tasks.

---

## 4.2 Multiple Queues

For example:

```text
default
high-priority
cpu
io
ml
```

A worker can subscribe to specific queues:

```bash
go-worker --queues=default,high-priority
```

Another:

```bash
go-worker --queues=cpu
```

---

## 4.3 Routing

Add routing rules:

```text
task.type = image_processing
        ↓
image queue

task.type = ml_inference
        ↓
ml queue
```

This way CPU-heavy tasks won't interfere with regular tasks.

---

# V5 — Scheduling

**Goal:** add delayed and periodic tasks.

## 5.1 Delayed Tasks

```http
POST /tasks
```

```json
{
  "type": "send_email",
  "execute_at": "2026-08-15T10:00:00Z"
}
```

The task must not appear in the regular queue until the specified time.

---

## 5.2 Periodic Tasks

For example:

```yaml
schedule:
  name: cleanup
  task: cleanup_old_files
  cron: "0 * * * *"
```

The scheduler creates tasks automatically:

```text
Scheduler
   │
   ├── 10:00 → cleanup
   ├── 11:00 → cleanup
   ├── 12:00 → cleanup
   └── ...
```

---

# V6 — Idempotency and Exactly-once Semantics

**Goal:** explore the problem of duplicate task execution.

In a distributed system, the following situation is possible:

```text
Worker
  │
  ├── executes task
  │
  ├── task completes
  │
  X worker crash
  │
  ▼
Queue considers task incomplete
  │
  ▼
Another worker re-executes it
```

A single task can be executed twice.

---

## 6.1 Idempotency Key

Add:

```json
{
  "idempotency_key": "payment-123"
}
```

Repeated enqueueing:

```text
payment-123
payment-123
payment-123
```

should result in a single logical execution.

---

## 6.2 Document Delivery Semantics

Explicitly define:

```text
At-most-once
At-least-once
Exactly-once
```

The project should aim to implement:

> **At-least-once delivery + idempotent task execution**

and separately explain why full exactly-once execution is a hard problem in distributed systems.

---

# V7 — Observability

**Goal:** achieve full system visibility.

## 7.1 Metrics

Use Prometheus.

Metrics:

```text
tasks_created_total
tasks_completed_total
tasks_failed_total
tasks_retried_total

task_execution_duration_seconds

queue_size
worker_count
worker_busy_count
```

---

## 7.2 Grafana

Create a dashboard:

```text
┌──────────────────────────────────────┐
│ Tasks/sec            124             │
│ Success rate         99.8%           │
│ Queue size            42             │
│ Active workers         8             │
├──────────────────────────────────────┤
│ Task latency                         │
│              ╱╲                      │
│        ╱╲   ╱  ╲                     │
│   ╱╲  ╱  ╲_╱    ╲                    │
└──────────────────────────────────────┘
```

---

## 7.3 Distributed Tracing

Add OpenTelemetry.

For example:

```text
HTTP request
     │
     ▼
FastAPI
     │
     ▼
Redis
     │
     ▼
Go Worker
     │
     ▼
Task execution
```

The trace should show the duration of each stage.

---

# V8 — Control Plane / Worker Plane

**Goal:** split the architecture into two logical parts.

```text
             Control Plane
                  │
        ┌─────────┴─────────┐
        │                   │
      API               Scheduler
        │                   │
        └─────────┬─────────┘
                  │
                  ▼
                Queue
                  │
       ┌──────────┼──────────┐
       ▼          ▼          ▼
   Worker 1   Worker 2   Worker 3
       │          │          │
       └──────────┼──────────┘
                  │
             Worker Plane
```

The Control Plane handles management.

The Worker Plane handles execution.

---

# V9 — Horizontal Scaling

**Goal:** test system scalability.

Run:

```text
1 worker
```

and measure throughput.

Then:

```text
2 workers
4 workers
8 workers
16 workers
```

Compare:

```text
workers | tasks/sec | p95 latency
--------|-----------|------------
1       | 50        | 120 ms
2       | 95        | 80 ms
4       | 180       | 45 ms
8       | 340       | 30 ms
```

The goal is not just to increase the number of workers, but to identify:

* bottleneck;
* CPU bottleneck;
* Redis bottleneck;
* PostgreSQL bottleneck;
* network bottleneck.

---

# V10 — Load Testing

Add load tests.

For example:

```text
10 000 tasks
100 workers
```

Verify:

* throughput;
* latency;
* failure rate;
* queue growth;
* memory consumption;
* CPU usage.

Can use:

* Locust;
* k6;
* custom Go load generator.

---

# V11 — Custom Message Broker

This is an **optional advanced stage**.

After the Redis version works, replace Redis with a custom implementation.

For example:

```text
Python API
     │
     ▼
┌───────────────────┐
│   Go Message      │
│      Broker       │
│                   │
│ Topics            │
│ Partitions        │
│ Consumer Groups   │
│ WAL               │
└─────────┬─────────┘
          │
          ▼
       Workers
```

Can implement:

* TCP protocol;
* message framing;
* append-only log;
* persistence;
* consumer offsets;
* partitions;
* consumer groups;
* batching.

This is already a separate large project and **not a mandatory part of Task Queue**.

---

# V12 — Distributed Broker

If you want to dive even deeper into distributed systems:

```text
             Broker Cluster

       ┌──────────┐
       │ Broker 1 │
       └────┬─────┘
            │
      ┌─────┴─────┐
      ▼           ▼
 Broker 2      Broker 3
```

Explore:

* replication;
* leader election;
* failure detection;
* quorum;
* consistency;
* recovery.

This stage is best treated as a separate experimental project.

---

# Implementation Priority

Don't implement all stages sequentially.

Optimal path:

```text
MVP
 │
 ▼
V2 Reliability
 │
 ├── Retry
 ├── Visibility Timeout
 ├── DLQ
 └── Timeout
 │
 ▼
V3 Worker Management
 │
 ├── Registration
 ├── Heartbeat
 └── Drain
 │
 ▼
V4 Advanced Queues
 │
 ├── Priority
 ├── Multiple Queues
 └── Routing
 │
 ▼
V5 Scheduling
 │
 ├── Delayed Tasks
 └── Periodic Tasks
 │
 ▼
V6 Idempotency
 │
 └── At-least-once delivery
 │
 ▼
V7 Observability
 │
 ├── Prometheus
 ├── Grafana
 └── OpenTelemetry
 │
 ▼
V8 Scaling
 │
 └── Load Testing
```

After this, the project will be substantial enough for a resume.

---

# What I Would NOT Do

Don't turn the project into an attempt to create Kubernetes + Kafka + Celery simultaneously.

Especially avoid adding unnecessarily:

* Kubernetes;
* service mesh;
* dozens of microservices;
* custom database;
* custom consensus protocol;
* complex UI dashboard.

Better to have:

```text
10 well-implemented mechanisms
```

than:

```text
50 technologies connected by Docker Compose.
```

---

# Final Version

If brought to a reasonable production-like state, the final architecture might look like this:

```text
                         ┌───────────────┐
                         │    Client     │
                         └───────┬───────┘
                                 │
                                 ▼
                         ┌───────────────┐
                         │   FastAPI     │
                         │ Control Plane │
                         └───────┬───────┘
                                 │
                  ┌──────────────┼──────────────┐
                  ▼              ▼              ▼
             PostgreSQL       Scheduler      Metrics
                  │
                  ▼
             ┌─────────┐
             │  Queue  │
             └────┬────┘
                  │
        ┌─────────┼─────────┐
        ▼         ▼         ▼
     Worker 1  Worker 2  Worker 3
        │         │         │
        └─────────┼─────────┘
                  │
                  ▼
             Task Results

       ┌─────────────────────────┐
       │ Prometheus + Grafana    │
       │ OpenTelemetry           │
       └─────────────────────────┘
```

The main idea of the roadmap is that **each new version should solve a specific distributed systems problem**:

```text
MVP
 ↓
"How to execute a task?"

Reliability
 ↓
"What happens if a worker crashes?"

Worker Management
 ↓
"How to manage multiple workers?"

Advanced Queues
 ↓
"How to route different workload types?"

Scheduling
 ↓
"How to execute tasks in the future?"

Idempotency
 ↓
"What to do about duplicate execution?"

Observability
 ↓
"How to understand what's happening in the system?"

Scaling
 ↓
"How does the system behave under load?"

Custom Broker
 ↓
"How does the queue work internally?"
```

The last part is especially interesting given your background: if after the main project you create a separate branch with a **custom message broker in Go**, it would make excellent material for discussing architecture, concurrency, network communication, and fault tolerance.
