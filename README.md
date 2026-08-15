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
cmd/web/middleware.go shared middleware: security headers, session peek, no directory listings
cmd/web/experiments_handlers.go the /experiments routes: image conversion (capped + rate-limited)
cmd/web/web.go    web site views: template loading, page handlers, shared view model
cmd/web/templates/ Go templates: base, header, footer, home, article, tag, 404, experiments
cmd/web/server.go startup banner
internal/db/      schema migrations (embedded SQL, applied at startup, tracked in schema_migrations)
internal/content/  site data accessors and shapes; articles come from the Turso database via a
                    61-minute in-memory cache (per-article cache capped at ~30MB, LRU eviction),
                    principles stay embedded
internal/battery/  system battery snapshot used by the header
internal/experiments/ experiment pages and image-conversion wiring (front matter, intro, progress)
internal/imgconv/  image decoding/encoding helpers with pixel and edge caps
internal/ratelimit/ sliding-window rate limiter (login + image conversion), keys pruned when idle
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

### Loading indicator color

The top sweep bar that shows during page navigations is colored by two tokens in
`static/css/theme.css`: `--color-mauve` (light end of the sweep) and
`--color-mauve-deep` (dark end). Change those two values and restart the server
to restyle it. The bar's shape and animation live in `static/css/animations.css`
(`.borum-loader-pill` and the `loader-sweep` keyframes).

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

## Admin login (/god)

The site has a single-user admin area behind `/login` (email/password; no
signup). Credentials come from the environment:

- `ADMIN_EMAIL` — the only email that may log in.
- `ADMIN_PASSWORD_HASH` — a **bcrypt hash** of the password, not the password
  itself. Generate it once with the repo's helper:

      go run ./cmd/hashpass 'your-password'

  Copy the whole `$2a$...` string into `.env` **wrapped in single quotes** —
  unquoted, the `$` characters are read as variable references and silently
  stripped, which makes every login fail. Any bcrypt tool works; the helper
  reads from stdin too (keeps the password out of shell history).
- `SESSION_TTL_HOURS` — optional session lifetime in hours (default 30 days).

Sessions are stored in the `sessions` table (only SHA-256 token hashes; the
random token itself lives in an HttpOnly, SameSite=Lax cookie, marked `Secure`
when served over HTTPS — the proxy's `X-Forwarded-Proto` header is trusted only
from loopback peers, so a directly exposed server can't be spoofed). Login
attempts are rate-limited, the `/god` forms are protected with per-session CSRF
tokens, and `/god` + `/login` send `X-Frame-Options: DENY`, `nosniff` and
`no-store`. The image-conversion endpoint is per-client rate-limited and rejects
uploads above 40 megapixels or a 10,000px edge before decoding.

Signed in, an "edit →" link appears on `/blog/{slug}`, and `/god/articles`
lists every article with create/edit forms. The code is built for more login
methods (Nostr) via the `Method` interface in `internal/auth`.

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
