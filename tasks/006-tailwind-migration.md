Status: [TODO]

# Migrate Legacy CSS to Tailwind Inline Classes

## Goal

Switch from two conflicting CSS systems (legacy custom utilities + Tailwind CDN)
to Tailwind-only inline classes in HTML templates, plus a small set of plain CSS
when totally indispensable. The end result should be a site looking exactly as
now but powered by Tailwind only. In the future, Tailwind CLI will replace the
CDN.

## Context

The site loads two CSS systems that overlap and sometimes conflict:

1. **Legacy CSS** — 10 files in `static/css/` concatenated into `/styles.css`
2. **Tailwind v4** — loaded from CDN via `@tailwindcss/browser@4`

The legacy system has ~300 lines of re-implemented utilities (`flex`,
`items-center`, `mt-3`, `text-center`, `font-weight-600`, etc.) that duplicate
Tailwind. These sometimes have different values or specificity, causing visual
conflicts.

Tailwind Preflight covers all legacy resets. Typography should be applied with
Tailwind classes on each element, not global CSS rules. Animations should use
Tailwind where possible.

## What gets DELETED entirely

| File | Reason |
|---|---|
| `utilities.css` | Every class replaced by Tailwind inline equivalents |
| `base.css` resets | Covered by Tailwind Preflight |
| `base.css` typography | Replaced by Tailwind classes on each element |
| `components.css` | All component classes converted to Tailwind inline |
| `breakpoints.css` | Responsive rules replaced by Tailwind responsive prefixes |
| `admin.css` | Theme-aware form overrides handled by Tailwind `dark:` variant |
| `animations.css` | Replaced by keyframes file + Tailwind arbitrary values |

## What survives as plain CSS (minimal, indispensable only)

### `base.css` (~40 lines) — things that cannot be Tailwind inline

1. Body background (`var(--background)`)
2. Font smoothing (vendor prefixes, not in Preflight)
3. SVG icon sizing (`var(--svg-icon-size)`)
4. Link base (`var(--links-border)`, hover opacity)
5. Heading font-family (same for all, global)
6. Code/pre formatting
7. HR separator
8. Skip-to-content (accessibility)
9. Article body prose (`.article`/`.prose-diary` — DB content, can't edit)

### `keyframes.css` (~50 lines) — `@keyframes` + JS-referenced classes

- `@keyframes` for: `rubberBand`, `flipOutY`, `rise-in`, `loader-sweep`,
  `modal-in`, `heartPop`, `heartParticleBurst`
- JS-referenced class mappings: `.nav-open-anim`, `.nav-close-anim`,
  `.borum-loader`, `.borum-loader-pill`, `.modal-in`, `.heart-particle`
- `prefers-reduced-motion` media query

### `components.css` (~80 lines) — JS-driven classes only

- Command palette: `.command-palette`, `.command-input`, `.command-hint`,
  `.command-results`, `.command-result`
- Modals: `.modal-backdrop`, `.modal`, `.modal-close`, `.modal-title`,
  `.modal-body`
- Principles: `.principle-row`, `.principle-num`, `.principle-title`,
  `.principle-detail`, `.principle-item`, `.principle-scroll`
- Heart button: `.heart-btn`
- Read more: `.read-more-btn`, `.read-more-text`
- Highlight: `.highlight`

### `prose.css` — `.article`/`.prose-diary` only (DB content)

### `editor.css` — unchanged (TipTap internals)

### `highlight.css` — unchanged (hljs theme)

### `theme.css` — unchanged (CSS custom properties + themes)

## Template migration

Every legacy class in templates becomes Tailwind inline. Examples:

| Before | After |
|---|---|
| `wrapper wrapper-wide text-center pb-2` | `mx-auto max-w-[768px] px-4 text-center pb-8` |
| `article-hero` | `font-['Cormorant'] italic font-semibold text-[clamp(...)] leading-none text-center text-balance` |
| `article-description` | `font-mono text-[clamp(...)] font-light text-center leading-tight` |
| `h4 font-italic font-weight-600` | `font-['Cormorant'] italic font-semibold text-[clamp(...)] leading-none` |
| `pill` | `inline-block px-2 py-1 font-mono text-[clamp(...)] ... rounded-full lowercase bg-[var(--high-background)]` |
| `rise-in text-center` | `animate-[rise-in_0.9s_var(--ease-shizuka)_both] text-center` |
| `bg-grey pb-2` | `bg-[var(--high-background-gradient)] pb-8` |
| `img-rounded img-rotate` | `rounded-lg rotate-[-1deg] transition-transform duration-300 hover:rotate-0` |
| `no-decoration` | `no-underline border-transparent` |
| `d-none` | `hidden` |

## CSS custom properties in Tailwind

Tailwind v4 browser supports arbitrary values:

- Colors: `text-[var(--color-faint)]`, `bg-[var(--color-ink)]`
- Font sizes: `text-[clamp(0.88rem,0.83rem+0.263vw,0.984rem)]`
- Fonts: `font-['Cormorant']`, `font-mono`
- Animations: `animate-[rise-in_0.9s_var(--ease-shizuka)_both]`
- Backgrounds: `bg-[var(--high-background-gradient)]`

## Updated `web.go` cssFiles

```go
var cssFiles = []string{
    "theme.css",
    "base.css",
    "components.css",
    "prose.css",
    "editor.css",
    "highlight.css",
    "keyframes.css",
}
```

Removed: `utilities.css`, `admin.css`, `breakpoints.css`, `animations.css`.

## Files

- `static/css/utilities.css` — delete
- `static/css/admin.css` — delete
- `static/css/breakpoints.css` — delete
- `static/css/animations.css` — delete
- `static/css/base.css` — slim to ~40 lines (resets removed, typography removed)
- `static/css/components.css` — slim to ~80 lines (JS-driven classes only)
- `static/css/prose.css` — slim to `.article`/`.prose-diary` only
- `static/css/keyframes.css` — new file, extracted from animations.css
- `cmd/web/web.go` — update cssFiles list
- `cmd/web/templates/home.html` — migrate all legacy classes to Tailwind
- `cmd/web/templates/article.html` — migrate
- `cmd/web/templates/tag.html` — migrate
- `cmd/web/templates/404.html` — migrate
- `cmd/web/templates/footer.html` — migrate
- `cmd/web/templates/header.html` — migrate
- `cmd/web/templates/login.html` — migrate
- `cmd/web/templates/love_bar.html` — migrate
- `cmd/web/templates/god_articles.html` — migrate
- `cmd/web/templates/god_article_form.html` — migrate
- `cmd/web/templates/god_carddav.html` — migrate
- `cmd/web/templates/god_carddav_contacts.html` — migrate
- `cmd/web/templates/god_experiments.html` — migrate
- `cmd/web/templates/god_drawer.html` — migrate
- `cmd/web/templates/god_dashboard.html` — migrate
- `cmd/web/templates/experiments/layout.html` — migrate

## Migration order

1. Create `keyframes.css` (extract from `animations.css`)
2. Slim `base.css` to minimal
3. Slim `components.css` to JS-driven classes only
4. Slim `prose.css` to `.article`/`.prose-diary` only
5. Delete `utilities.css`, `admin.css`, `breakpoints.css`, old `animations.css`
6. Update `web.go` cssFiles list
7. Migrate templates one by one (public first, then admin)
8. `go build`, visual QA each page

## Acceptance criteria

- `go build ./...` passes
- Site looks pixel-identical (light + dark mode)
- `utilities.css` deleted
- No legacy utility classes in any template
- No legacy reset rules (Preflight handles them)
- Typography on every element via Tailwind classes
- Only CSS files remain: `theme.css`, `base.css`, `components.css`, `prose.css`,
  `editor.css`, `highlight.css`, `keyframes.css`
