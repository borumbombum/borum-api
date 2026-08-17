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
	Slug           string   `json:"slug"`
	Title          string   `json:"title"`
	Subtitle       string   `json:"subtitle"`
	Date           string   `json:"date"`
	Tags           []string `json:"tags"`
	Excerpt        string   `json:"excerpt"`
	Image          string   `json:"image,omitempty"`
	ImageCaption   string   `json:"imageCaption,omitempty"`
	InitialLove    int      `json:"initialLove"`
	Star           bool     `json:"star"`
	Featured       bool     `json:"featured"`
	Body           string   `json:"body"`
	Status         string   `json:"status"`
	Lang           string   `json:"lang"`
	TranslationOf  string   `json:"translationOf,omitempty"`
	TitlePT        string   `json:"titlePt,omitempty"`
	SubtitlePT     string   `json:"subtitlePt,omitempty"`
	ExcerptPT      string   `json:"excerptPt,omitempty"`
	ImageCaptionPT string   `json:"imageCaptionPt,omitempty"`
	BodyPT         string   `json:"bodyPt,omitempty"`
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
	Status       string
	Lang         string
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
		return querySummaries(ctx, "en")
	})
}

// ListByLang returns the article summaries for a specific language.
func ListByLang(ctx context.Context, lang string) []ArticleSummary {
	if lang == "en" {
		return List(ctx)
	}
	rows, err := store.db.QueryContext(ctx, `SELECT slug, title, date, tags,
		star, featured, image, image_caption, status, lang
		FROM articles WHERE status = 'published' AND lang = ? ORDER BY date DESC`, lang)
	if err != nil {
		return nil
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
			&star, &feat, &s.Image, &s.ImageCaption, &s.Status, &s.Lang); err != nil {
			return nil
		}
		s.Tags = unmarshalTags(tags)
		s.Star = star != 0
		s.Featured = feat != 0
		list = append(list, s)
	}
	return list
}

// Palette returns the minimal article data for the command palette, cached
// server-side like List.
func Palette(ctx context.Context) []PaletteItem {
	return palette.get(ctx, func(ctx context.Context) ([]PaletteItem, error) {
		return queryPalette(ctx)
	})
}

// GetArticle returns the full article for the given slug and language, or nil when it does
// not exist. Each slug+lang is cached separately; only the requested row is read.
func GetArticle(ctx context.Context, slug, lang string) *Article {
	key := slug + ":" + lang
	a, ok := articleStore.get(ctx, key)
	if !ok {
		return nil
	}
	return &a
}

// ListAll returns every article summary (draft and published) for the admin
// list. It does not cache because it is only called from /god pages.
func ListAll(ctx context.Context) []ArticleSummary {
	rows, err := store.db.QueryContext(ctx, `SELECT slug, title, date, tags,
		star, featured, image, image_caption, status, lang
		FROM articles ORDER BY date DESC`)
	if err != nil {
		return nil
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
			&star, &feat, &s.Image, &s.ImageCaption, &s.Status, &s.Lang); err != nil {
			return nil
		}
		s.Tags = unmarshalTags(tags)
		s.Star = star != 0
		s.Featured = feat != 0
		list = append(list, s)
	}
	return list
}

// GetArticleAny returns the full article for the given slug regardless of
// status, for admin edit pages. Uncached because it is only called from /god.
func GetArticleAny(ctx context.Context, slug, lang string) *Article {
	var (
		a        Article
		tags     string
		star     int
		featured int
	)
	err := store.db.QueryRowContext(ctx, `SELECT slug, title, subtitle, date, tags,
		excerpt, image, image_caption, initial_love, star, featured, body, status, lang, translation_of
		FROM articles WHERE slug = ? AND lang = ?`, slug, lang).Scan(
		&a.Slug, &a.Title, &a.Subtitle, &a.Date, &tags,
		&a.Excerpt, &a.Image, &a.ImageCaption, &a.InitialLove, &star, &featured, &a.Body, &a.Status, &a.Lang, &a.TranslationOf)
	if err != nil {
		return nil
	}
	a.Tags = unmarshalTags(tags)
	a.Star = star != 0
	a.Featured = featured != 0
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

func (c *articleCache) get(ctx context.Context, key string) (Article, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if at, ok := c.at[key]; ok && time.Since(at) <= cacheTTL {
		c.used[key] = time.Now()
		return c.by[key], true
	}
	if store.db == nil {
		a, ok := c.by[key]
		return a, ok
	}
	// Parse key as "slug:lang"
	slug := key
	lang := "en"
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			slug = key[:i]
			lang = key[i+1:]
			break
		}
	}
	a, err := queryArticle(ctx, slug, lang)
	if err != nil {
		a, ok := c.by[key]
		return a, ok
	}
	if c.at == nil {
		c.at = map[string]time.Time{}
		c.used = map[string]time.Time{}
		c.sz = map[string]int{}
		c.by = map[string]Article{}
	}
	now := time.Now()
	if prev, ok := c.sz[key]; ok {
		c.total -= prev
	}
	c.at[key] = now
	c.used[key] = now
	c.sz[key] = sizeOf(a)
	c.total += c.sz[key]
	c.by[key] = a
	c.trim()
	return a, true
}

// remove drops one key (slug:lang) from the per-article cache, used by Invalidate after
// an admin edit so the next article read hits the database.
func (c *articleCache) remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if prev, ok := c.sz[key]; ok {
		c.total -= prev
	}
	delete(c.at, key)
	delete(c.used, key)
	delete(c.sz, key)
	delete(c.by, key)
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
// Only published articles are included.
func querySummaries(ctx context.Context, lang string) ([]ArticleSummary, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT slug, title, date, tags,
		star, featured, image, image_caption, status, lang
		FROM articles WHERE status = 'published' AND lang = ? ORDER BY date DESC`, lang)
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
			&star, &feat, &s.Image, &s.ImageCaption, &s.Status, &s.Lang); err != nil {
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
// Only published articles are included.
func queryPalette(ctx context.Context) ([]PaletteItem, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT slug, title, tags, lang FROM articles WHERE status = 'published'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PaletteItem
	for rows.Next() {
		var (
			it   PaletteItem
			tags string
			lang string
		)
		if err := rows.Scan(&it.Slug, &it.Title, &tags, &lang); err != nil {
			return nil, err
		}
		it.Tags = unmarshalTags(tags)
		items = append(items, it)
	}
	return items, rows.Err()
}

// queryArticle loads one full row by slug and language (primary key lookup).
// Only published articles are returned.
func queryArticle(ctx context.Context, slug, lang string) (Article, error) {
	var (
		a        Article
		tags     string
		star     int
		featured int
	)
	err := store.db.QueryRowContext(ctx, `SELECT slug, title, subtitle, date, tags,
		excerpt, image, image_caption, initial_love, star, featured, body, status, lang, translation_of
		FROM articles WHERE slug = ? AND lang = ? AND status = 'published'`, slug, lang).Scan(
		&a.Slug, &a.Title, &a.Subtitle, &a.Date, &tags,
		&a.Excerpt, &a.Image, &a.ImageCaption, &a.InitialLove, &star, &featured, &a.Body, &a.Status, &a.Lang, &a.TranslationOf)
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

// Save inserts the article or, when the slug+lang already exists, updates every
// column. The write goes straight to the database; caches are invalidated so
// the next read sees the new data immediately.
func Save(ctx context.Context, a Article) error {
	tags, err := json.Marshal(a.Tags)
	if err != nil {
		return err
	}
	status := a.Status
	if status == "" {
		status = "published"
	}
	lang := a.Lang
	if lang == "" {
		lang = "en"
	}
	
	// For Portuguese, save translation fields
	if lang == "pt" {
		_, err = store.db.ExecContext(ctx, `INSERT INTO articles
			(slug, title, subtitle, date, tags, excerpt, image, image_caption,
			 initial_love, star, featured, body, status, lang, translation_of)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(slug, lang) DO UPDATE SET
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
				body = excluded.body,
				status = excluded.status,
				translation_of = excluded.translation_of`,
			a.Slug, a.Title, a.Subtitle, a.Date, string(tags), a.Excerpt, a.Image,
			a.ImageCaption, a.InitialLove, boolInt(a.Star), boolInt(a.Featured), a.Body, status, lang, a.TranslationOf)
	} else {
		_, err = store.db.ExecContext(ctx, `INSERT INTO articles
			(slug, title, subtitle, date, tags, excerpt, image, image_caption,
			 initial_love, star, featured, body, status, lang, translation_of)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(slug, lang) DO UPDATE SET
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
				body = excluded.body,
				status = excluded.status`,
			a.Slug, a.Title, a.Subtitle, a.Date, string(tags), a.Excerpt, a.Image,
			a.ImageCaption, a.InitialLove, boolInt(a.Star), boolInt(a.Featured), a.Body, status, lang, a.TranslationOf)
	}
	if err != nil {
		return err
	}
	Invalidate(a.Slug)
	return nil
}

// Delete removes the article with the given slug and language and drops it from every
// cache so public pages stop showing it immediately.
func Delete(ctx context.Context, slug, lang string) error {
	if _, err := store.db.ExecContext(ctx, `DELETE FROM articles WHERE slug = ? AND lang = ?`, slug, lang); err != nil {
		return err
	}
	Invalidate(slug)
	return nil
}

// ChangeSlug renames an article's primary key. Only call this for drafts whose
// slug can still change. SQLite allows updating the PRIMARY KEY directly.
func ChangeSlug(ctx context.Context, oldSlug, newSlug, lang string) error {
	if _, err := store.db.ExecContext(ctx,
		`UPDATE articles SET slug = ? WHERE slug = ? AND lang = ?`, newSlug, oldSlug, lang); err != nil {
		return err
	}
	Invalidate(oldSlug)
	return nil
}

// Invalidate drops the cached copies of the given article so the next read
// reloads from the database. Every admin write calls it; public reads never do.
func Invalidate(slug string) {
	summaries.reset()
	palette.reset()
	// Remove all language variants from cache
	for _, lang := range []string{"en", "pt"} {
		articleStore.remove(slug + ":" + lang)
	}
}

// GetTranslationSlug returns the slug of the alternate language version of an article,
// or empty string if no translation exists.
func GetTranslationSlug(ctx context.Context, slug, lang string) string {
	var otherLang string
	if lang == "en" {
		otherLang = "pt"
	} else {
		otherLang = "en"
	}
	
	// First try: look for an article with translation_of pointing to this slug
	var transSlug string
	err := store.db.QueryRowContext(ctx,
		`SELECT slug FROM articles WHERE translation_of = ? AND lang = ? AND status = 'published' LIMIT 1`,
		slug, otherLang).Scan(&transSlug)
	if err == nil {
		return transSlug
	}
	
	// Second try: look for an article where this slug is the translation_of
	err = store.db.QueryRowContext(ctx,
		`SELECT slug FROM articles WHERE slug = ? AND lang = ? AND status = 'published'`, 
		slug, otherLang).Scan(&transSlug)
	if err == nil {
		return transSlug
	}
	
	// Third try: look for article with same slug in other language
	err = store.db.QueryRowContext(ctx,
		`SELECT slug FROM articles WHERE slug = ? AND lang = ? AND status = 'published'`,
		slug, otherLang).Scan(&transSlug)
	if err == nil {
		return transSlug
	}
	
	return ""
}

// boolInt converts a bool to SQLite's integer representation.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
