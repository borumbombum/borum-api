#!/usr/bin/env bash
# Meant to be run on Termux (Android).
# Ctrl+C stops the tunnel and the server.
# Manually: pkill -x borum-api; pkill -x cloudflared
set -euo pipefail

CGO_ENABLED=0 go build -o borum-api ./cmd/api
./borum-api serve &
SERVER_PID=$!
trap 'kill $SERVER_PID 2>/dev/null' EXIT

cloudflared tunnel --url http://127.0.0.1:8091
