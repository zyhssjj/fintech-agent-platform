# WFAgent Backend Base

This directory is the reusable backend base for the technology finance system. It is an independently runnable empty system: login, role/permission assignment data, audit logs, file upload, async tasks, routing, config, and shared utilities are ready before business modules are added.

## Structure

- `cmd/server`: HTTP server entrypoint.
- `cmd/migrate`: SQL migration runner for `migrations/*.up.sql` and `*.down.sql`.
- `internal/base`: base platform code for config, database, auth, middleware, response, file service, task queue, router, DTOs, and utilities.
- `template/module-xxx`: backend module starter template for business teams.
- `migrations`: base SQL migration files.
- `deploy`: Dockerfile and local dependency compose file.

No `internal/module-*` business module is registered or shipped in the backend base runtime. Business teams start from `template/module-xxx`.

## Commands

```bash
go run ./cmd/server
go test ./...
go test ./internal/base/...
go run ./cmd/migrate up
go run ./cmd/migrate down
go build -o bin/wfagent-server ./cmd/server
```

The Makefile wraps the same commands with project-local Go caches:

```bash
make run
make test
make test-base
make migrate-up
make build
```

## Local Startup

1. Copy `.env.example` to `.env`, then adjust secrets and dependency endpoints.
2. Start PostgreSQL, Redis, and MinIO with `docker compose -f deploy/docker-compose.yml up -d`.
3. Set `DB_ENABLED=true` when you want persistent users, roles, audit logs, and tasks.
4. Run `make migrate-up`, then `make run`.
5. Check `GET /healthz`.

Default bootstrap login when DB is enabled:

- Username: `admin`
- Password: `Admin@123456`

## Base APIs

- `POST /v1/auth/login`
- `POST /v1/auth/refresh`
- `POST /v1/auth/logout`
- `GET /v1/auth/me`
- `GET /v1/system/roles` and `GET /v1/system/permissions`
- `PUT /v1/system/users/{user_id}/roles` with `{ "codes": ["VIEWER"] }`
- `PUT /v1/system/roles/{role_id}/permissions` with `{ "codes": ["base:read"] }`
- `POST /v1/files`
- `POST /v1/upload/init`
- `POST /v1/upload/{upload_id}/chunk`
- `POST /v1/upload/{upload_id}/complete`
- `GET /v1/files/preview/{object_name}`
- `GET /v1/files/download/{object_name}`
- `DELETE /v1/files/{object_name}`
- `POST /v1/tasks`
- `GET /v1/tasks/{task_id}`
- `GET /v1/tasks/{task_id}/events`
- `POST /v1/tasks/{task_id}/cancel`

## Development Rules

- Handlers must return through `internal/base/response`.
- Protected routes use `middleware.Auth()`.
- Role checks use `middleware.RequireRole(...)`.
- Resource/action checks use `middleware.RequireResource(resource, action)` or `middleware.RequirePermission(...)`.
- Request/response structs belong in `internal/base/dto`.
- New business work starts by copying `template/module-xxx`; business teams should not edit `internal/base`.
