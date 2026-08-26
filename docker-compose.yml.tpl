services:
  redis:
    image: redis:7-alpine
    ports:
      - "${REDIS_PORT}:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: ${REDIS_HEALTHCHECK_INTERVAL}
      timeout: ${REDIS_HEALTHCHECK_TIMEOUT}
      retries: ${REDIS_HEALTHCHECK_RETRIES}

  postgres:
    image: postgres:16-alpine
    ports:
      - "${POSTGRES_PORT}:5432"
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: ${POSTGRES_HEALTHCHECK_INTERVAL}
      timeout: ${POSTGRES_HEALTHCHECK_TIMEOUT}
      retries: ${POSTGRES_HEALTHCHECK_RETRIES}

  api:
    build:
      context: .
      dockerfile: src/Dockerfile
    ports:
      - "${API_PORT}:8000"
    environment:
      DATABASE_URL: postgresql+asyncpg://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      REDIS_URL: redis://redis:${REDIS_PORT}/0
      QUEUE_KEY: ${QUEUE_KEY}
      LOG_LEVEL: ${LOG_LEVEL}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  worker:
    build:
      context: .
      dockerfile: worker/Dockerfile
    environment:
      DATABASE_URL: postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      REDIS_URL: redis://redis:${REDIS_PORT}/0
      QUEUE_KEY: ${QUEUE_KEY}
      LOG_LEVEL: ${LOG_LEVEL}
      CONCURRENCY: ${CONCURRENCY}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
