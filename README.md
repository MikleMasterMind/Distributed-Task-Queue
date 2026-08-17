# Distributed-Task-Queue

Распределённая очередь задач. Реализован HTTP API на Python (FastAPI) и Go-worker для выполнения задач.

Архитектура (план): `FastAPI → Redis Queue → Go Workers → PostgreSQL`. Сейчас хранение — JSON-файлы в папке `data/`, API и worker работают с одним и тем же каталогом задач: один файл `{task_id}.json` на задачу.

## Запуск

Требуется [uv](https://docs.astral.sh/uv/) и Python 3.12+.

```bash
uv sync --extra dev
uv run distributed-task-queue
```

API будет доступен на `http://0.0.0.0:8000` (Swagger UI — `/docs`).

По умолчанию задачи сохраняются в `data/tasks/`. Путь можно переопределить через переменную окружения `DATA_DIR`.

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

Worker периодически сканирует каталог задач, берёт задачи со статусом `PENDING`, выполняет их и записывает результат обратно в файлы. Выполнение идёт параллельно (количество горутин задаётся конфигурацией). Ошибка одной задачи не влияет на остальные.

Требуется Go 1.26+.

```bash
cd worker
go build -o worker ./cmd/worker
./worker
```

Запуск worker'а поверх данных API:

```bash
./worker --tasks-dir=data/tasks
```

Параметры:

- `--tasks-dir` (env `TASKS_DIR`) — каталог с JSON-файлами задач, по умолчанию `data/tasks`;
- `--concurrency` (env `CONCURRENCY`) — число одновременно выполняемых задач, по умолчанию `4`;
- `--poll-interval` (env `POLL_INTERVAL_MS`) — интервал сканирования каталога, по умолчанию `1s`.

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
