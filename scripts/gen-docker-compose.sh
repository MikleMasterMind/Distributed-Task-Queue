#!/usr/bin/env bash
set -euo pipefail

if [ ! -f docker-compose.yml.tpl ]; then
  echo "Error: docker-compose.yml.tpl not found" >&2
  exit 1
fi

if [ ! -f .env ]; then
  echo "Warning: .env not found, using defaults" >&2
fi

if ! command -v envsubst >/dev/null 2>&1; then
  echo "Error: envsubst not found. Install gettext: brew install gettext" >&2
  exit 1
fi

if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

: "${REDIS_PORT:=6379}"
: "${POSTGRES_PORT:=5432}"
: "${POSTGRES_DB:=dtq}"
: "${POSTGRES_USER:=dtq}"
: "${POSTGRES_PASSWORD:=dtq}"
: "${REDIS_HEALTHCHECK_INTERVAL:=5s}"
: "${REDIS_HEALTHCHECK_TIMEOUT:=3s}"
: "${REDIS_HEALTHCHECK_RETRIES:=5}"
: "${POSTGRES_HEALTHCHECK_INTERVAL:=5s}"
: "${POSTGRES_HEALTHCHECK_TIMEOUT:=3s}"
: "${POSTGRES_HEALTHCHECK_RETRIES:=5}"

export REDIS_PORT POSTGRES_PORT POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD \
  REDIS_HEALTHCHECK_INTERVAL REDIS_HEALTHCHECK_TIMEOUT REDIS_HEALTHCHECK_RETRIES \
  POSTGRES_HEALTHCHECK_INTERVAL POSTGRES_HEALTHCHECK_TIMEOUT POSTGRES_HEALTHCHECK_RETRIES

envsubst '$REDIS_PORT $POSTGRES_PORT $POSTGRES_DB $POSTGRES_USER $POSTGRES_PASSWORD $REDIS_HEALTHCHECK_INTERVAL $REDIS_HEALTHCHECK_TIMEOUT $REDIS_HEALTHCHECK_RETRIES $POSTGRES_HEALTHCHECK_INTERVAL $POSTGRES_HEALTHCHECK_TIMEOUT $POSTGRES_HEALTHCHECK_RETRIES' \
  < docker-compose.yml.tpl > docker-compose.yml
