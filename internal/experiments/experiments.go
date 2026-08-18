// Package experiments owns the site's experiments: small self-contained
// tools (each with its own template folder and static asset folder) linked
// from the home page above the articles.
//
// Experiment definitions are hardcoded in this file (the registry). The
// database only stores admin-tunable state per experiment: whether it is
// visible on the home page and its display order. Reads are cached for
// cacheTTL exactly like the article store, so the home page never pays a
// per-visitor database cost for the experiments section.
package experiments

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/borumbombum/borum-api/internal/i18n"
)

// Experiment is one hardcoded experiment. Dir is the experiment's own folder
// name under cmd/web/templates/experiments/ (and static/assets/experiments/),
// carrying a numeric prefix that sets its default display order.
type Experiment struct {
	Slug        string // URL path segment, e.g. /experiments/img2webp
	Title       string
	Description string
	Dir         string // e.g. "01-img2webp"
}

// Item is an experiment plus its admin-tunable state and its 1-based display
// number (used for the [#N] home page labels). Intro is the admin-written
// HTML shown above the experiment's form; empty means none.
type Item struct {
	Experiment
	Enabled bool
	Sort    int
	Number  int
	Intro   string
	IntroPT string
}

// ExperimentTranslation holds translated content for an experiment.
type ExperimentTranslation struct {
	Slug        string
	Lang        string
	Title       string
	Description string
	Intro       string
}

// all is the hardcoded registry: the source of truth for which experiments
// exist. Add a new experiment here, give it its own folder under
// cmd/web/templates/experiments/ and static/assets/experiments/, and register
// its routes in cmd/web/routes.go.
var all = []Experiment{
	{
		Slug:        "img2webp",
		Title:       "Image → WebP converter",
		Description: "Upload a PNG or JPEG and download it as WebP.",
		Dir:         "01-img2webp",
	},
	{
		Slug:        "qr-smuggler-kit",
		Title:       "QR Smuggler Kit",
		Description: "Decode and encode QR codes in your browser.",
		Dir:         "02-qr-smuggler-kit",
	},
}

// All returns the registry in default (folder-prefix) order.
func All() []Experiment {
	return append([]Experiment(nil), all...)
}

// BySlug returns the registry entry with the given slug, or nil.
func BySlug(slug string) *Experiment {
	for i := range all {
		if all[i].Slug == slug {
			return &all[i]
		}
	}
	return nil
}

// defaultOrder reads the numeric prefix of the folder name. Prefixless
// folders sort after every prefixed one.
func defaultOrder(dir string) int {
	n := 0
	seen := false
	for _, r := range dir {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
			seen = true
			continue
		}
		break
	}
	if !seen {
		return 1 << 30
	}
	return n
}

// cacheTTL mirrors the article cache: short enough that manual database edits
// show up, long enough to keep repeat home page loads off the remote DB.
const cacheTTL = 61 * time.Minute

// store holds the database handle, wired by Init at startup.
var store struct {
	mu sync.Mutex
	db *sql.DB
}

// Init wires the experiment store to the database.
func Init(db *sql.DB) error {
	store.mu.Lock()
	store.db = db
	store.mu.Unlock()
	return nil
}

// ttlCache caches List for cacheTTL. A failed load returns the last good
// value and the next call retries.
type ttlCache struct {
	mu     sync.Mutex
	value  []Item
	at     time.Time
	loaded bool
}

var cache = &ttlCache{}

func (c *ttlCache) get(ctx context.Context, load func(context.Context) ([]Item, error)) []Item {
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

func (c *ttlCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loaded = false
}

// List returns every registry experiment with its stored enabled/sort state,
// ordered by (sort, folder-prefix) with Number as the 1-based position.
// Cached for cacheTTL; admin writes reset the cache.
func List(ctx context.Context) []Item {
	return cache.get(ctx, queryAll)
}

// Enabled reports whether the experiment is visible on the home page.
func Enabled(ctx context.Context, slug string) bool {
	for _, it := range List(ctx) {
		if it.Slug == slug {
			return it.Enabled
		}
	}
	return false
}

// SetEnabled shows or hides the experiment. Admin write; resets the cache.
func SetEnabled(ctx context.Context, slug string, enabled bool) error {
	res, err := store.db.ExecContext(ctx,
		`UPDATE experiments SET enabled = ? WHERE slug = ?`, boolInt(enabled), slug)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		e := BySlug(slug)
		if e == nil {
			return fmt.Errorf("unknown experiment %s", slug)
		}
		_, err = store.db.ExecContext(ctx,
			`INSERT INTO experiments (slug, enabled, sort) VALUES (?, ?, ?)`,
			slug, boolInt(enabled), defaultOrder(e.Dir))
		if err != nil {
			return err
		}
	}
	cache.reset()
	return nil
}

// UpdateIntro replaces the admin-written intro text shown above the
// experiment's form. Admin write; resets the cache.
func UpdateIntro(ctx context.Context, slug, lang, intro string) error {
	if lang == i18n.DefaultLang {
		res, err := store.db.ExecContext(ctx,
			`UPDATE experiments SET intro = ? WHERE slug = ?`, intro, slug)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err == nil && n == 0 {
			e := BySlug(slug)
			if e == nil {
				return fmt.Errorf("unknown experiment %s", slug)
			}
			_, err = store.db.ExecContext(ctx,
				`INSERT INTO experiments (slug, enabled, sort, intro) VALUES (?, 1, ?, ?)`,
				slug, defaultOrder(e.Dir), intro)
			if err != nil {
				return err
			}
		}
	} else {
		// For other languages, save to experiment_translations
		_, err := store.db.ExecContext(ctx,
			`INSERT INTO experiment_translations (slug, lang, title, description, intro) VALUES (?, ?, '', '', ?)
			 ON CONFLICT(slug, lang) DO UPDATE SET intro = excluded.intro`,
			slug, lang, intro)
		if err != nil {
			return err
		}
	}
	cache.reset()
	return nil
}

// Move shifts the experiment one position up (dir -1) or down (dir +1) in the
// display order by swapping its sort value with the neighbour. Admin write;
// resets the cache.
func Move(ctx context.Context, slug string, dir int) error {
	rows, err := store.db.QueryContext(ctx, `SELECT slug, sort FROM experiments`)
	if err != nil {
		return err
	}
	type row struct {
		slug string
		sort int
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.slug, &r.sort); err != nil {
			rows.Close()
			return err
		}
		list = append(list, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	// Order with the registry prefix as tie-break so swaps match List.
	index := map[string]int{}
	for i := range all {
		index[all[i].Slug] = i
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].sort != list[j].sort {
			return list[i].sort < list[j].sort
		}
		return index[list[i].slug] < index[list[j].slug]
	})

	pos := -1
	for i, r := range list {
		if r.slug == slug {
			pos = i
			break
		}
	}
	if pos == -1 {
		return fmt.Errorf("unknown experiment %s", slug)
	}
	other := pos + dir
	if other < 0 || other >= len(list) {
		return nil // already at the edge
	}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`UPDATE experiments SET sort = ? WHERE slug = ?`, list[other].sort, slug); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE experiments SET sort = ? WHERE slug = ?`, list[pos].sort, list[other].slug); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	cache.reset()
	return nil
}

// GetTranslation returns the translated content for an experiment, or nil if not found.
func GetTranslation(ctx context.Context, slug, lang string) (*ExperimentTranslation, error) {
	if lang == i18n.DefaultLang {
		return nil, nil // English is the base row, not a translation
	}
	var t ExperimentTranslation
	row := store.db.QueryRowContext(ctx,
		`SELECT slug, lang, title, description, intro FROM experiment_translations WHERE slug = ? AND lang = ?`,
		slug, lang)
	err := row.Scan(&t.Slug, &t.Lang, &t.Title, &t.Description, &t.Intro)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// SaveTranslation inserts or updates a translation for an experiment.
func SaveTranslation(ctx context.Context, slug, lang, title, description, intro string) error {
	_, err := store.db.ExecContext(ctx,
		`INSERT INTO experiment_translations (slug, lang, title, description, intro) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(slug, lang) DO UPDATE SET title = excluded.title, description = excluded.description, intro = excluded.intro`,
		slug, lang, title, description, intro)
	return err
}

// DeleteTranslation removes a translation for an experiment.
func DeleteTranslation(ctx context.Context, slug, lang string) error {
	_, err := store.db.ExecContext(ctx,
		`DELETE FROM experiment_translations WHERE slug = ? AND lang = ?`, slug, lang)
	return err
}

// GetTranslationSlug returns the slug of the alternate language version of an experiment,
// or empty string if no translation exists.
func GetTranslationSlug(ctx context.Context, slug, lang string) string {
	var otherLang string
	if lang == i18n.DefaultLang {
		otherLang = "pt"
	} else {
		otherLang = i18n.DefaultLang
	}

	var transSlug string
	err := store.db.QueryRowContext(ctx,
		`SELECT slug FROM experiment_translations WHERE slug = ? AND lang = ?`,
		slug, otherLang).Scan(&transSlug)
	if err == nil {
		return transSlug
	}

	return ""
}

// queryAll loads the stored state for every registry experiment. Slugs with
// no row (should not happen after Init) fall back to enabled + prefix order.
func queryAll(ctx context.Context) ([]Item, error) {
	type state struct {
		enabled int
		sort    int
		intro   string
	}
	bySlug := map[string]state{}
	rows, err := store.db.QueryContext(ctx, `SELECT slug, enabled, sort, intro FROM experiments`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s state
		var slug string
		if err := rows.Scan(&slug, &s.enabled, &s.sort, &s.intro); err != nil {
			return nil, err
		}
		bySlug[slug] = s
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	introPT := map[string]string{}
	trows, err := store.db.QueryContext(ctx,
		`SELECT slug, intro FROM experiment_translations WHERE lang = 'pt'`)
	if err != nil {
		return nil, err
	}
	defer trows.Close()
	for trows.Next() {
		var slug, intro string
		if err := trows.Scan(&slug, &intro); err != nil {
			return nil, err
		}
		introPT[slug] = intro
	}
	if err := trows.Err(); err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(all))
	for _, e := range all {
		s, ok := bySlug[e.Slug]
		if !ok {
			s = state{enabled: 1, sort: defaultOrder(e.Dir)}
		}
		items = append(items, Item{
			Experiment: e,
			Enabled:    s.enabled != 0,
			Sort:       s.sort,
			Intro:      s.intro,
			IntroPT:    introPT[e.Slug],
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Sort != items[j].Sort {
			return items[i].Sort > items[j].Sort
		}
		return defaultOrder(items[i].Dir) > defaultOrder(items[j].Dir)
	})
	for i := range items {
		items[i].Number = i + 1
	}
	return items, nil
}

// boolInt converts a bool to SQLite's integer representation.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
