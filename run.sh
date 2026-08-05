#!/usr/bin/env bash
# Meant to be run on Termux (Android).
# Builds borum-api, starts the server + Cloudflare ephemeral tunnel in a
# tmux session, then prints the public URL.
set -euo pipefail

CGO_ENABLED=0 go build -o borum-api .

pkill -x borum-api 2>/dev/null || true
pkill -x cloudflared 2>/dev/null || true

tmux kill-session -t borum 2>/dev/null || true

tmux new-session -d -s borum -n api './borum-api serve; read'
tmux new-window  -t borum -n tunnel 'cloudflared tunnel --url http://127.0.0.1:8091 > /tmp/borum-tunnel.log 2>&1; read'

URL=""
for i in $(seq 1 60); do
  URL=$(grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' /tmp/borum-tunnel.log 2>/dev/null | head -1 || true)
  [ -n "$URL" ] && break
  sleep 1
done

echo "Server: http://127.0.0.1:8091"
[ -n "$URL" ] && echo "Public: $URL" || echo "Tunnel URL not found — see /tmp/borum-tunnel.log"
echo "Attach:  tmux attach -t borum   (detach: Ctrl-B D)"
