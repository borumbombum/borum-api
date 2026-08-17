# Translations (i18n)

This document explains how the translation system works for articles and experiments.

## Supported Languages

- `en` (English) — default, no URL prefix
- `pt` (Portuguese) — subpath prefix: `/pt/...`

To add a new language: add to `Supported` slice in `internal/i18n/i18n.go`, add route prefixes in `cmd/web/routes.go`, add admin fields in templates.

## Database Schema

### Articles

Articles store translations as **separate rows** with the same slug but different `lang`:

```sql
ALTER TABLE articles ADD COLUMN lang TEXT NOT NULL DEFAULT 'en';
ALTER TABLE articles ADD COLUMN translation_of TEXT NOT NULL DEFAULT '';
```

- `lang`: `'en'` or `'pt'`
- `translation_of`: slug of the original article (empty for originals)

Example: An English article with slug `my-post` and its Portuguese translation both have `slug = 'my-post'`, but different `lang` values.

### Experiments

Experiments use a **separate translations table**:

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

- English content lives in the `experiments` table (hardcoded + DB state)
- Translations live in `experiment_translations`

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
func Prefix(lang string) string  // "" for en, "/pt" for pt
func OtherLang(lang string) string
```

### Article Model (`internal/content/content.go`)

```go
type Article struct {
    // ... existing fields ...
    Lang           string `json:"lang"`
    TranslationOf  string `json:"translationOf,omitempty"`
}
```

Key functions:
- `GetArticle(ctx, slug, lang)` — returns article by slug + lang
- `GetTranslationSlug(ctx, slug, lang)` — finds alternate language version
- `ListByLang(ctx, lang)` — filters articles by language
- `Save(ctx, article)` — upserts by (slug, lang)

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
- `UpdateIntro(ctx, slug, lang, intro)` — saves intro for any language

## Middleware

Language detection middleware in `cmd/web/routes.go`:

```go
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        lang := "en"
        if strings.HasPrefix(r.URL.Path, "/pt/") || r.URL.Path == "/pt" {
            lang = "pt"
        }
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

Every page includes hreflang links in the `<head>`:

```html
<link rel="alternate" hreflang="en" href="https://borum.dev/blog/{slug}" />
<link rel="alternate" hreflang="pt" href="https://borum.dev/pt/blog/{slug}" />
<link rel="alternate" hreflang="x-default" href="https://borum.dev/blog/{slug}" />
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
2. Add route prefixes in `cmd/web/routes.go`:
   - `/{lang}`, `/{lang}/blog/{slug}`, `/{lang}/tags/{tag}`, `/{lang}/experiments/{slug}`
3. Update language detection middleware in `cmd/web/routes.go`
4. Add admin fields in templates:
   - `god_article_form.html` — add new language section
   - `god_experiments.html` — add new language intro editor
5. Update handlers to load/save new language
6. Update hreflang tags in templates
