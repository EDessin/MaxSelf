#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/env.sh"

if docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
  echo "Postgres container ${DB_CONTAINER} is already running."
elif docker ps -a --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
  echo "Starting existing Postgres container ${DB_CONTAINER}..."
  docker start "${DB_CONTAINER}" >/dev/null
else
  echo "Creating Postgres container ${DB_CONTAINER} on localhost:${DB_PORT}..."
  docker run \
    --name "${DB_CONTAINER}" \
    -e POSTGRES_USER="${POSTGRES_USER}" \
    -e POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" \
    -e POSTGRES_DB="${POSTGRES_DB}" \
    -p "${DB_PORT}:5432" \
    -d "${DB_IMAGE}" >/dev/null
fi

echo "Waiting for Postgres..."
for _ in {1..30}; do
  if docker exec "${DB_CONTAINER}" pg_isready -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" >/dev/null 2>&1; then
    echo "Postgres is ready at ${DATABASE_URL}"
    exit 0
  fi
  sleep 1
done

echo "Postgres did not become ready in time."
exit 1
