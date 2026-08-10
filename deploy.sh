#!/usr/bin/env bash
# Ctrl+C stops the tunnel and the server.
# Manually: pkill -x borum-api; pkill -x cloudflared
set -euo pipefail

# Listen address and port default to 127.0.0.1:8091. Override with env vars:
#   PORT=8092 ./deploy.sh
#   ADDRESS=0.0.0.0 PORT=8092 ./deploy.sh
ADDRESS="${ADDRESS:-127.0.0.1}"
PORT="${PORT:-8091}"
BASE="http://${ADDRESS}:${PORT}"

CGO_ENABLED=0 go build -o borum-api ./cmd/web
./borum-api --address="$ADDRESS" --port="$PORT" &
SERVER_PID=$!
trap 'kill $SERVER_PID 2>/dev/null; pkill -x cloudflared 2>/dev/null' EXIT

sleep 1
for i in $(seq 1 30); do
  if curl -fsS "$BASE/api/health" >/dev/null 2>&1; then
    echo "borum-api ready on $BASE"
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "borum-api died before becoming ready" >&2
    exit 1
  fi
  sleep 1
done

cloudflared tunnel --url "$BASE" --config /dev/null