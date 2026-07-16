SHELL := /bin/bash

APP_NAME ?= maxself
DB_CONTAINER ?= maxself-local-postgres
DB_IMAGE ?= postgres:16-alpine
DB_PORT ?= 5432
POSTGRES_USER ?= maxself
POSTGRES_PASSWORD ?= maxself
POSTGRES_DB ?= maxself
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(DB_PORT)/$(POSTGRES_DB)?sslmode=disable
JWT_SECRET ?= dev-only-change-me
FRONTEND_URL ?= http://localhost:4201
IDENTITY_SERVICE_URL ?= http://localhost:8081
ACTIVITY_SERVICE_URL ?= http://localhost:8082
PROGRESS_SERVICE_URL ?= http://localhost:8083
GOOGLE_CLIENT_ID ?=
GOOGLE_CLIENT_SECRET ?=
GOOGLE_REDIRECT_URL ?= http://localhost:8081/auth/google/callback
GOMODCACHE ?= $(CURDIR)/.cache/go-mod
GOCACHE ?= $(CURDIR)/.cache/go-build
GOSUMDB ?= off

export APP_NAME DB_CONTAINER DB_IMAGE DB_PORT POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB
export DATABASE_URL JWT_SECRET FRONTEND_URL IDENTITY_SERVICE_URL ACTIVITY_SERVICE_URL PROGRESS_SERVICE_URL
export GOOGLE_CLIENT_ID GOOGLE_CLIENT_SECRET GOOGLE_REDIRECT_URL GOMODCACHE GOCACHE GOSUMDB

.PHONY: help dev db-up db-down db-reset db-logs db-shell services api identity activity progress web test coverage coverage-backend coverage-web build-web compose-config ps

help:
	@printf "\nMaxSelf local development\n\n"
	@printf "  make dev            Start Postgres in Docker, then Go services + Angular locally\n"
	@printf "  make db-up          Start only local Postgres in Docker\n"
	@printf "  make services       Run Go microservices locally, without Angular\n"
	@printf "  make web            Run Angular locally on http://localhost:4201\n"
	@printf "  make test           Run backend tests\n"
	@printf "  make coverage       Run backend + frontend coverage checks (90%% statement/line target)\n"
	@printf "  make build-web      Build Angular frontend\n"
	@printf "  make db-down        Stop and remove the local PostgreSQL container\n"
	@printf "  make db-reset       Recreate the local database from scratch\n\n"

dev:
	@bash scripts/dev.sh

db-up:
	@bash scripts/db-up.sh

db-down:
	@bash scripts/db-down.sh

db-reset: db-down db-up

db-logs:
	@docker logs -f "$(DB_CONTAINER)"

db-shell:
	@docker exec -it "$(DB_CONTAINER)" psql -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)"

services:
	@bash scripts/services.sh

identity: db-up
	@PORT=8081 go run ./cmd/identity

activity: db-up
	@PORT=8082 go run ./cmd/activity

progress: db-up
	@PORT=8083 go run ./cmd/progress

api:
	@PORT=8080 go run ./cmd/api

web:
	@cd web && npm start -- --port 4201

test:
	@go test ./cmd/... ./internal/...

coverage: coverage-backend coverage-web

coverage-backend:
	@bash scripts/check-go-coverage.sh 90

coverage-web:
	@cd web && npm run test:coverage

build-web:
	@cd web && npm run build

compose-config:
	@docker compose config --quiet

ps:
	@docker ps --filter "name=$(DB_CONTAINER)"
