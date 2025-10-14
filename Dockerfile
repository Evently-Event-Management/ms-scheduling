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
# Use a standard minimal image for the final container
FROM alpine:3.19

# Install curl for health checks and ca-certificates for HTTPS
RUN apk add --no-cache curl ca-certificates

# Create a non-root user for security
RUN adduser -D -H -h /app appuser

# Set the working directory
WORKDIR /app

# Environment variables will be supplied at runtime
# Defaults are managed in the Go application's config.go file

# Copy the built binary from the builder stage
COPY --from=builder /scheduler-service .

# Copy the migrations folder from the builder stage
COPY --from=builder /app/migrations ./migrations

# Switch to the non-root user
USER appuser

EXPOSE 8085

# Add the health check instruction using the installed curl binary
# NOTE: Update '/health' to your Go application's actual health endpoint.
HEALTHCHECK --interval=30s --timeout=10s --retries=5 \
  CMD curl -f http://localhost:8085/api/scheduler/health || exit 1

# Set the command to run the application
ENTRYPOINT ["./scheduler-service"]

