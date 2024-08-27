# Stage 1: Build the Go application
ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm AS builder

# Set the working directory
WORKDIR /usr/src/app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the application source code into the container
COPY . .

# Build the Go application
WORKDIR /usr/src/app/cmd/api
RUN go build -v -o /run-app .

# Stage 2: Run the migrations (optional)
FROM golang:${GO_VERSION}-bookworm AS migrator

WORKDIR /usr/src/app

# Copy the built application and source code from the builder stage
COPY --from=builder /usr/src/app .

# Install the migration tool if needed
# RUN go install -tags 'migrate' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run the migrations
WORKDIR /usr/src/app/cmd/api
RUN migrate -path /usr/src/app/migrations -database "${IVYWHIZ_DB_DSN}" up

# Stage 3: Create the final runtime image
FROM alpine:latest

# Install any required packages (e.g., SSL certificates)
RUN apk add --no-cache ca-certificates

# Copy the compiled application from the builder stage
# COPY --from=builder /run-app /usr/local/bin/

# Expose the application's port (if needed)
EXPOSE 8080

# Run the application
CMD ["run-app"]
