// Package content holds the blog and principles data. Articles live in the
// SQLite (Turso) database. Reads are targeted SQL queries whose results are
// cached in memory for a short time (see cacheTTL); principles live in the
// generated data_principles.go. This file defines the shapes and accessors.
package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"
)

// Article is a single blog post (full row, used by the article page).
type Article struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Subtitle     string   `json:"subtitle"`
	Date         string   `json:"date"`
	Tags         []string `json:"tags"`
	Excerpt      string   `json:"excerpt"`
	Image        string   `json:"image,omitempty"`
	ImageCaption string   `json:"imageCaption,omitempty"`
	InitialLove  int      `json:"initialLove"`
	Star         bool     `json:"star"`
	Featured     bool     `json:"featured"`
	Body         string   `json:"body"`
}

// ArticleSummary is the lightweight row used by the archive (home and tag)
// pages: everything except the heavy fields (body, excerpt, subtitle).
type ArticleSummary struct {
	Slug         string
	Title        string
	Date         string
	Tags         []string
	Star         bool
	Featured     bool
	Image        string
	ImageCaption string
}

// PaletteItem is the minimal shape served to the command palette.
type PaletteItem struct {
	Slug  string   `json:"slug"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

// Principle is one numbered maxim from the home page.
type Principle struct {
	N     int
	Title string
	Body  string
}

// cacheTTL is how long a cached query result stays fresh. Short enough that a
// manual edit in the database shows up quickly, long enough to keep repeat
// page loads off the remote database.
const cacheTTL = 61 * time.Minute

// maxArticleCacheBytes caps the total memory the per-slug article cache may
// occupy (an approximation). When the cap is hit the least-recently-used
// entry is evicted. Tune here; there is no env or flag for this.
const maxArticleCacheBytes = 30 << 20 // 30 MB

// store holds the database handle. It must be set with Init before use.
var store struct {
	mu sync.Mutex
	db *sql.DB
}

// Init wires the article store to the database. It must be called once at
// startup, after Migrate has run, before serving requests.
func Init(db *sql.DB) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.db = db
}

// ttlCache caches the result of a load for cacheTTL. A failed load returns
// the last good value (possibly empty) and the next call retries.
type ttlCache[T any] struct {
	mu     sync.Mutex
	value  T
	at     time.Time
	loaded bool
}

func (c *ttlCache[T]) get(ctx context.Context, load func(context.Context) (T, error)) T {
	c.mu.Lock()
	defer c.mu.Unlock()
	if store.db == nil || !c.loaded || time.Since(c.at) > cacheTTL {
		if v, err := load(ctx); err == nil {
			c.value = v
			c.at = time.Now()
			c.loaded = true
		}
	}
	return c.value
}

// reset marks the cache stale so the next get reloads from the database. It
// is called by Invalidate after an admin write.
func (c *ttlCache[T]) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loaded = false
}

var (
	summaries = ttlCache[[]ArticleSummary]{}
	palette   = ttlCache[[]PaletteItem]{}
)

// List returns the article summaries for the archive pages, ordered by date
// descending, refreshed from the database at most once per cacheTTL.
func List(ctx context.Context) []ArticleSummary {
	return summaries.get(ctx, func(ctx context.Context) ([]ArticleSummary, error) {
		return querySummaries(ctx)
	})
}

// Palette returns the minimal article data for the command palette, cached
// server-side like List.
func Palette(ctx context.Context) []PaletteItem {
	return palette.get(ctx, func(ctx context.Context) ([]PaletteItem, error) {
		return queryPalette(ctx)
	})
}

// GetArticle returns the full article for the given slug, or nil when it does
// not exist. Each slug is cached separately; only the requested row is read.
func GetArticle(ctx context.Context, slug string) *Article {
	a, ok := articleStore.get(ctx, slug)
	if !ok {
		return nil
	}
	return &a
}

// articleCache is a per-slug TTL cache for full articles, bounded in memory
// by maxArticleCacheBytes with least-recently-used eviction.
type articleCache struct {
	mu    sync.Mutex
	at    map[string]time.Time // load time: drives the TTL freshness check
	used  map[string]time.Time // last access: drives LRU eviction
	sz    map[string]int       // approximate bytes per entry
	total int                  // sum of sz
	by    map[string]Article
}

var articleStore = &articleCache{}

func (c *articleCache) get(ctx context.Context, slug string) (Article, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if at, ok := c.at[slug]; ok && time.Since(at) <= cacheTTL {
		c.used[slug] = time.Now()
		return c.by[slug], true
	}
	if store.db == nil {
		a, ok := c.by[slug]
		return a, ok
	}
	a, err := queryArticle(ctx, slug)
	if err != nil {
		a, ok := c.by[slug]
		return a, ok
	}
	if c.at == nil {
		c.at = map[string]time.Time{}
		c.used = map[string]time.Time{}
		c.sz = map[string]int{}
		c.by = map[string]Article{}
	}
	now := time.Now()
	if prev, ok := c.sz[slug]; ok {
		c.total -= prev
	}
	c.at[slug] = now
	c.used[slug] = now
	c.sz[slug] = sizeOf(a)
	c.total += c.sz[slug]
	c.by[slug] = a
	c.trim()
	return a, true
}

// remove drops one slug from the per-article cache, used by Invalidate after
// an admin edit so the next article read hits the database.
func (c *articleCache) remove(slug string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if prev, ok := c.sz[slug]; ok {
		c.total -= prev
	}
	delete(c.at, slug)
	delete(c.used, slug)
	delete(c.sz, slug)
	delete(c.by, slug)
}

// trim evicts the least-recently-used entries until total bytes fit under
// maxArticleCacheBytes. A single entry larger than the cap is kept.
func (c *articleCache) trim() {
	for c.total > maxArticleCacheBytes && len(c.by) > 1 {
		oldest := ""
		var at time.Time
		for slug, t := range c.used {
			if oldest == "" || t.Before(at) {
				oldest = slug
				at = t
			}
		}
		c.total -= c.sz[oldest]
		delete(c.at, oldest)
		delete(c.used, oldest)
		delete(c.sz, oldest)
		delete(c.by, oldest)
	}
}

// sizeOf returns an approximate memory footprint of a cached article. Bodies
// dominate; per-string headers and slice overhead are rough constants.
func sizeOf(a Article) int {
	n := len(a.Slug) + len(a.Title) + len(a.Subtitle) + len(a.Date) +
		len(a.Excerpt) + len(a.Image) + len(a.ImageCaption) + len(a.Body)
	for _, t := range a.Tags {
		n += len(t)
	}
	return n + 16*(8+len(a.Tags))
}

// querySummaries loads the archive columns only, never the heavy fields.
func querySummaries(ctx context.Context) ([]ArticleSummary, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT slug, title, date, tags,
		star, featured, image, image_caption
		FROM articles ORDER BY date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ArticleSummary
	for rows.Next() {
		var (
			s    ArticleSummary
			tags string
			star int
			feat int
		)
		if err := rows.Scan(&s.Slug, &s.Title, &s.Date, &tags,
			&star, &feat, &s.Image, &s.ImageCaption); err != nil {
			return nil, err
		}
		s.Tags = unmarshalTags(tags)
		s.Star = star != 0
		s.Featured = feat != 0
		list = append(list, s)
	}
	return list, rows.Err()
}

// queryPalette loads only the fields the command palette reads.
func queryPalette(ctx context.Context) ([]PaletteItem, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT slug, title, tags FROM articles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PaletteItem
	for rows.Next() {
		var (
			it   PaletteItem
			tags string
		)
		if err := rows.Scan(&it.Slug, &it.Title, &tags); err != nil {
			return nil, err
		}
		it.Tags = unmarshalTags(tags)
		items = append(items, it)
	}
	return items, rows.Err()
}

// queryArticle loads one full row by slug (primary key lookup).
func queryArticle(ctx context.Context, slug string) (Article, error) {
	var (
		a        Article
		tags     string
		star     int
		featured int
	)
	err := store.db.QueryRowContext(ctx, `SELECT slug, title, subtitle, date, tags,
		excerpt, image, image_caption, initial_love, star, featured, body
		FROM articles WHERE slug = ?`, slug).Scan(
		&a.Slug, &a.Title, &a.Subtitle, &a.Date, &tags,
		&a.Excerpt, &a.Image, &a.ImageCaption, &a.InitialLove, &star, &featured, &a.Body)
	if err != nil {
		return Article{}, err
	}
	a.Tags = unmarshalTags(tags)
	a.Star = star != 0
	a.Featured = featured != 0
	return a, nil
}

// unmarshalTags parses the JSON array stored in the tags column.
func unmarshalTags(s string) []string {
	var tags []string
	if err := json.Unmarshal([]byte(s), &tags); err != nil {
		return nil
	}
	return tags
}

// Principles returns every imported principle.
func Principles() []Principle {
	return principles
}

// Save inserts the article or, when the slug already exists, updates every
// column. The write goes straight to the database; caches are invalidated so
// the next read sees the new data immediately.
func Save(ctx context.Context, a Article) error {
	tags, err := json.Marshal(a.Tags)
	if err != nil {
		return err
	}
	_, err = store.db.ExecContext(ctx, `INSERT INTO articles
		(slug, title, subtitle, date, tags, excerpt, image, image_caption,
		 initial_love, star, featured, body)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET
			title = excluded.title,
			subtitle = excluded.subtitle,
			date = excluded.date,
			tags = excluded.tags,
			excerpt = excluded.excerpt,
			image = excluded.image,
			image_caption = excluded.image_caption,
			initial_love = excluded.initial_love,
			star = excluded.star,
			featured = excluded.featured,
			body = excluded.body`,
		a.Slug, a.Title, a.Subtitle, a.Date, string(tags), a.Excerpt, a.Image,
		a.ImageCaption, a.InitialLove, boolInt(a.Star), boolInt(a.Featured), a.Body)
	if err != nil {
		return err
	}
	Invalidate(a.Slug)
	return nil
}

// Invalidate drops the cached copies of the given article so the next read
// reloads from the database. Every admin write calls it; public reads never do.
func Invalidate(slug string) {
	summaries.reset()
	palette.reset()
	articleStore.remove(slug)
}

// boolInt converts a bool to SQLite's integer representation.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
