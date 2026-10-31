# Contributing to Leoxy

Thanks for helping. Leoxy is a Go reverse proxy with HTTP, TCP and UDP upstreams, so changes touch network and security code. Keep them small, tested, and easy to review.

## Requirements

- Go 1.26.2 (see `.go-version`)
- Git
- Docker, only for image and compose changes
- `govulncheck`: `go install golang.org/x/vuln/cmd/govulncheck@latest`

## Layout

```text
app/cli/                    CLI entrypoint (start, stop, version, hidden run)
app/pkg/core/               platform process and signal helpers, server wiring and shutdown
app/pkg/internal/common/    client IP, request logger, metrics
app/pkg/internal/file/      config loading, validation, defaults
app/pkg/internal/security/  access control, rate limits, body limit
app/pkg/internal/http_server/  proxy, health, TCP, UDP, shared helpers
tools/                      benchmark tool (separate module)
docker/                     Dockerfiles and compose
example/                    sample config
docs/                       CLI and security notes
```

`go.work` links `app` and `tools`. Keep new implementation code under `app/pkg/internal/` unless it is an executable entrypoint.

## Setup

```bash
git clone https://github.com/elrefai99/Leoxy.git
cd Leoxy
make build
make run
```

`make run` creates `leoxy/config.yaml` on first start. Edit it, or start from `example/config.yaml`. Environment overrides are listed in `docs/cli.md`.

## Before you open a PR

Run from `app/`:

```bash
gofmt -l .
go vet ./...
go test -race ./...
govulncheck ./...
```

`gofmt -l` must print nothing. CI runs vet, test and build, plus the security workflow when present.

## Coding rules

- Standard Go style. mixedCaps identifiers, short local names, export only what other packages use.
- Return or log errors with enough context to identify the operation. Never log secrets, tokens, API keys, or query strings.
- No comments unless the code is not self-explanatory.
- Keep route registration in `app/pkg/server` and request behavior in focused handlers.
- Every change to proxy, security, or config behavior needs a test. Tests live next to the package as `_test.go`, table-driven when several inputs are involved.
- New config fields: add the `mapstructure` tag, validate in `file.Validate`, document in `README.md`, `docs/security.md` when relevant, and `example/config.yaml`.
- Never hold a mutex while calling the next handler or doing network I/O.
- Do not trust client-supplied forwarding headers except through `trusted_proxy_cidrs`.
- Never commit `.env` files, credentials, tokens, or private upstream URLs.
- Do not add a dependency without a clear reason. Prefer the standard library.

## Security-sensitive changes

Changes to authentication, rate limiting, client IP detection, TLS, TCP/UDP listeners, or default config must include:

1. The threat or failure the change addresses.
2. A test that fails without the change.
3. Notes on any new default or breaking behavior.

Report vulnerabilities privately by email to the maintainer listed in `AUTHORS`. Do not open a public issue for an unfixed vulnerability.

## Commits and pull requests

Use Conventional Commit titles: lowercase, imperative, no trailing period, under about 70 characters.

| Type | Use for | Version bump |
| --- | --- | --- |
| `feat(scope)` | new capability, any file count | minor |
| `chore(scope)` | small updates, at most 2 files | patch |
| `refactor(scope)` | 3 or more files, no new capability | patch |
| `fix(scope)` | bug fix for a bug live in production | patch |
| `docs` | documentation only, no scope | none |

Scope is the package or feature name used in `git log` (for example `security`, `proxy`, `tcp`, `docker`). Omit it for repo-wide changes. Use `!` after the scope for a breaking change and call it out in the PR.

Each PR body has exactly three summary bullets, a version line, and a verification line:

```text
## Summary
- <change 1>
- <change 2>
- <change 3>

## Version
<old> -> <new>, see CHANGELOG.md

## Verification
<exact commands and results, and what was not verified>
```

Each PR also:

- adds the same three points to `CHANGELOG.md` under Added, Changed, Fixed, or Documentation (docs go under `[Unreleased]`)
- bumps the version in both `version.json` and the `version` constant in `app/cli/main.go`, and keeps `app/README.md` in step
- describes only its own changes, not inherited ones from a parent branch

State what you verified and what you did not. Do not write "safe" or "fixed" without pointing at a test or command.

## Docker and CI

- Build the image with `docker build -f docker/lx-2/Dockerfile .` and validate compose with `docker compose -f docker/lx-2/docker-compose.yml config --quiet`.
- Do not change `Dockerfile`, compose files, or `ecosystem.config.cjs` unless the change needs it.
- Pin new GitHub Actions to a major version at minimum and keep `permissions` minimal.

## AI coding agents

Follow `AGENTS.md`. In short: ask before making a code change, show the intended change, keep changes simple, add no comments, never read or modify `.env` files, never run destructive git or shell commands, and do not rewrite large parts of the codebase without approval.

## License

By contributing you agree that your contribution is licensed under the Apache License 2.0 in `LICENSE`.
