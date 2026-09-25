#!/bin/bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$DIR"

USERS=${1:-50}
SLOTS=${2:-25}
CYCLES=${3:-1}
MODE=${4:-gate}

echo "[BENCH] Compiling concurrency_demo binary..."
(cd src/backend && go build -o bin/concurrency_demo ./cmd/concurrency_demo)

echo "[BENCH] Executing benchmark: Users=$USERS, Slots=$SLOTS, Mode=$MODE, Cycles=$CYCLES"
./src/backend/bin/concurrency_demo \
  -users "$USERS" \
  -slots "$SLOTS" \
  -cycles "$CYCLES" \
  -mode "$MODE" \
  -dbport 5433
