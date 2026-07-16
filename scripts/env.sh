#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

: "${APP_NAME:=maxself}"
: "${DB_CONTAINER:=maxself-local-postgres}"
: "${DB_IMAGE:=postgres:16-alpine}"
: "${DB_PORT:=5432}"
: "${POSTGRES_USER:=maxself}"
: "${POSTGRES_PASSWORD:=maxself}"
: "${POSTGRES_DB:=maxself}"
: "${DATABASE_URL:=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${DB_PORT}/${POSTGRES_DB}?sslmode=disable}"
: "${JWT_SECRET:=dev-only-change-me}"
: "${FRONTEND_URL:=http://localhost:4201}"
: "${IDENTITY_SERVICE_URL:=http://localhost:8081}"
: "${ACTIVITY_SERVICE_URL:=http://localhost:8082}"
: "${PROGRESS_SERVICE_URL:=http://localhost:8083}"
: "${GOOGLE_CLIENT_ID:=}"
: "${GOOGLE_CLIENT_SECRET:=}"
: "${GOOGLE_REDIRECT_URL:=http://localhost:8081/auth/google/callback}"
: "${GOMODCACHE:=${PROJECT_ROOT}/.cache/go-mod}"
: "${GOCACHE:=${PROJECT_ROOT}/.cache/go-build}"
: "${GOSUMDB:=off}"

export APP_NAME DB_CONTAINER DB_IMAGE DB_PORT POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB
export DATABASE_URL JWT_SECRET FRONTEND_URL IDENTITY_SERVICE_URL ACTIVITY_SERVICE_URL PROGRESS_SERVICE_URL
export GOOGLE_CLIENT_ID GOOGLE_CLIENT_SECRET GOOGLE_REDIRECT_URL GOMODCACHE GOCACHE GOSUMDB
export PROJECT_ROOT
