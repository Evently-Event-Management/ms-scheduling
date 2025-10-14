# --- Build Stage ---
# Use the official Go image as a builder
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.* ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application, creating a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /scheduler-service .

# --- Final Stage ---
# Use a minimal, non-root image for the final container
FROM gcr.io/distroless/static-debian12

# Environment variables with default values
ENV AWS_REGION="ap-south-1" \
    EVENT_SERVICE_URL="http://localhost:8081/api/event-seating" \
    KEYCLOAK_URL="http://auth.ticketly.com:8080" \
    KEYCLOAK_REALM="event-ticketing" \
    KEYCLOAK_CLIENT_ID="scheduler-service-client" \
    KAFKA_URL="" \
    KAFKA_TOPIC=""

# Required environment variables (these need to be provided at runtime)
# AWS_SQS_SESSION_SCHEDULING_URL
# AWS_SQS_SESSION_SCHEDULING_ARN
# AWS_SCHEDULER_ROLE_ARN
# AWS_SCHEDULER_GROUP_NAME
# SCHEDULER_CLIENT_SECRET

# Copy the built binary from the builder stage
COPY --from=builder /scheduler-service /scheduler-service

EXPOSE 8085

# Set the command to run the application
ENTRYPOINT ["/scheduler-service"]
