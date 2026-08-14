# Distributed Task Queue — Roadmap

## Цель

После завершения MVP постепенно превратить проект из простой очереди задач в полноценную распределённую систему выполнения фоновых задач.

Расширение должно происходить поэтапно. Каждый этап должен добавлять отдельную архитектурную возможность, которую можно протестировать и продемонстрировать.

---

# V1 — MVP

**Цель:** базовое выполнение задач.

Уже реализовано:

* FastAPI;
* PostgreSQL;
* Redis Queue;
* Go Worker;
* Worker Pool;
* несколько типов задач;
* статусы `PENDING`, `RUNNING`, `SUCCESS`, `FAILED`;
* обработка ошибок;
* graceful shutdown;
* Docker Compose;
* unit и integration tests.

Архитектура:

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

# V2 — Надёжность выполнения

**Цель:** система должна корректно работать при падении worker'ов.

Это первый этап, где появляются настоящие distributed-systems проблемы.

## 2.1 Retry

Добавить автоматический повтор выполнения задачи.

Например:

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

Добавить:

```yaml
retry:
  max_attempts: 3
  backoff: exponential
```

Поддержать:

* `max_attempts`;
* exponential backoff;
* configurable delay;
* retryable/non-retryable errors.

---

## 2.2 Visibility Timeout

Проблема:

```text
Worker A
   │
   └── получил Task 123
          │
          X CRASH
```

Task не должна потеряться.

При получении задача становится временно невидимой для других workers.

Если worker не завершил её за определённое время:

```text
visibility timeout
        │
        ▼
Task → QUEUED
```

---

## 2.3 Dead Letter Queue

Если задача постоянно падает:

```text
attempt 1 → FAILED
attempt 2 → FAILED
attempt 3 → FAILED
attempt 4 → FAILED
```

Она должна попасть в:

```text
Dead Letter Queue
```

Добавить API:

```http
GET /dead-letter/tasks
POST /dead-letter/tasks/{id}/retry
```

---

## 2.4 Task timeout

Добавить ограничение времени выполнения:

```yaml
task:
  timeout: 60s
```

Если задача работает слишком долго:

```text
RUNNING
   │
   │ 60 sec
   ▼
TIMEOUT
```

---

# V3 — Worker Management

**Цель:** превратить набор workers в управляемый кластер.

## 3.1 Worker Registration

При запуске worker регистрируется:

```text
Worker
   │
   ▼
Control Plane
```

Информация:

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

Worker регулярно сообщает:

```text
Worker
   │
   ├── heartbeat
   ├── heartbeat
   ├── heartbeat
   └── heartbeat
```

Если heartbeat перестал приходить:

```text
Worker
   │
   X
   │
   ▼
DEAD
```

Это позволит обнаруживать падение worker'ов независимо от task timeout.

---

## 3.3 Worker Status

Добавить состояния:

```text
STARTING
IDLE
BUSY
DRAINING
DEAD
```

Например:

```http
GET /workers
```

Ответ:

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

Добавить возможность вывести worker из эксплуатации:

```http
POST /workers/{id}/drain
```

Worker:

```text
DRAINING
   │
   ├── перестаёт принимать новые задачи
   │
   ├── завершает текущие
   │
   ▼
STOPPED
```

Это позволит безопасно обновлять workers.

---

# V4 — Advanced Queues

**Цель:** сделать очередь более функциональной.

## 4.1 Priority Queue

Добавить приоритет:

```text
priority = 1
priority = 10
priority = 100
```

Worker сначала получает задачи с высоким приоритетом.

---

## 4.2 Multiple Queues

Например:

```text
default
high-priority
cpu
io
ml
```

Worker может подписаться на определённые queues:

```bash
go-worker --queues=default,high-priority
```

Другой:

```bash
go-worker --queues=cpu
```

---

## 4.3 Routing

Добавить routing rules:

```text
task.type = image_processing
        ↓
image queue

task.type = ml_inference
        ↓
ml queue
```

Таким образом CPU-heavy задачи не будут мешать обычным задачам.

---

# V5 — Scheduling

**Цель:** добавить отложенные и периодические задачи.

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

Task не должна появиться в обычной очереди до указанного времени.

---

## 5.2 Periodic Tasks

Например:

```yaml
schedule:
  name: cleanup
  task: cleanup_old_files
  cron: "0 * * * *"
```

Scheduler создаёт задачи автоматически:

```text
Scheduler
   │
   ├── 10:00 → cleanup
   ├── 11:00 → cleanup
   ├── 12:00 → cleanup
   └── ...
```

---

# V6 — Idempotency и Exactly-once Semantics

**Цель:** исследовать проблему повторного выполнения задач.

В распределённой системе возможна ситуация:

```text
Worker
  │
  ├── выполняет task
  │
  ├── task завершилась
  │
  X worker crash
  │
  ▼
Queue считает task незавершённой
  │
  ▼
Другой worker выполняет её повторно
```

Одна задача может быть выполнена дважды.

---

## 6.1 Idempotency Key

Добавить:

```json
{
  "idempotency_key": "payment-123"
}
```

Повторная постановка:

```text
payment-123
payment-123
payment-123
```

должна приводить к одному логическому выполнению.

---

## 6.2 Документировать Delivery Semantics

Явно определить:

```text
At-most-once
At-least-once
Exactly-once
```

В проекте желательно реализовать:

> **At-least-once delivery + idempotent task execution**

и отдельно описать, почему полноценный exactly-once execution является сложной задачей в распределённых системах.

---

# V7 — Observability

**Цель:** получить полноценное наблюдение за системой.

## 7.1 Metrics

Использовать Prometheus.

Метрики:

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

Создать dashboard:

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

Добавить OpenTelemetry.

Например:

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

В trace должно быть видно время каждого этапа.

---

# V8 — Control Plane / Worker Plane

**Цель:** разделить архитектуру на две логические части.

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

Control Plane отвечает за управление.

Worker Plane отвечает за выполнение.

---

# V9 — Horizontal Scaling

**Цель:** проверить масштабирование системы.

Запустить:

```text
1 worker
```

и измерить throughput.

Затем:

```text
2 workers
4 workers
8 workers
16 workers
```

Сравнить:

```text
workers | tasks/sec | p95 latency
--------|-----------|------------
1       | 50        | 120 ms
2       | 95        | 80 ms
4       | 180       | 45 ms
8       | 340       | 30 ms
```

Важно не просто увеличить количество workers, а определить:

* bottleneck;
* CPU bottleneck;
* Redis bottleneck;
* PostgreSQL bottleneck;
* network bottleneck.

---

# V10 — Load Testing

Добавить нагрузочные тесты.

Например:

```text
10 000 tasks
100 workers
```

Проверить:

* throughput;
* latency;
* failure rate;
* queue growth;
* memory consumption;
* CPU usage.

Можно использовать:

* Locust;
* k6;
* собственный Go load generator.

---

# V11 — Собственный Message Broker

Это **опциональный advanced этап**.

После того как Redis-версия работает, заменить Redis собственной реализацией.

Например:

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

Можно реализовать:

* TCP protocol;
* message framing;
* append-only log;
* persistence;
* consumer offsets;
* partitions;
* consumer groups;
* batching.

Это уже отдельный большой проект и **не является обязательной частью Task Queue**.

---

# V12 — Distributed Broker

Если хочется ещё глубже погрузиться в distributed systems:

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

Исследовать:

* replication;
* leader election;
* failure detection;
* quorum;
* consistency;
* recovery.

Этот этап лучше рассматривать как отдельный экспериментальный проект.

---

# Приоритет реализации

Не стоит реализовывать все этапы подряд.

Оптимальный путь:

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

После этого проект уже будет достаточно серьёзным для резюме.

---

# Что я бы НЕ делал

Не нужно превращать проект в попытку создать Kubernetes + Kafka + Celery одновременно.

Особенно не стоит без необходимости добавлять:

* Kubernetes;
* service mesh;
* несколько десятков микросервисов;
* собственную БД;
* собственный consensus protocol;
* сложную UI-панель.

Лучше иметь:

```text
10 хорошо реализованных механизмов
```

чем:

```text
50 технологий, соединённых Docker Compose.
```

---

# Финальная версия

Если довести проект до разумного production-like состояния, итоговая архитектура может выглядеть так:

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

Главная идея roadmap — **каждая следующая версия должна решать конкретную проблему распределённых систем**:

```text
MVP
 ↓
"Как выполнить задачу?"

Reliability
 ↓
"Что будет, если worker упадёт?"

Worker Management
 ↓
"Как управлять множеством workers?"

Advanced Queues
 ↓
"Как распределять разные типы нагрузки?"

Scheduling
 ↓
"Как выполнять задачи в будущем?"

Idempotency
 ↓
"Что делать с повторным выполнением?"

Observability
 ↓
"Как понять, что происходит в системе?"

Scaling
 ↓
"Как система ведёт себя под нагрузкой?"

Custom Broker
 ↓
"Как работает сама очередь изнутри?"
```

Именно последняя часть особенно интересна с учётом твоего образования: если после основного проекта сделать отдельную ветку с **собственным message broker на Go**, получится очень хороший материал для обсуждения архитектуры, concurrency, сетевого взаимодействия и отказоустойчивости.
