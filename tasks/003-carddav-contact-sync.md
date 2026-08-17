Status: [TODO]

# CardDAV Contact Sync (Sovereign Backup)

## Context

Need to sync contacts between phone (iOS/Android) and the borum-api server for sovereign backup — no third-party services, full control over contact data. Only contacts syncing is needed (no calendars).

The existing Go/chi web server and Turso DB are ideal:
- CardDAV is built on HTTP/WebDAV (chi handles routing)
- Turso DB stores vCard payloads alongside sync metadata
- `emersion/go-webdav` library provides standard CardDAV implementation

## Security Considerations

1. **Separate app-scoped credential** — Don't reuse admin password for CardDAV Basic Auth. Generate random token, store hashed in .env (like `ADMIN_PASSWORD_HASH`), scoped to CardDAV only.
2. **HTTPS enforcement** — Basic Auth = plaintext-equivalent. Enforce at middleware (reject non-TLS, check X-Forwarded-Proto), add HSTS.
3. **Rate-limit DAV endpoints** — Basic Auth has no CSRF/session friction. chi throttling middleware protects against credential stuffing.
4. **Tenant isolation via joins** — Even for single-user, scope queries through `address_books.user_id`. Cheap insurance.
5. **Size-limit PUT bodies** — Prevent DoS via huge vCards with embedded photos. Simple middleware check.
6. **ETags as real hash (SHA-256)** — Content-based ETags for correct If-Match/If-None-Match preconditions.
7. **Don't log PII** — Never log Basic Auth headers or full vCard bodies (addresses, birthdays, notes).

## Simplicity Considerations

1. **Skip sync-tokens initially** — Sync-tokens require tombstone tracking (deleted_at column, retention window). Start with addressbook-multiget + PROPFIND/etag-based sync. Fast-follow later.
2. **Drop MKCOL** — Single-user sovereign backup doesn't need clients creating collections. Pre-create one default address book in migration (not app startup). Reduces attack surface.
3. **Soft-delete with deleted_at** — When sync-tokens are added later, simpler than parallel tombstone table. Prune on schedule.
4. **Single implicit user** — Keep `user_id` column with `DEFAULT 'default'` for future flexibility. Auth middleware hardcodes `userID = "default"`. Every Backend method derives userID from authenticated principal. Every query includes `AND user_id = ?` even though today it's always 'default'.

## Requirements

### 1. **CardDAV Backend**
- Implement `carddav.Backend` interface from `github.com/emersion/go-webdav/carddav`
- Support addressbook-multiget + PROPFIND/etag-based sync
- Skip sync-tokens initially (fast-follow later)
- Skip MKCOL (pre-create default address book in migration)
- REPORT method supports addressbook-multiget only, NOT sync-collection (deferred)
- Every query defaults to `WHERE deleted_at IS NULL` (soft-deleted rows excluded)
- Every query includes `AND user_id = ?` (hardcoded to 'default' for now)

### 2. **Database Schema**
- `address_books` table: id, user_id (DEFAULT 'default'), name, description, created_at
- `contacts` table: id (card UID), address_book_id, path, vcard_data, etag, updated_at, deleted_at (soft-delete)
- Indexes on path and address_book_id

### 3. **HTTP Endpoints**
- Mount `carddav.Handler` on `/carddav/*`
- Support WebDAV methods: PROPFIND, REPORT (addressbook-multiget only), PUT, DELETE
- Well-known redirect: `/.well-known/carddav` → `/carddav/`
- Size-limit PUT bodies (middleware)
- Rate-limit DAV endpoints (chi throttling)
- Default address book pre-created in migration (not app startup)

### 4. **Authentication**
- Separate app-scoped credential (random token, hashed in .env like `ADMIN_PASSWORD_HASH`)
- HTTP Basic Auth middleware (required by DAV clients)
- Auth middleware hardcodes `userID = "default"` (single admin user from .env)
- HTTPS enforcement at middleware (reject non-TLS, check X-Forwarded-Proto)
- HSTS headers
- Don't log Basic Auth headers or vCard PII

### 5. **Download/Upload Support**
- Phone can upload contacts (full sync)
- Phone can download all contacts
- Server acts as CardDAV server (not client)

### 6. **Admin Dashboard (`/god/carddav`)**
- Display CardDAV stats (contact count, last sync, address book info)
- Show dynamic CardDAV URL for client configuration (e.g., `https://{{.Host}}/carddav/`)
- URL uses `{{.Host}}` from page context (same pattern as hreflang tags) — never hardcoded
- Upload button: import vCard file(s) from admin interface
- Download button: export all contacts as vCard file
- Minimal interface — just stats, URL display, and buttons for now
- Full web interface for contact management is a separate future task

## Acceptance criteria

- iOS/Android CardDAV client can connect and sync contacts
- Contacts created on phone appear on server
- Contacts created on server appear on phone
- ETags change on modification for efficient sync (SHA-256 hash)
- Well-known redirect works for client auto-discovery
- Basic Auth required for all DAV endpoints
- Separate app-scoped credential (not admin password, hashed in .env)
- HTTPS enforced at middleware level
- Rate-limiting on DAV endpoints
- `/god/carddav` shows stats and has upload/download buttons
- `go vet` passes; no security vulnerabilities introduced
- Documentation created in `docs/carddav.md`

## Progress

- 2026-08-17: Task opened. Scope: contacts only (no calendars).
  Focus on sovereign backup — self-hosted, full control over data.
  Admin page at `/god/carddav` with stats and upload/download.
  Security: separate credential, HTTPS enforcement, rate-limiting, PII logging.
  Simplicity: skip sync-tokens/MKCOL initially, single user, soft-delete.
  Final: hardcode user_id='default', queries scope by user_id, default address book in migration.
