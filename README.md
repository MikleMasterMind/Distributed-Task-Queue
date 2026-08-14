# Distributed-Task-Queue

Распределённая очередь задач. На текущем этапе реализован минимальный HTTP API на Python (FastAPI).

Архитектура (план): `FastAPI → Redis Queue → Go Workers → PostgreSQL`. Сейчас хранение — JSON-файлы в папке `data/`.

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

Поддерживаемые типы задач: `echo` (`message`), `sleep` (`seconds`), `fibonacci` (`n`).

## Тесты

```bash
uv run pytest
```
