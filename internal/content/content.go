// Package content holds the blog and principles data. Articles live in the
// SQLite (Turso) database and are read through a short-lived in-memory cache
// (see Init); principles live in the generated data_principles.go. This file
// defines the shapes and the accessor functions.
package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"
)

// Article is a single blog post.
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

// Principle is one numbered maxim from the home page.
type Principle struct {
	N     int
	Title string
	Body  string
}

// cacheTTL is how long a loaded article set stays fresh. Short enough that a
// manual edit in the database shows up quickly, long enough to keep page loads
// off the remote database.
const cacheTTL = time.Minute

// store holds the database handle and the last loaded article set. Reads are
// served from cache; a fetch happens at most once per cacheTTL.
var store = struct {
	mu       sync.Mutex
	db       *sql.DB
	articles []Article
	loadedAt time.Time
}{}

// Init wires the article store to the database. It must be called once at
// startup, after Migrate has run, before serving requests.
func Init(db *sql.DB) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.db = db
}

// Articles returns every article, refreshing from the database when the cache
// is stale. On a query error it returns the last good set (possibly empty).
func Articles(ctx context.Context) []Article {
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.db != nil && (len(store.articles) == 0 || time.Since(store.loadedAt) > cacheTTL) {
		if arts, err := queryAll(ctx, store.db); err == nil {
			store.articles = arts
			store.loadedAt = time.Now()
		}
	}
	return store.articles
}

// FindArticle returns the article with the given slug, or nil.
func FindArticle(ctx context.Context, slug string) *Article {
	arts := Articles(ctx)
	for i := range arts {
		if arts[i].Slug == slug {
			return &arts[i]
		}
	}
	return nil
}

// queryAll loads every article row from the database.
func queryAll(ctx context.Context, db *sql.DB) ([]Article, error) {
	rows, err := db.QueryContext(ctx, `SELECT slug, title, subtitle, date, tags,
		excerpt, image, image_caption, initial_love, star, featured, body
		FROM articles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var arts []Article
	for rows.Next() {
		var (
			a           Article
			tags        string
			star        int
			featured    int
		)
		if err := rows.Scan(&a.Slug, &a.Title, &a.Subtitle, &a.Date, &tags,
			&a.Excerpt, &a.Image, &a.ImageCaption, &a.InitialLove, &star, &featured, &a.Body); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tags), &a.Tags); err != nil {
			a.Tags = nil
		}
		a.Star = star != 0
		a.Featured = featured != 0
		arts = append(arts, a)
	}
	return arts, rows.Err()
}

// Principles returns every imported principle.
func Principles() []Principle {
	return principles
}
