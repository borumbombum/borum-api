# Translations (i18n)

This document explains how the translation system works for articles and experiments.

## Supported Languages

- `en` (English) — default, no URL prefix
- `pt` (Portuguese) — subpath prefix: `/pt/...`

To add a new language: add to `Supported` slice in `internal/i18n/i18n.go`. Routes, middleware, and cache invalidation follow automatically.

## Database Schema

### Articles

Articles use a **base row + translations table** model:

**Base table** (one row per article):
```sql
CREATE TABLE articles (
    slug TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    subtitle TEXT NOT NULL DEFAULT '',
    date TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '[]',
    excerpt TEXT NOT NULL DEFAULT '',
    image TEXT NOT NULL DEFAULT '',
    image_caption TEXT NOT NULL DEFAULT '',
    initial_love INTEGER NOT NULL DEFAULT 0,
    star INTEGER NOT NULL DEFAULT 0,
    featured INTEGER NOT NULL DEFAULT 0,
    body TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft'
);
```

**Translations table** (one row per language):
```sql
CREATE TABLE article_translations (
    slug TEXT NOT NULL,
    lang TEXT NOT NULL DEFAULT 'en',
    title TEXT NOT NULL DEFAULT '',
    subtitle TEXT NOT NULL DEFAULT '',
    excerpt TEXT NOT NULL DEFAULT '',
    image_caption TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (slug, lang)
);
```

- English content lives in the `articles` table (base row)
- Translations live in `article_translations`
- Shared metadata (date, tags, image, star, featured, status) lives once on the base row

### Experiments

Experiments use the same pattern:

**Base table** (hardcoded + DB state):
```sql
CREATE TABLE experiments (
    slug TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 1,
    sort INTEGER NOT NULL DEFAULT 0,
    intro TEXT NOT NULL DEFAULT ''
);
```

**Translations table**:
```sql
CREATE TABLE experiment_translations (
    slug TEXT NOT NULL,
    lang TEXT NOT NULL DEFAULT 'en',
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    intro TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (slug, lang)
);
```

## URL Structure

| Language | Article URL | Experiment URL |
|----------|-------------|----------------|
| English | `/blog/{slug}` | `/experiments/{slug}` |
| Portuguese | `/pt/blog/{slug}` | `/pt/experiments/{slug}` |

## Go Code

### Package: `internal/i18n`

Constants and helpers:

```go
const DefaultLang = "en"
var Supported = []string{"en", "pt"}
func IsValid(lang string) bool
func URLFor(lang, path string) string  // "" for en, "/pt" for pt
func LangFromPath(path string) string  // extracts lang from URL path
```

### Article Model (`internal/content/content.go`)

```go
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

type ArticleTranslation struct {
    Slug         string `json:"slug"`
    Lang         string `json:"lang"`
    Title        string `json:"title"`
    Subtitle     string `json:"subtitle"`
    Excerpt      string `json:"excerpt"`
    ImageCaption string `json:"imageCaption"`
    Body         string `json:"body"`
}
```

Key functions:
- `GetArticle(ctx, slug)` — returns base article (English)
- `GetTranslation(ctx, slug, lang)` — returns translation from DB
- `SaveTranslation(ctx, translation)` — upsert translation
- `DeleteTranslation(ctx, slug, lang)` — delete single translation
- `GetTranslationSlug(ctx, slug, lang)` — finds alternate language version
- `ListByLang(ctx, lang)` — filters articles by language (joins with translations)
- `Save(ctx, article)` — upserts base article
- `Delete(ctx, slug)` — deletes base article + all translations

### Experiment Model (`internal/experiments/experiments.go`)

```go
type ExperimentTranslation struct {
    Slug        string
    Lang        string
    Title       string
    Description string
    Intro       string
}
```

Key functions:
- `GetTranslation(ctx, slug, lang)` — returns translation from DB
- `SaveTranslation(ctx, slug, lang, title, desc, intro)` — upsert
- `DeleteTranslation(ctx, slug, lang)` — delete translation
- `GetTranslationSlug(ctx, slug, lang)` — finds alternate language version
- `UpdateIntro(ctx, slug, lang, intro)` — saves intro for any language

## Middleware

Language detection middleware in `cmd/web/routes.go` uses `i18n.LangFromPath()`:

```go
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        lang := i18n.LangFromPath(r.URL.Path)
        ctx := context.WithValue(r.Context(), "lang", lang)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
})
```

## Templates

### Dynamic `html lang`

Both `base.html` and `god_base.html` use:

```html
<html lang="{{.Lang}}">
```

### hreflang Tags

Every page includes hreflang links in the `<head>`, conditional on translation existence:

```html
<link rel="canonical" href="https://borum.dev{{if ne .Lang "en"}}/{{.Lang}}{{end}}/blog/{{.Article.Slug}}" />
{{if ne .Lang "en"}}<link rel="alternate" hreflang="en" href="https://borum.dev/blog/{{.Article.Slug}}" />{{end}}
{{if ne .Lang "pt"}}<link rel="alternate" hreflang="pt" href="https://borum.dev/pt/blog/{{if eq .Lang "pt"}}{{.Article.Slug}}{{else}}{{.TranslationSlug}}{{end}}" />{{end}}
<link rel="alternate" hreflang="x-default" href="https://borum.dev/blog/{{if ne .Lang "en"}}{{.TranslationSlug}}{{else}}{{.Article.Slug}}{{end}}" />
```

### JSON-LD Structured Data

Articles include Article schema:

```json
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "...",
  "inLanguage": "en",
  ...
}
```

Experiments include WebApplication schema.

## Admin Workflow

### Articles

1. Edit article at `/god/articles/{slug}/edit`
2. Scroll to "Portuguese Translation" section
3. Fill in Portuguese fields (title, subtitle, excerpt, image caption, body)
4. Click "save translation"

### Experiments

1. Go to `/god/experiments`
2. Click "intro (pt)" to expand Portuguese intro editor
3. Write Portuguese intro
4. Click "save intro (pt)"

## Adding a New Language

1. Add language code to `Supported` in `internal/i18n/i18n.go`
2. Routes are automatically generated for the new language
3. Language detection middleware uses `i18n.LangFromPath()` — no changes needed
4. Add admin fields in templates:
   - `god_article_form.html` — add new language section
   - `god_experiments.html` — add new language intro editor
5. Update handlers to load/save new language
6. hreflang tags in templates use `i18n.Supported` — no changes needed

## Migration Notes

The unified model uses migration 0008 to:
1. Create `article_translations` table
2. Migrate existing PT data from articles to article_translations
3. Drop dead columns from articles table (`lang`, `translation_of`)
4. Clean up any PT rows that were incorrectly stored in the articles table
