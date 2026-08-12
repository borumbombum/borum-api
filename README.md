# borum-api

Single Go binary: a custom chi HTTP server that serves both the API and the
web site. The data layer (Turso/libsql) is wired in and configured via
environment variables (see `.env.example`).
Custom routes are declared in `cmd/web/routes.go` (apiRoutes table).

## Setup

    cp .env.example .env

Then fill in `TURSO_URL` and `TURSO_TOKEN` with your Turso database values.

## Run

    go run ./cmd/web

## Code layout

cmd/web/main.go    entry point: HTTP server bootstrap, graceful shutdown on SIGINT/SIGTERM
cmd/web/routes.go  route table (apiRoutes) + chi router — single source of truth for endpoints
cmd/web/handlers.go HTTP handlers as methods on *app (injected dependencies)
cmd/web/web.go    web site views: template loading, page handlers, shared view model
cmd/web/templates/ Go templates: base, header, footer, home, article, tag, 404
cmd/web/server.go startup banner
data/            site data: articles.json (loaded at startup; served as /data/articles.json for the command palette)
internal/content/  site data accessors and shapes; principles stay embedded, articles load from data/articles.json
internal/battery/  system battery snapshot used by the header
internal/tasks/    minimal in-process scheduler for background jobs

## Styling

The site uses plain CSS. Styles are
split into subject files under `static/css/` for editing, then concatenated in
cascade order at server startup and served as a single `/styles.css` by
`concatCSS()` in `cmd/web/web.go`. The browser makes one CSS request; edits to
`static/css/*.css` apply on the next restart.

- `static/css/theme.css` — design tokens: custom properties (`:root`) and light/dark theme overrides.
- `static/css/base.css` — resets, base typography, wrappers, code/pre, backgrounds, images.
- `static/css/components.css` — nav/header, footer, buttons, pill, archive, principles,
  command palette, hero headings, heart button, highlight marks.
- `static/css/prose.css` — article body typography and tables.
- `static/css/animations.css` — keyframes and view transitions.
- `static/css/breakpoints.css` — the responsive media queries (mobile/desktop breakpoints).
- `static/css/utilities.css` — utility classes used by the templates and `app.js`.

The concat order defines the cascade: theme → base → components → prose →
animations → breakpoints → utilities.

## Port

The API listens on `127.0.0.1:8091` by default. Configure it with flags on the
binary:

```
./borum-api -port=8092
./borum-api -address=0.0.0.0 -port=8092
```

or with env vars when using `./deploy.sh`:

```
PORT=8092 ./deploy.sh
ADDRESS=0.0.0.0 PORT=8092 ./deploy.sh
```

## Build & run on Termux

    pkg install golang git tmux
    git clone https://github.com/borumbombum/borum-api.git
    cd borum-api
    CGO_ENABLED=0 go build -o borum-api ./cmd/web
    ./borum-api

Binds 127.0.0.1:8091 by default. The listen address and port are flags defined
in cmd/web/main.go (`-address`, `-port`), overridable as shown in "Port" above.

Keep it alive with tmux (pkg install tmux).

## Access

- Custom API (8091): http://<host>:8091

## Expose to the internet (Cloudflare Tunnel)

Only the custom API (8091) is exposed.
The tunnel origin below must match whatever address/port the server actually
listens on (default 127.0.0.1:8091).

Install: pkg install cloudflared tmux

**Quick (ephemeral, no account):**
    cloudflared tunnel --url <http://127.0.0.1:8091>
→ random <https://xxx.trycloudflare.com>. New URL each restart, no SLA.

**Named (persistent, needs a Cloudflare-managed domain):**

1. Dashboard → Zero Trust → Networks → Tunnels → Create → copy token
2. Add hostname (e.g. api.yourdomain.com) → <http://127.0.0.1:8091>
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

## Connect to phone

```
ssh -p 8022 [IP]
```
