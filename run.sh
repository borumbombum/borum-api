#!/usr/bin/env bash
# Ctrl+C stops the tunnel and the server.
# Manually: pkill -x borum-api; pkill -x cloudflared
set -euo pipefail

CGO_ENABLED=0 go build -o borum-api ./cmd/api
./borum-api &
SERVER_PID=$!
trap 'kill $SERVER_PID 2>/dev/null; pkill -x cloudflared 2>/dev/null' EXIT

sleep 1
for i in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:8091/ >/dev/null 2>&1; then
    echo "borum-api ready on 127.0.0.1:8091"
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "borum-api died before becoming ready" >&2
    exit 1
  fi
  sleep 1
done

cloudflared tunnel --url http://127.0.0.1:8091 --config /dev/null