# Repository Guidelines

## Project Structure & Module Organization

This repository is split into a Go backend and a React/Vite frontend.

- `backend/`: Gin + GORM service base.
  - `cmd/server/`: HTTP server entrypoint.
  - `internal/base/`: shared platform code: config, routing, middleware, auth, response, DTOs, file upload, and tasks.
  - `internal/module-*`: business module boundaries (`agent`, `fund`, `invest`, `project`, `meeting`, `exit`).
  - `migrations/`: SQL migration files.
  - `template/module-xxx/`: backend module starter template.
- `frontend/`: React + Vite + TypeScript app.
  - `src/base/`: shared API client, contracts, stores, layout, components, and common views.
  - `src/module-*`: business module API wrappers, hooks, components, and views.
  - `src/template/module-xxx/`: frontend module starter template.

## Build, Test, and Development Commands

Backend:

- `cd backend && go run ./cmd/server`: start the API server on `:8080`.
- `cd backend && go test ./...`: compile and run all Go tests.
- `cd backend && gofmt -w cmd internal pkg template`: format project Go source.
- `cd backend && make build`: build `bin/wfagent-server`.

Frontend:

- `cd frontend && npm ci`: install dependencies from `package-lock.json`.
- `cd frontend && npm run dev`: start the local app server on `:3000`.
- `cd frontend && npm run lint`: run TypeScript checks.
- `cd frontend && npm run build`: build the Vite app and Node server bundle.

## Coding Style & Naming Conventions

Use `gofmt` for Go. Backend handlers must return through `internal/base/response`, not hand-built JSON. Keep request/response structs in `internal/base/dto` and preserve API document `snake_case` JSON fields.

Use TypeScript for frontend code. Keep page logic in `composables/` hooks and request logic in module `api/` folders. Component files use PascalCase; hooks use `useXxx.ts`.

## Mandatory Base Rules

The following rules are mandatory for all module development on top of this base:

1. API responses must go through `backend/internal/base/response`; do not hand-build JSON payloads in handlers.
2. Auth and permission checks must reuse base middleware such as `middleware.Auth()` and the existing permission guards.
3. Audit logging must remain on the base path for create/update/delete, permission assignment, upload, and other sensitive operations.
4. File upload and object access must reuse `backend/internal/base/file`; do not create ad hoc local-file upload flows.
5. Long-running workflows must reuse `backend/internal/base/task` for submit/query/cancel/progress instead of custom task queues.
6. Schema changes must be delivered through `migrations/` with clear incremental SQL migration files.
7. Error handling must reuse the unified response/error-code system; new error codes should be added centrally first.
8. Backend logging must use structured logging conventions; avoid `fmt.Println`-style debug output in production code.
9. Backend API changes must keep Swagger annotations updated so frontend and backend stay aligned.
10. Frontend HTTP calls must reuse the shared request layer in `@/base/api`; do not scatter raw `fetch` logic inside views.

## Testing Guidelines

Add Go tests as `*_test.go` near the package under test. For frontend additions, at minimum keep `npm run lint` and `npm run build` passing; add component or hook tests when a test framework is introduced.

## Commit & Pull Request Guidelines

No readable Git history is available in this workspace, so use Conventional Commits going forward, for example `feat: add investment project API wrapper` or `fix: handle token refresh failure`.

Pull requests should include a concise summary, changed modules, validation commands run, related issues, and screenshots for visible UI changes.

## Security & Configuration Tips

Do not commit real secrets. Copy `.env.example` to `.env` for local settings. Keep generated caches and logs out of commits (`node_modules/`, `.npm-cache/`, `.gocache/`, `.gomodcache/`, `logs/`, `dist/`).
