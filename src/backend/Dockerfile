# ==========================================
# Backend Dockerfile — Production-optimized
# ==========================================

# Stage 1: Build binary
FROM golang:1.22-alpine AS builder

RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /app

# Cache dependencies layer
COPY go.mod go.sum ./
RUN go mod download

# Build statically linked binary, strip debug symbols
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/server ./cmd/server/main.go

# Stage 2: Minimal runtime
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

# Security: Non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy only the binary (migrations are embedded via go:embed)
COPY --from=builder /app/server .

# Copy static data needed at runtime
COPY data/ data/

# Set ownership to non-root user
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

# ENTRYPOINT receives SIGTERM from K8s correctly for graceful shutdown
ENTRYPOINT ["./server"]
