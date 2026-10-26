# Repository Guidelines

## Project Structure & Module Organization

Leoxy is a Go HTTP reverse-proxy service. The executable entrypoint is in `cmd/main.go`. Runtime configuration is loaded by `internal/config/dotenv.go`; HTTP handlers and proxy behavior live in `internal/server/`; shared helpers belong in `internal/common/`. The repository currently has no dedicated test files or asset directories. Keep new implementation code under `internal/` unless it is specifically an executable entrypoint.

## Build, Test, and Development Commands

- `go run ./cmd` starts the service locally using `.env`.
- `go build ./...` compiles all packages and checks that the module builds.
- `go test ./...` runs the repository test suite.
- `gofmt -w path/to/file.go` formats changed Go files.

Configure `PORT` and the `PROXY_SERVER_1` through `PROXY_SERVER_4` variables before running. Use valid absolute URLs for upstream servers, such as `http://localhost:9000`.

## Coding Style & Naming Conventions

Follow standard Go style and always run `gofmt`. Use mixedCaps for Go identifiers, short descriptive names for local variables, and exported names only when they are used outside their package. Keep HTTP route registration in the application entrypoint and request behavior in focused handlers. Return or log errors with enough context to identify the affected operation.

## Testing Guidelines

Add tests beside the package they cover using files ending in `_test.go`. Name tests with the `TestXxx` convention and table-drive cases where multiple proxy paths, URLs, or configuration values are involved. Run `go test ./...` before submitting changes; add regression coverage for behavior changes.

## Commit & Pull Request Guidelines

Use concise Conventional Commit-style subjects, matching existing history: `feat:`, `refactor:`, or similar prefixes followed by an imperative summary. Keep commits focused. Pull requests should describe the behavior changed, configuration or endpoint impact, test commands run, and any setup required to reproduce the change. Include request/response examples when modifying proxy or health endpoints.

## Security & Configuration

Do not commit `.env` files, credentials, tokens, or private upstream URLs. Keep secrets in the local environment and document required variable names without exposing values. Validate configured upstream URLs before registering proxy routes.

## Permissions

Global rule:

- Ask the user first before making any code change.
- Show the intended change for review when possible.
- Wait for the user to accept or reject the change before editing project code, never change code else take premition from me.
- Always keep the changes in the code to be as simple as possible, straight to the point, with no comments added

### Allow

- Read tracked project files needed for the task.
- Read source code under `src/`, configuration under `config/`, and docs such as `README.md`, `CLAUDE.md`, and this file.
- Create new source or documentation files when they are required for the requested change.
- Edit application code, route files, models, security, utilities, tests, and markdown documentation.
- Update `package.json` when the task explicitly requires script or dependency changes.
- Run safe repo-local commands such as `rg`, `ls`, `sed`, `git status`, `pnpm lint`, and `pnpm test`.

### Deny

- Do not read, print, or copy secrets from `.env`, `.env.dev`, or any credential file.
- Do not modify `.env`, `.env.dev`, or other secret-bearing files unless the user explicitly asks.
- Do not modify `node_modules/`, generated caches, or log files.
- Do not change `pnpm-lock.yaml` unless dependency work is part of the task.
- Do not delete files, rename major directories, or rewrite large parts of the codebase without explicit approval.
- Do not run destructive git or shell commands such as `git reset --hard`, `git checkout --`, or broad `rm` operations.
- Do not alter deployment/infrastructure files (`Dockerfile`, `docker-compose.yml`, `ecosystem.config.cjs`) unless the task explicitly requires it.
