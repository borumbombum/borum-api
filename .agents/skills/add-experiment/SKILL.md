# Add Experiment

How to add a new experiment to the site.

## Overview

Experiments are hardcoded in Go code (the registry). The database only stores admin-tunable state: whether each experiment is visible and its display order. No migrations needed for new experiments.

## Files Involved

- `internal/experiments/experiments.go` — registry (add entry to `all` slice)
- `cmd/web/templates/experiments/{NN-slug}/index.html` — experiment template
- `static/assets/experiments/{NN-slug}/` — JS/CSS assets (optional)
- `cmd/web/routes.go` — API routes for form submissions (if needed)
- `cmd/web/experiments_handlers.go` — handler for form processing (if needed)

## Step-by-Step

### 1. Add to Registry

Edit `internal/experiments/experiments.go`, add to the `all` slice:

```go
var all = []Experiment{
    {
        Slug:        "img2webp",
        Title:       "Image → WebP converter",
        Description: "Upload a PNG or JPEG and download it as WebP.",
        Dir:         "01-img2webp",
    },
    {
        Slug:        "my-new-experiment",
        Title:       "My New Experiment",
        Description: "What this experiment does.",
        Dir:         "02-my-new-experiment",
    },
}
```

**Conventions:**
- `Slug` — URL path segment (e.g., `/experiments/my-new-experiment`)
- `Dir` — folder name with numeric prefix (e.g., `02-my-new-experiment`). Prefix sets default display order.
- `Title` — short name shown on home page
- `Description` — one-line description shown on home page

### 2. Create Template

Create `cmd/web/templates/experiments/{NN-slug}/index.html`:

```html
{{define "experiment_body"}}
<div class="wrapper wrapper-wide relative pb-2">
    <!-- Your experiment form/content here -->
</div>
{{end}}
{{define "page_scripts"}}
<script src="/assets/experiments/{NN-slug}/script.js"></script>
{{end}}
```

**Conventions:**
- Must define `experiment_body` block
- Optional `page_scripts` block for JS
- Uses shared `layout.html` for page chrome (hero, index link, admin intro)
- Style with Tailwind utility classes only

### 3. Create Static Assets (if needed)

Create `static/assets/experiments/{NN-slug}/` for JS/CSS files.

### 4. Add API Routes (if needed)

If experiment has form submissions, add routes in `cmd/web/routes.go`:

```go
// Experiment API routes (no language prefix)
{http.MethodPost, "/experiments/my-new-experiment/action", a.myNewExperimentHandler},
```

### 5. Add Handler (if needed)

Add handler in `cmd/web/experiments_handlers.go`:

```go
func (a *app) myNewExperimentHandler(w http.ResponseWriter, r *http.Request) {
    if !experiments.Enabled(r.Context(), "my-new-experiment") {
        http.Error(w, "experiment disabled", http.StatusNotFound)
        return
    }
    // Handle form submission
}
```

### 6. Test

1. Run `go build ./...`
2. Start server
3. Visit `/experiments/my-new-experiment`
4. Verify experiment appears on home page
5. Test admin toggle at `/god/experiments`

## i18n

Translations are stored in `experiment_translations` table via admin UI. No code changes needed for translations.

## Notes

- New experiments default to enabled with prefix order
- Admin can disable/reorder experiments via `/god/experiments`
- DB only stores overrides, not defaults
- No migrations needed — `queryAll()` handles missing experiments with fallback logic
