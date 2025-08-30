# Stage 1: Build
FROM golang:1.23.4-alpine AS builder

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

# Note: No worker binary in this repo; skip building worker

# Stage 1.5: Dev with hot reload (Air)
FROM golang:1.23.4-alpine AS dev

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

# Install dev tools (Air)
RUN apk add --no-cache git curl && \
    go install github.com/air-verse/air@latest

# Cache deps first
COPY go.mod go.sum ./
RUN go mod download

# Bring in source (overridden by bind-mount in docker-compose dev)
COPY . .

EXPOSE 8088 9090

CMD ["/go/bin/air", "-c", ".air.toml"]

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
