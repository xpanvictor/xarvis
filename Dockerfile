# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Set Go env
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set working dir
WORKDIR /app

# Copy go.mod + go.sum first (caching deps)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build API binary
RUN go build -o bin/api ./cmd/api/main.go

# Build Worker binary
RUN go build -o bin/worker ./cmd/worker/main.go

# Stage 2: Minimal runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binaries
COPY --from=builder /app/bin /app/bin

# Optional: copy configs
COPY internal/config /app/config

# Expose port for API & QT
EXPOSE 8080 9090

# Default entrypoint (can be overridden in docker-compose)
ENTRYPOINT ["/app/bin/api"]