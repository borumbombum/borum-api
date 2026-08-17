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

	"github.com/borumbombum/borum-api/internal/i18n"
)

// Article is a single blog post (base row, used by the article page).
// Translations are stored in article_translations table.
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
	Status       string   `json:"status"`
}

// ArticleTranslation holds translated content for a specific language.
type ArticleTranslation struct {
	Slug         string `json:"slug"`
	Lang         string `json:"lang"`
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle"`
	Excerpt      string `json:"excerpt"`
	ImageCaption string `json:"imageCaption"`
	Body         string `json:"body"`
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

// ListByLang returns the article summaries for a specific language.
// For English, returns base rows. For other languages, filters by translation existence.
func ListByLang(ctx context.Context, lang string) []ArticleSummary {
	if lang == i18n.DefaultLang {
		return List(ctx)
	}
	// For non-English, join with translations to only show translated articles
	rows, err := store.db.QueryContext(ctx, `SELECT a.slug, a.title, a.date, a.tags,
		a.star, a.featured, a.image, a.image_caption, a.status
		FROM articles a
		INNER JOIN article_translations t ON a.slug = t.slug
		WHERE a.status = 'published' AND t.lang = ?
		ORDER BY a.date DESC`, lang)
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
			&star, &feat, &s.Image, &s.ImageCaption, &s.Status); err != nil {
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

// GetArticle returns the full article for the given slug, or nil when it does
// not exist. Only published articles are returned.
func GetArticle(ctx context.Context, slug string) *Article {
	key := slug
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
		star, featured, image, image_caption, status
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
			&star, &feat, &s.Image, &s.ImageCaption, &s.Status); err != nil {
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
func GetArticleAny(ctx context.Context, slug string) *Article {
	var (
		a        Article
		tags     string
		star     int
		featured int
	)
	err := store.db.QueryRowContext(ctx, `SELECT slug, title, subtitle, date, tags,
		excerpt, image, image_caption, initial_love, star, featured, body, status
		FROM articles WHERE slug = ?`, slug).Scan(
		&a.Slug, &a.Title, &a.Subtitle, &a.Date, &tags,
		&a.Excerpt, &a.Image, &a.ImageCaption, &a.InitialLove, &star, &featured, &a.Body, &a.Status)
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
	// Parse key as "slug" (no more :lang suffix)
	a, err := queryArticle(ctx, key)
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

// remove drops one key (slug) from the per-article cache, used by Invalidate after
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
func querySummaries(ctx context.Context) ([]ArticleSummary, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT slug, title, date, tags,
		star, featured, image, image_caption, status
		FROM articles WHERE status = 'published' ORDER BY date DESC`)
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
			&star, &feat, &s.Image, &s.ImageCaption, &s.Status); err != nil {
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
	rows, err := store.db.QueryContext(ctx, `SELECT slug, title, tags FROM articles WHERE status = 'published'`)
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
// Only published articles are returned.
func queryArticle(ctx context.Context, slug string) (Article, error) {
	var (
		a        Article
		tags     string
		star     int
		featured int
	)
	err := store.db.QueryRowContext(ctx, `SELECT slug, title, subtitle, date, tags,
		excerpt, image, image_caption, initial_love, star, featured, body, status
		FROM articles WHERE slug = ? AND status = 'published'`, slug).Scan(
		&a.Slug, &a.Title, &a.Subtitle, &a.Date, &tags,
		&a.Excerpt, &a.Image, &a.ImageCaption, &a.InitialLove, &star, &featured, &a.Body, &a.Status)
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
	status := a.Status
	if status == "" {
		status = "published"
	}

	// Single SQL statement for all languages
	_, err = store.db.ExecContext(ctx, `INSERT INTO articles
		(slug, title, subtitle, date, tags, excerpt, image, image_caption,
		 initial_love, star, featured, body, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			body = excluded.body,
			status = excluded.status`,
		a.Slug, a.Title, a.Subtitle, a.Date, string(tags), a.Excerpt, a.Image,
		a.ImageCaption, a.InitialLove, boolInt(a.Star), boolInt(a.Featured), a.Body, status)
	if err != nil {
		return err
	}
	Invalidate(a.Slug)
	return nil
}

// Delete removes the article with the given slug and all its translations,
// then drops it from every cache so public pages stop showing it immediately.
func Delete(ctx context.Context, slug string) error {
	// Delete translations first (foreign key constraint)
	if _, err := store.db.ExecContext(ctx, `DELETE FROM article_translations WHERE slug = ?`, slug); err != nil {
		return err
	}
	// Delete the base article
	if _, err := store.db.ExecContext(ctx, `DELETE FROM articles WHERE slug = ?`, slug); err != nil {
		return err
	}
	Invalidate(slug)
	return nil
}

// ChangeSlug renames an article's primary key. Only call this for drafts whose
// slug can still change. Also updates translation keys.
func ChangeSlug(ctx context.Context, oldSlug, newSlug string) error {
	// Update base article
	if _, err := store.db.ExecContext(ctx,
		`UPDATE articles SET slug = ? WHERE slug = ?`, newSlug, oldSlug); err != nil {
		return err
	}
	// Update translations
	if _, err := store.db.ExecContext(ctx,
		`UPDATE article_translations SET slug = ? WHERE slug = ?`, newSlug, oldSlug); err != nil {
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
	articleStore.remove(slug)
}

// GetTranslation returns the full translation for the given slug and language,
// or nil when no translation exists.
func GetTranslation(ctx context.Context, slug, lang string) *ArticleTranslation {
	if lang == i18n.DefaultLang {
		return nil // English is the base row, not a translation
	}
	var t ArticleTranslation
	err := store.db.QueryRowContext(ctx,
		`SELECT slug, lang, title, subtitle, excerpt, image_caption, body
		FROM article_translations WHERE slug = ? AND lang = ?`,
		slug, lang).Scan(&t.Slug, &t.Lang, &t.Title, &t.Subtitle, &t.Excerpt, &t.ImageCaption, &t.Body)
	if err != nil {
		return nil
	}
	return &t
}

// SaveTranslation inserts or updates a translation for the given slug and language.
func SaveTranslation(ctx context.Context, t ArticleTranslation) error {
	_, err := store.db.ExecContext(ctx, `INSERT INTO article_translations
		(slug, lang, title, subtitle, excerpt, image_caption, body)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(slug, lang) DO UPDATE SET
			title = excluded.title,
			subtitle = excluded.subtitle,
			excerpt = excluded.excerpt,
			image_caption = excluded.image_caption,
			body = excluded.body`,
		t.Slug, t.Lang, t.Title, t.Subtitle, t.Excerpt, t.ImageCaption, t.Body)
	if err != nil {
		return err
	}
	Invalidate(t.Slug)
	return nil
}

// DeleteTranslation removes a single translation for the given slug and language.
func DeleteTranslation(ctx context.Context, slug, lang string) error {
	if _, err := store.db.ExecContext(ctx,
		`DELETE FROM article_translations WHERE slug = ? AND lang = ?`, slug, lang); err != nil {
		return err
	}
	Invalidate(slug)
	return nil
}

// GetTranslationSlug returns the slug of the alternate language version of an article,
// or empty string if no translation exists.
func GetTranslationSlug(ctx context.Context, slug, lang string) string {
	var otherLang string
	if lang == i18n.DefaultLang {
		otherLang = "pt"
	} else {
		otherLang = i18n.DefaultLang
	}

	// For non-English, check if translation exists
	if otherLang != i18n.DefaultLang {
		var transSlug string
		err := store.db.QueryRowContext(ctx,
			`SELECT slug FROM article_translations WHERE slug = ? AND lang = ?`,
			slug, otherLang).Scan(&transSlug)
		if err == nil {
			return transSlug
		}
	}

	// For English, check if translation exists for the other language
	var transSlug string
	err := store.db.QueryRowContext(ctx,
		`SELECT slug FROM article_translations WHERE slug = ? AND lang = ?`,
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
