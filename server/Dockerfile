# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static linking, GOOS=linux for cross-compilation if build env differs
# Output binary named 'keeper'
RUN go build -o keeper .

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# Create a non-root user and group
RUN addgroup -S keeper_group -g 1000 && adduser -S -u 1000 -G keeper_group -h /app keeper_user

# Copy the built binary from the builder stage
COPY --from=builder /app/keeper .

# Copy fixtures directory
ADD fixtures ./fixtures/

# Create a /data directory for potential database storage, owned by keeper_user
RUN mkdir -p /data && chown -R keeper_user:keeper_group /data
# Set ownership of the /app directory (where the binary and fixtures are)
RUN chown -R keeper_user:keeper_group /app

# Expose the port the application runs on (set by KEEPER_PORT, defaults to 8080 in main.go)
# This is for documentation; actual port exposure is in docker-compose.yml
EXPOSE 8080

# Switch to the non-root user
USER keeper_user

# Command to run the application
# The KEEPER_PORT and KEEPER_DB_PATH will be set via docker-compose.yml
CMD ["./keeper"]
