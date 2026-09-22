#!/bin/bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

# 1. Build server if needed
echo "[BUILD] Building UniHub Backend binary..."
(cd src/backend && go build -o bin/server ./cmd/server)

# 2. Export environment variables for FULL CPU execution
unset GOMAXPROCS
export APP_MODE=all
export DB_HOST=localhost
export DB_PORT=5433
export DB_USER=unihub
export DB_PASSWORD=unihub_secret
export DB_NAME=unihub_workshop
export DB_SSLMODE=disable
export REDIS_ADDR=localhost:6379
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
export SERVER_PORT=8080
export AUTH_SECRET=default-secret
export SMTP_HOST=localhost
export SMTP_PORT=1025

echo "[START] Launching backend with ALL $(nproc) CPU cores..."
exec ./src/backend/bin/server

