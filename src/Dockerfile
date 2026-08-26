FROM python:3.12-slim AS base

RUN pip install --no-cache-dir uv

WORKDIR /app

COPY pyproject.toml uv.lock ./
COPY src/ src/
RUN uv sync --frozen --no-dev

COPY src/ .

EXPOSE 8000

CMD [".venv/bin/distributed-task-queue"]
