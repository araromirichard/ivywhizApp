# Define the Go version to use
ARG GO_VERSION=1.20

# Stage 1: Build the Go application
FROM golang:${GO_VERSION}-bookworm as builder

# Set the working directory inside the container
WORKDIR /usr/src/app

# Copy go.mod and go.sum, then download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the rest of the application code
COPY . .

# Build the Go application, outputting to bin/linux_amd64/api
RUN GOOS=linux GOARCH=amd64 go build -v -o ./bin/linux_amd64/api ./cmd/api

# Stage 2: Create a smaller runtime image
FROM debian:bookworm

# Set the working directory inside the container
WORKDIR /root/

# Copy the pre-built binary from the builder stage to the runtime image
COPY --from=builder /usr/src/app/bin/linux_amd64/api /usr/local/bin/

# Set the command to run the binary
CMD ["api"]
