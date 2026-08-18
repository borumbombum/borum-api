Status: [TODO]

# RSS Feed

## Context

Readers and feed aggregators need a way to subscribe to new articles.
Adding a standard RSS 2.0 feed at `/rss` lets them do that without any
third-party service. The feed is built from published articles already
stored in the database.

## Requirements

1. **RSS 2.0 XML** — Standard `<rss>` / `<channel>` / `<item>` structure.
2. **Route** — `GET /rss` returns the feed. No auth required.
3. **Channel metadata** — Title, link (`https://{host}`), description,
   language (`en`), lastBuildDate.
4. **Items** — One `<item>` per published article, ordered by date
   descending. Each item includes: title, `<link>` (permalink),
   `<pubDate>` (RFC 822), `<description>` (excerpt), `<guid>` (permalink),
   `<category>` per tag.
5. **Handler** — New file `cmd/web/rss_handler.go` with
   `func (a *app) rssHandler(w http.ResponseWriter, r *http.Request)`.
6. **Data source** — `content.List()` (already cached, returns only
   published articles). No new database queries.
7. **Content-Type** — `application/rss+xml; charset=utf-8`.
8. **No new dependencies** — Use `encoding/xml` from stdlib.
9. **RSS discovery** — Add `<link rel="alternate" type="application/rss+xml"
   title="RSS" href="https://{host}/rss" />` in `base.html` `<head>`,
   after the `{{template "meta" .}}` block. This makes the feed
   discoverable from every page.
10. **Footer link** — Add an RSS icon/link in `footer.html` next to the
    existing Nostr link, using the Lucide `rss` icon (already loaded),
    pointing to `/rss`.

## Files

- `cmd/web/rss_handler.go` — new handler
- `cmd/web/routes.go` — add route
- `cmd/web/templates/base.html` — RSS discovery `<link>` in `<head>`
- `cmd/web/templates/footer.html` — RSS icon/link

## Acceptance criteria

- `go build ./...` passes
- `GET /rss` returns valid RSS 2.0 XML
- Feed contains all published articles ordered by date descending
- Each item has title, link, pubDate, description, guid, categories
- `<link rel="alternate" ...>` appears in every page's `<head>`
- Footer shows a clickable RSS icon linking to `/rss`
