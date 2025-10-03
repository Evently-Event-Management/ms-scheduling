# ms-scheduling

Refactored for maintainability by splitting monolithic `main.go` into internal packages.

## Structure

- `main.go` – application entrypoint and main loop.
- `internal/config` – configuration loading from environment variables.
- `internal/models` – shared data models (`SQSMessageBody`, `DebeziumEvent`).
- `internal/auth` – Keycloak client credentials token retrieval.
- `internal/sqsutil` – SQS helper utilities (receive & delete messages).
- `internal/session` – business logic for processing session state changes.
- `internal/kafka` – Kafka consumer for processing Debezium events.

## Build & Run

### Local Development
```bash
go build ./...
go run .
```

### Using Docker
```bash
# Build the Docker image
docker build -t ms-scheduling .

# Run with environment variables
docker run -p 8080:8080 \
  --env-file .env \
  -e AWS_REGION=ap-south-1 \
  ms-scheduling
```

### Using Docker Compose
```bash
# Start all services defined in docker-compose.yml
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

## Environment Variables

The application uses environment variables for configuration. You can set these variables in your environment or use a `.env` file. The application looks for a `.env` file in the following locations:

1. Current directory (`.env`)
2. Parent directory (`../.env`)
3. Home directory project path (`~/projects/ticketly/ms-scheduling/.env`)

### Required Environment Variables

```
AWS_SQS_SESSION_ON_SALE_URL=<SQS queue URL for on-sale events>
AWS_SQS_SESSION_CLOSED_URL=<SQS queue URL for closed events>
SCHEDULER_CLIENT_SECRET=<Client secret for authentication>
AWS_REGION=<AWS region, default: ap-south-1>
```

### Optional Environment Variables

```
AWS_LOCAL_ENDPOINT_URL=<URL for local AWS endpoint, used for development>
EVENT_SERVICE_URL=<URL for event service, default: http://localhost:8081/api/event-seating>
KEYCLOAK_URL=<URL for Keycloak, default: http://auth.ticketly.com:8080>
KEYCLOAK_REALM=<Keycloak realm, default: event-ticketing>
KEYCLOAK_CLIENT_ID=<Keycloak client ID, default: scheduler-service-client>
KAFKA_URL=<Kafka broker URL, e.g. localhost:9092>
KAFKA_TOPIC=<Kafka topic for Debezium events, e.g. dbz.ticketly.public.event_sessions>
```

## Notes
- Internal packages keep implementation details hidden from external consumers.
- AWS and app config packages are aliased to avoid name collision (`awsconfig` vs `appconfig`).
- Further enhancements could include: context cancellation, structured logging, retry/backoff abstraction, unit tests with interfaces for SQS & HTTP clients.
