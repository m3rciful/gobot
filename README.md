# gobot

Telegram bot core in Go. Provides shared plumbing for logging, configuration, database access, and routing of messages/commands.

## Features
- Bootstrap pipeline: initialize logger, connect to DB, apply migrations.
- Configuration via `envconfig` (defaults to `CONFIG_PATH`).
- PostgreSQL support with `sqlx`, migrations with `golang-migrate`.
- Telegram engine on `telebot.v4`: middleware, routers for commands/messages/callbacks, sending helpers.
- Build metadata via `core/buildinfo` (ldflags friendly).

## Quick start (core)
1. Requirements: Go 1.23+ (toolchain 1.24), PostgreSQL if you need a DB.
2. Fetch deps: `go mod tidy`.
3. Run tests: `go test ./...`.
4. Build your bot using module `github.com/m3rciful/gobot` as the core dependency.

## Layout
- `core/bootstrap` — startup pipeline and DI-like options.
- `core/config` — configuration loading from env/files.
- `core/database` — DB connection and migrations.
- `core/logger` — slog setup and runtime metadata.
- `core/telegram` — Telebot integration: routers, middleware, sending, state.
- `core/cmd` — application launcher (config loading, graceful shutdown).

## Build with metadata
Example with version and commit:
```
go build -ldflags "-X 'github.com/m3rciful/gobot/core/buildinfo.Version=v1.2.3' -X 'github.com/m3rciful/gobot/core/buildinfo.Commit=abcdef0' -X 'github.com/m3rciful/gobot/core/buildinfo.Date=2025-08-30T12:00:00Z'" ./...
```
