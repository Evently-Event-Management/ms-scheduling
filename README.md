# ms-scheduling

Refactored for maintainability by splitting monolithic `main.go` into internal packages.

## Structure

- `main.go` – application entrypoint and main loop.
- `internal/config` – configuration loading from environment variables.
- `internal/models` – shared data models (`SQSMessageBody`, `DebeziumEvent`).
- `internal/auth` – Keycloak client credentials token retrieval and user information access.
- `internal/sqsutil` – SQS helper utilities (receive & delete messages).
- `internal/session` – business logic for processing session state changes.
- `internal/kafka` – Kafka consumer for processing Debezium events.
- `internal/scheduler` – AWS EventBridge scheduler operations for event scheduling.

## Build & Run

### Local Development
```bash
go build ./...
go run .
```

### Testing User Email Retrieval
```bash
# Test retrieving a user's email by ID
go run main.go -test-user <user-id>

# Example
go run main.go -test-user aa6bbdd9-6c98-49f8-ac72-c43bbc6a4269
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
AWS_SQS_SESSION_SCHEDULING_URL=<SQS queue URL for all session scheduling events>
AWS_SQS_SESSION_SCHEDULING_ARN=<SQS queue ARN for all session scheduling events>
AWS_SCHEDULER_ROLE_ARN=<IAM role ARN for EventBridge Scheduler to access SQS>
AWS_SCHEDULER_GROUP_NAME=<EventBridge Scheduler group name>
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

## Authentication Features

### Keycloak Integration

The service integrates with Keycloak for authentication and user information retrieval:

1. **M2M (Machine-to-Machine) Authentication**
   - Uses client credentials grant flow to obtain access tokens
   - Required for accessing protected endpoints and Keycloak Admin APIs

2. **User Information Retrieval**
   - `GetUserEmailByID(cfg, client, userID)` - Retrieves a user's email address by their Keycloak user ID
   - Requires the client to have the "view-users" role from realm-management client

#### Example Usage

```go
import (
    "net/http"
    "time"
    "ms-scheduling/internal/auth"
    "ms-scheduling/internal/config"
)

// Create HTTP client and load config
httpClient := &http.Client{Timeout: 10 * time.Second}
cfg := config.Load()

// Retrieve a user's email by ID
userID := "aa6bbdd9-6c98-49f8-ac72-c43bbc6a4269"
email, err := auth.GetUserEmailByID(cfg, httpClient, userID)
if err != nil {
    // Handle error
    log.Printf("Error retrieving email: %v", err)
} else {
    log.Printf("User email: %s", email)
}
```

#### Required Keycloak Permissions

The service account must have the "view-users" role from the "realm-management" client to access user information. This can be configured in Keycloak or through Terraform:

```terraform
resource "keycloak_openid_client_service_account_role" "service_view_users" {
  realm_id                = keycloak_realm.realm_name.id
  service_account_user_id = keycloak_openid_client.client_name.service_account_user_id
  client_id               = data.keycloak_openid_client.realm_management.id
  role                    = "view-users"
}
```

## Notes
- Internal packages keep implementation details hidden from external consumers.
- AWS and app config packages are aliased to avoid name collision (`awsconfig` vs `appconfig`).
- Further enhancements could include: context cancellation, structured logging, retry/backoff abstraction, unit tests with interfaces for SQS & HTTP clients.
