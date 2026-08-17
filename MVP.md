# Distributed Task Queue — MVP

## 1. Цель проекта

Создать минимальную распределённую систему для постановки фоновых задач в очередь и их выполнения отдельными Go-worker'ами.

MVP должен поддерживать следующий сценарий:

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

Пользователь создаёт задачу через HTTP API, задача попадает в очередь, Go-worker забирает её и выполняет, а результат и состояние сохраняются в PostgreSQL.

---

## 2. Компоненты

MVP состоит из четырех основных сервисов и инфраструктуры.

### Python API

Отвечает за:

* HTTP API;
* создание задач;
* получение состояния задачи;
* хранение metadata о задачах.

Технологии:

* Python 3.12+
* FastAPI
* SQLAlchemy
* PostgreSQL
* Pydantic

### Redis

Используется как message queue.

Отвечает за:

* хранение ожидающих задач;
* передачу задач worker'ам.

На этапе MVP не требуется реализовывать собственный message broker.

### Go Worker

Отвечает за:

* подключение к Redis;
* получение задач;
* выполнение задач;
* обновление статуса задачи;
* graceful shutdown.

Технологии:

* Go
* goroutines
* context
* Redis client

### PostgreSQL

Хранит:

* задачи;
* их статус;
* результат;
* ошибку;
* timestamps.

---

# 3. Task Model

Каждая задача должна иметь примерно такую модель:

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

Каждая задача выполняется **только один раз**: `started_at`/`finished_at` — единственная пара отметок запуска и завершения.

### Status

MVP должен поддерживать состояния:

```text
PENDING
RUNNING
SUCCESS
FAILED
```

Жизненный цикл:

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

## Создание задачи

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

## Получение задачи

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

# 5. Типы задач

Для MVP достаточно реализовать 2–3 простых типа задач.

Например:

### Echo

Возвращает полученный payload.

```json
{
  "type": "echo",
  "payload": {
    "message": "hello"
  }
}
```

Результат:

```json
{
  "message": "hello"
}
```

### Sleep

Используется для тестирования длительных задач.

```json
{
  "type": "sleep",
  "payload": {
    "seconds": 5
  }
}
```

Worker должен подождать 5 секунд и завершить задачу.

### Fibonacci

Позволяет проверить выполнение CPU-bound задачи.

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

При создании задачи:

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

В Redis желательно помещать не всю задачу, а только её ID:

```text
queue:
    task_id_1
    task_id_2
    task_id_3
```

Worker получает ID:

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

Это позволяет PostgreSQL оставаться источником истины для состояния задачи.

---

# 7. Go Worker

Worker запускается отдельно:

```bash
go-worker
```

После запуска он:

1. подключается к Redis;
2. проверяет соединение с PostgreSQL;
3. начинает ожидать задачи;
4. получает ID задачи;
5. получает задачу из PostgreSQL;
6. меняет статус на `RUNNING`;
7. выполняет задачу;
8. сохраняет результат;
9. меняет статус на `SUCCESS` или `FAILED`.

Пример:

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

Даже в MVP worker должен поддерживать несколько одновременно выполняемых задач.

Например:

```text
Worker
  │
  ├── goroutine → Task 1
  ├── goroutine → Task 2
  ├── goroutine → Task 3
  └── goroutine → Task 4
```

Количество параллельных задач должно задаваться конфигурацией:

```yaml
worker:
  concurrency: 4
```

Например:

```bash
go-worker --concurrency=4
```

Если одновременно пришло 20 задач, worker выполняет только 4:

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

# 9. Ошибки

Если задача завершилась ошибкой:

```text
RUNNING
   │
   ▼
FAILED
```

В PostgreSQL необходимо сохранить ошибку:

```json
{
  "status": "FAILED",
  "error": "division by zero"
}
```

Worker не должен падать из-за ошибки одной задачи.

Например:

```text
Task 1 → FAILED
Task 2 → SUCCESS
Task 3 → SUCCESS
```

Worker продолжает работать.

---

# 10. Graceful Shutdown

При получении `SIGTERM` или `SIGINT` worker должен:

1. перестать принимать новые задачи;
2. дождаться выполнения текущих задач;
3. закрыть соединения;
4. завершиться.

Например:

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

Это обязательная часть MVP, потому что worker является долгоживущим сервисом.

---

# 11. Конфигурация

Конфигурация не должна быть захардкожена.

Минимальный набор:

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

Для production-подобного варианта секреты лучше передавать через environment variables.

---

# 12. Docker Compose

Весь MVP должен запускаться одной командой:

```bash
docker compose up
```

Минимальные контейнеры:

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

# 13. Минимальные тесты

### Python

Проверить:

* создание задачи;
* получение задачи;
* валидацию payload;
* обработку несуществующего task ID.

### Go

Проверить:

* обработку каждой task type;
* успешное выполнение;
* ошибку выполнения;
* worker pool;
* graceful shutdown.

### Integration test

Основной сценарий:

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

Ожидаемый результат:

```text
PENDING → RUNNING → SUCCESS
```

---

# 14. Что НЕ нужно делать в MVP

Следующие возможности лучше оставить на следующие версии:

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
* собственный message broker;
* Kubernetes.

Они интересны, но сильно увеличат объём проекта.

---

# 15. Definition of Done

MVP считается завершённым, если можно выполнить следующий сценарий.

### Шаг 1

Запустить систему:

```bash
docker compose up
```

### Шаг 2

Создать задачу:

```http
POST /tasks
```

### Шаг 3

Получить:

```text
task_id = 123
status = PENDING
```

### Шаг 4

Go Worker автоматически получает задачу:

```text
PENDING
   ↓
RUNNING
```

### Шаг 5

Worker выполняет задачу:

```text
RUNNING
   ↓
SUCCESS
```

### Шаг 6

Получить результат:

```http
GET /tasks/123
```

и увидеть:

```json
{
  "id": "123",
  "status": "SUCCESS",
  "result": {
    "message": "hello"
  }
}
```

### Шаг 7

Запустить несколько worker'ов:

```text
Worker 1
Worker 2
Worker 3
```

и убедиться, что задачи распределяются между ними.

---

# Итоговая архитектура MVP

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

## Основная цель MVP

Не количество функций, а доказательство того, что работает полный pipeline:

**создание задачи → сохранение → очередь → распределённый worker → выполнение → сохранение результата → получение результата через API.**

После этого уже имеет смысл переходить к **V2: retries + visibility timeout + heartbeat + DLQ** — именно там начнётся самая интересная часть distributed systems.
