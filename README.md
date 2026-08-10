# borum-api

Single Go binary: a custom chi HTTP API. The data layer (Turso/libsql) is wired in
and configured via environment variables (see `.env.example`).
Custom routes are declared in `cmd/api/routes.go` (apiRoutes table).

## Setup

    cp .env.example .env

Then fill in `TURSO_URL` and `TURSO_TOKEN` with your Turso database values.

## Code layout

cmd/api/main.go   entry point: HTTP server bootstrap, graceful shutdown on SIGINT/SIGTERM
cmd/api/routes.go route table (apiRoutes) + chi router — single source of truth for endpoints
cmd/api/handlers.go HTTP handlers as methods on *app (injected dependencies)
cmd/api/server.go  startup banner
internal/tasks/    minimal in-process scheduler for background jobs

## Port
The API listens on `127.0.0.1:8091` by default. Configure it with flags on the
binary:
```
./borum-api -port=8092
./borum-api -address=0.0.0.0 -port=8092
```
or with env vars when using `run.sh`:
```
PORT=8092 ./run.sh
ADDRESS=0.0.0.0 PORT=8092 ./run.sh
```

## Build & run on Termux
    pkg install golang git tmux
    git clone https://github.com/borumbombum/borum-api.git
    cd borum-api
    CGO_ENABLED=0 go build -o borum-api ./cmd/api
    ./borum-api

Binds 127.0.0.1:8091 by default. The listen address and port are flags defined
in cmd/api/main.go (`-address`, `-port`), overridable as shown in "Port" above.

Keep it alive with tmux (pkg install tmux).

## Access
- Custom API (8091): http://<host>:8091

## Expose to the internet (Cloudflare Tunnel)
Only the custom API (8091) is exposed.
The tunnel origin below must match whatever address/port the server actually
listens on (default 127.0.0.1:8091).

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