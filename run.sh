#!/usr/bin/env bash
# Meant to be run on Termux (Android).

echo "Build:"
echo "  CGO_ENABLED=0 go build -o borum-api ."
echo ""
echo "Start the server:"
echo "  ./borum-api serve"
echo ""
echo "Start the ephemeral tunnel (another window):"
echo "  cloudflared tunnel --url http://127.0.0.1:8091"
