# SECURITY — borum-api audit

Audit date: August 15, 2026. Re-verified: August 17, 2026 (draft system and
Portuguese i18n landed since; Bugs #3 and #4 below updated to match). Scope:
full `borum-api` codebase (auth, routes, handlers, templates, static assets,
DB layer, scheduler).

## What's already solid

- Session tokens: 256-bit random, only SHA-256 hashes stored in the DB,
  HttpOnly + SameSite=Lax cookie.
- Passwords: bcrypt-hashed via `ADMIN_PASSWORD_HASH`, generic error messages
  (no user enumeration), constant-time email compare.
- CSRF: stateless HMAC tokens on every `/god` write, verified constant-time.
- SQL: all parameterized queries — no SQL injection found.
- Upload cap (`MaxBytesReader` 25 MB), timeouts on the server, graceful
  shutdown, panic recovery.
- Image conversion is bounded: `image.DecodeConfig` rejects >40 MP or >10k
  edge before any decode, plus a per-client rate limit (10/min).
- Rate limiting keys on the real client IP (first `X-Forwarded-For` entry) and
  prunes idle keys, so it survives the tunnel and cannot leak memory.
- Security headers site-wide (`nosniff`, `Referrer-Policy`) and on `/god` +
  `/login` (`X-Frame-Options: DENY`, `Cache-Control: no-store`).
- Session cookie `Secure` trusts `X-Forwarded-Proto` only from a loopback
  proxy, so a directly exposed server cannot be spoofed.

---

## Security vulnerabilities

### 🟠 Medium

**1. Stored XSS by design — with no output sanitization**

Article bodies and experiment intros are rendered raw via
`{{safe .Article.Body}}` / `{{safe .Experiment.Intro}}`. Admin-only input, so it
is safe *only* while the session + CSRF hold. Since the CSRF token sits in the
page, any clickjacking or session leak becomes full stored XSS for every
visitor. Portuguese translations (added after this audit) render through the
same raw `{{safe}}` paths, so they carry the same risk. Sanitize on save (e.g.
`bluemonday`) or accept the risk knowingly.

**2. CDN scripts without SRI (supply-chain risk)**

Tailwind browser@4 and Google Fonts load from `jsdelivr`/`googleapis` with no
integrity attributes. If the CDN is compromised, arbitrary JS runs on the site.
Also: shipping Tailwind's *browser compiler* to production is a heavy runtime
payload — self-host the generated CSS instead.

### 🟡 Low

**1. Logout POST has no CSRF** — a cross-site form can force a logout
(nuisance).

**2. `hashpass` — a compiled binary — is committed to git. Remove it and
gitignore `/hashpass`.**

**3. No `robots.txt`, no `Server` header stripping, no tests, no CI, no
`govulncheck` run on the (very new, pseudo-versioned) deps.**

---

## Bugs & improvements (non-security)

1. **Double session DB round-trip** on every admin request: `Peek` resolves the
   session, then `RequirePage` resolves it again. Resolve once.
2. **DB hit per static asset while logged in**: `Peek` runs router-wide, so
   every `/app.js`, `/styles.css`, image request triggers a remote Turso
   session lookup. Skip session resolution for static paths.
3. **`last_seen` written only at session creation** — updated 2026-08-17: the
   insert now sets `last_seen = datetime('now')`, but nothing refreshes it on
   activity. Still no sliding session renewal.
4. **Cache mutexes held during remote DB loads** (`ttlCache.get`,
   `articleCache.get`) — updated 2026-08-17: articles now use a per-slug TTL
   cache with LRU eviction (`articleCache`), but one global mutex is still held
   during the remote `queryArticle` load, so reads still serialize against the
   *remote* DB. Use singleflight/per-key locking.
5. **Shutdown order is backwards**: `sqlDB.Close()` runs *before*
   `srv.Shutdown()`, so in-flight requests during shutdown hit a closed DB.
   Shut down the server first.
6. **Server-side validation gaps**: slug `[a-z0-9-]+` is only enforced
   client-side; `date` format is not validated server-side (a bad date silently
   reorders the whole archive); tags are not validated.
7. **Not actually a single binary**: templates and `static/` load from the
   working directory (`templateDir`, `http.Dir("static")`), so it only runs
   from the repo root. `go:embed` would make the README's "single Go binary"
   claim true.
8. **Heart button is decorative** — localStorage only, nothing persists
   server-side (`initial_love` is baked into the page). Fine if intended; the
   README even hints it might be "broke."
9. Hardcoded "1 min read" on every article; health endpoint does a DB ping per
   request (cache it).

---

## Post-audit changes (August 15–17, 2026)

- Draft system, syntax highlighting, code-block copy buttons, article copy,
  and editor auto-save. No new security findings beyond what is listed above.
- Portuguese i18n: language-prefixed routes (`/pt/...`), translation tables,
  `hreflang` tags, JSON-LD. Translations render through the same raw
  `{{safe}}` paths as English content — see Medium #1. In-progress parts
  (`internal/i18n/`, migration `0007_add_i18n.sql`) are uncommitted.
- `last_seen` and the article cache changed — see Bugs #3 and #4.

---

## Highest-value fixes

The five highest-value fixes from the audit were implemented and verified on
2026-08-15:

1. Image endpoint: pixel caps (40 MP / 10k edge) + per-client rate limiting.
2. Rate limiter: keyed on `X-Forwarded-For` (real client IP), keys pruned.
3. Security headers on `/god` + `/login` (`X-Frame-Options`, `nosniff`,
   `no-store`).
4. `initial_love` preservation bug (love count survives article edits).
5. `Secure` cookie: proxy header trusted only from loopback peers.

Also fixed while in there: unbounded login body, `/api/health` info/resource
leak and `latancy` typo, static directory listings, `validCSRF` dead argument.
