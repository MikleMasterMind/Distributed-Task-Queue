# Distributed-Task-Queue

Распределённая очередь задач. HTTP API на Python (FastAPI), очередь на Redis и Go-worker для выполнения задач.

Архитектура: `FastAPI → Redis Queue → Go Workers → файловое хранилище`. При создании задачи FastAPI сохраняет её состояние и помещает `task_id` в очередь Redis. Go-worker получает ID из очереди (блокирующий `BRPOP`), забирает задачу, выполняет и записывает результат. Состояние задач хранится в JSON-файлах в папке `data/` (один файл `{task_id}.json` на задачу).

## Запуск

Требуется [uv](https://docs.astral.sh/uv/), Python 3.12+, Go 1.26+ и Redis (или Docker).

```bash
cp .env.example .env          # при необходимости
docker compose up -d redis    # поднять Redis
uv sync --extra dev
```

Запуск API:

```bash
uv run distributed-task-queue
```

Запуск Go-worker (в отдельном терминале):

```bash
cd worker
go build -o worker ./cmd/worker
./worker --queue=redis
```

API будет доступен на `http://0.0.0.0:8000` (Swagger UI — `/docs`). Worker забирает задачи из Redis и выполняет их, записывая результат в `data/tasks/`.

Конфигурация читается из `.env` (см. `.env.example`):

| Переменная | Значение по умолчанию | Описание |
| --- | --- | --- |
| `DATA_DIR` | `data` | базовая директория хранения задач |
| `QUEUE_TYPE` | `redis` | бэкенд очереди API: `redis`, `memory` |
| `REDIS_URL` | `redis://localhost:6379/0` | URL подключения к Redis |
| `QUEUE_KEY` | `dtq:tasks` | ключ списка Redis с очередью задач |
| `LOG_LEVEL` | `info` | уровень логов API (`debug`, `info`, `warning`, `error`, `critical`) |

## Примеры запросов

Создание задачи:

```bash
curl -X POST localhost:8000/tasks \
  -H 'Content-Type: application/json' \
  -d '{"type": "echo", "payload": {"message": "hello"}}'
```

Получение задачи:

```bash
curl localhost:8000/tasks/<task_id>
```

Удаление задачи (только если она ещё не начала выполняться, `PENDING`):

```bash
curl -X DELETE localhost:8000/tasks/<task_id>
```

Успех — `204 No Content`. Ошибки: `404` (задача не найдена), `409` (задача уже начала выполняться).

Список задач с пагинацией и фильтрами:

```bash
curl 'localhost:8000/tasks?limit=20&status=PENDING&type=echo'
```

Ответ:

```json
{
  "items": [ ... ],
  "next-token": "...",
  "limit": 20
}
```

Параметры: `limit` (1–100, по умолчанию 20), `next-token` (пагинация), `status`, `type`, `created_after`/`created_before`, `started_after`/`started_before` (по дате запуска), `finished_after`/`finished_before` (по дате завершения). Задачи отсортированы по дате создания (старые сверху).

Поддерживаемые типы задач: `echo` (`message`), `sleep` (`seconds`), `fibonacci` (`n`).

## Go Worker

Worker получает ID задач из Redis (`BRPOP`) и выполняет их параллельно (количество горутин задаётся конфигурацией). Ошибка одной задачи не влияет на остальные.

```bash
cd worker
go build -o worker ./cmd/worker
./worker --queue=redis
```

Параметры (флаг `--name` или переменная окружения из `.env`):

- `--queue` (env `QUEUE_TYPE`) — бэкенд очереди: `redis` или `dir` (поллинг каталога, по умолчанию `dir`);
- `--redis-url` (env `REDIS_URL`) — URL подключения к Redis, по умолчанию `redis://localhost:6379/0`;
- `--queue-key` (env `QUEUE_KEY`) — ключ очереди в Redis, по умолчанию `dtq:tasks`;
- `--tasks-dir` (env `TASKS_DIR`) — каталог с JSON-файлами задач, по умолчанию `data/tasks` в корне репозитория;
- `--concurrency` (env `CONCURRENCY`) — число одновременно выполняемых задач, по умолчанию `4`;
- `--poll-interval` (env `POLL_INTERVAL_MS`) — интервал сканирования для бэкенда `dir`, по умолчанию `1s`;
- `--log-level` (env `LOG_LEVEL`) — уровень логирования (`debug`, `info`, `warn`, `error`), по умолчанию `info`.

Бэкенд очереди — плагин: чтобы добавить новый, достаточно реализовать интерфейс `Queue` (`Pop(ctx) (string, error)`) и зарегистрировать его в `worker/internal/queue/factory.go`.

Worker останавливается по `SIGINT`/`SIGTERM`: перестаёт брать новые задачи, дожидается завершения текущих и выходит.

Тесты:

```bash
cd worker
go test ./...
```

## Тесты

```bash
uv run pytest
```

Интеграционные тесты (`tests/test_integration.py`) прогоняют полный конвейер: API создаёт задачу через HTTP, реальный Go-worker (автоматически собирается во временный каталог при запуске тестов) выполняет её через очередь Redis, а результат читается обратно через API. Требуется установленный Go и доступный Redis; если Redis не запущен, тесты автоматически пытаются поднять его через `docker compose up -d redis`. Если Redis недоступен, интеграционные тесты пропускаются.