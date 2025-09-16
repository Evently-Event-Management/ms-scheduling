# ms-scheduling

Refactored for maintainability by splitting monolithic `main.go` into internal packages.

## Structure

- `main.go` – application entrypoint and main loop.
- `internal/config` – configuration loading from environment variables.
- `internal/models` – shared data models (`SQSMessageBody`).
- `internal/auth` – Keycloak client credentials token retrieval.
- `internal/sqsutil` – SQS helper utilities (receive & delete messages).
- `internal/session` – business logic for processing session state changes.

## Build

```bash
go build ./...
```

## Notes
- Internal packages keep implementation details hidden from external consumers.
- AWS and app config packages are aliased to avoid name collision (`awsconfig` vs `appconfig`).
- Further enhancements could include: context cancellation, structured logging, retry/backoff abstraction, unit tests with interfaces for SQS & HTTP clients.
