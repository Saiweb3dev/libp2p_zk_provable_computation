FROM golang:1.23-alpine AS builder

# Install git for go mod download
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN go build -o /app/bin/bootstrap ./cmd/bootstrap/
RUN go build -o /app/bin/peer ./cmd/peer/

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/ ./bin/
COPY --from=builder /app/ ./

# Create a script to choose which binary to run
RUN echo '#!/bin/sh\nif [ "$1" = "bootstrap" ]; then\n  exec ./bin/bootstrap "${@:2}"\nelse\n  exec ./bin/peer "$@"\nfi' > /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]