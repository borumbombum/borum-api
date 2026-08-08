# borum-api

Single Go binary: a custom chi HTTP API. The data layer (Turso) is wired in
separately; right now the API runs with no database.
Custom routes are declared in `cmd/api/routes.go` (apiRoutes table).

## Code layout

cmd/api/main.go   entry point: HTTP server bootstrap, graceful shutdown on SIGINT/SIGTERM
cmd/api/routes.go route table (apiRoutes) + chi router — single source of truth for endpoints
cmd/api/handlers.go HTTP handlers as methods on *app (injected dependencies)
cmd/api/server.go  startup banner
internal/tasks/    minimal in-process scheduler for background jobs

## Port
- 8091 — custom chi API (the only thing you expose)

## Build & run on Termux
    pkg install golang git tmux
    git clone https://github.com/borumbombum/borum-api.git
    cd borum-api
    CGO_ENABLED=0 go build -o borum-api ./cmd/api
    ./borum-api

Binds 127.0.0.1:8091 (API port is configured in cmd/api/main.go, const apiPort).

Keep it alive with tmux (pkg install tmux).

## Access
- Custom API (8091): http://<host>:8091

## Expose to the internet (Cloudflare Tunnel)
Only the custom API (8091) is exposed.

Install: pkg install cloudflared tmux

**Quick (ephemeral, no account):**
    cloudflared tunnel --url http://127.0.0.1:8091
→ random https://xxx.trycloudflare.com. New URL each restart, no SLA.

**Named (persistent, needs a Cloudflare-managed domain):**
1. Dashboard → Zero Trust → Networks → Tunnels → Create → copy token
2. Add hostname (e.g. api.yourdomain.com) → http://127.0.0.1:8091
3. cloudflared tunnel run --token <TOKEN>

Keep the server + tunnel alive with tmux (Ctrl-B C new window,
Ctrl-B D detach).

## Keep Termux from sleeping (Android)

If your Termux CPU sleeps/suspends and the API stops, hold a Termux wake lock while the
server is running.

In one Termux session:
```sh
termux-wake-lock
```