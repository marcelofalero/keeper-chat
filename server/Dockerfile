# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy fixtures directory
# This needs to be done before the general COPY . . if the fixtures are within the build context's server/ directory
# and if the path used by the application is relative to the WORKDIR /app
COPY server/fixtures ./server/fixtures/

# Copy the rest of the source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static linking, GOOS=linux for cross-compilation if build env differs
# Output binary named 'keeper'
RUN go build -o keeper .

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/keeper .

# Expose the port the application runs on (set by KEEPER_PORT, defaults to 8080 in main.go)
# This is for documentation; actual port exposure is in docker-compose.yml
EXPOSE 8080

# Command to run the application
# The KEEPER_PORT and KEEPER_DB_PATH will be set via docker-compose.yml
CMD ["./keeper"]
