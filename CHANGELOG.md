# Changelog

## v1.0.0 (2025-11-22)
- Initial stable release of the reusable Telegram bot core on telebot.v4 with webhook/long-polling modes, tuned HTTP client retries, async sender/dispatcher, command & callback registry, routers for commands/text/callbacks, default middlewares (recover, logging, metrics, rate limiting), admin guards, FSM hooks, and helpers for sending, formatting, payload parsing, and keyboards/UI.
- Configuration loader (`core/config`) that reads YAML plus environment overrides, validates Telegram modes (webhook vs. long-poll), webhook parameters, long-poll timeouts, rate-limit exclusions, and logging profile defaults.
- Bootstrap and runner flow to load config from env/default path, initialize structured logging, open PostgreSQL with pool sizing and readiness checks, apply golang-migrate migrations, and start the Telegram runtime with lifecycle hooks; includes seeder/service-provider adapters for wiring services.
- Structured logging subsystem (slog) with JSON or key-value output, ordered keys, sampling/trace override, async fan-out to stdout and optional files, contextual RID/update/user/chat enrichment, and build metadata in startup logs.
- Database utilities for Postgres DSN construction, ping verification, wait-for-ready helper, and migration runner with file preview/summary logging.
- Build metadata package (`core/buildinfo`) to inject Version/Commit/Date via ldflags for diagnostics.
