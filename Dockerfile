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

# Copy the built binary from the builder stage
COPY --from=builder /scheduler-service /scheduler-service

# Set the command to run the application
ENTRYPOINT ["/scheduler-service"]
