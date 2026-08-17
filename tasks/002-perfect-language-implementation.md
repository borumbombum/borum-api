Status: [DONE]

# Perfect the language (i18n) implementation

## Context

The translation system works end to end for `en` + `pt` but is fragile and
half-finished. `docs/translations.md` describes the design; the reality is:

- Articles use separate rows with a `lang` column; experiments use a separate
  `experiment_translations` table — two different models for the same concept.
- The admin "Portuguese Translation" section for articles cannot save: the
  form posts `_pt` fields that `parseArticleForm` maps to `TitlePT`/`BodyPT`,
  but `content.Save` never writes them, and the update handler rejects the
  post because `title`/`date` are empty. The PT fields also never load on
  edit (only the English row is read).
- `/pt/tags/{tag}` exists but renders English articles (`tagHandler` uses the
  English-only `content.List`).
- `GetTranslationSlug` runs the same query twice (second and third attempts
  are identical) and the whole `translation_of` mechanism is redundant with
  the shared slug.
- hreflang links point at the home page when no translation exists
  (`TranslationSlug` is empty).
- The language list `en`/`pt` is hardcoded in routes, middleware, cache
  invalidation, templates (hreflang), and admin forms. Adding a language
  means editing all of them.
- `internal/i18n` has dead helpers (`Prefix`, `OtherLang`) and `IsValid` is
  never called; `OtherLang` hardcodes the en↔pt pair.
- The `Article` struct carries per-language fields (`TitlePT`, `BodyPT`, ...)
  — a shape that does not scale to a third language.
- `content.Save` duplicates its SQL (near-identical pt vs non-pt branches).
- Translated bodies render through the same raw `{{safe}}` paths as English
  (SECURITY.md Medium #1): any sanitization must cover all languages from one
  write point.
- PT article lists are uncached (`ListByLang` special-cases `en`).

Goal: one consistent, data-driven translation system — clean, secure, and
easy to extend to new languages.

## Requirements

1. **`internal/i18n` is the single source of truth.**
   - `Supported` (or equivalent) drives route prefixes, the language
     detection middleware, hreflang tags, cache invalidation loops, and any
     other per-language iteration.
   - Add a `URLFor(lang, path)`-style helper so templates never hardcode
     language URLs.
   - Remove dead helpers; `IsValid` is used for every lang value entering the
     system.

2. **Fix the article translation flow.**
   - Portuguese (and future-language) article translations save and load
     correctly through the admin UI — no empty-title rows, no false
     validation failures.
   - Remove the per-language struct fields (`TitlePT`, `BodyPT`, ...) in
     favour of per-language rows loaded on demand.
   - Deduplicate the `Save` SQL branches into one statement.

3. **Unify the two data models.**
   - Pick one pattern (same-table `lang` column, or one translations table)
     and apply it to both articles and experiments, migrating existing data.

4. **Fix the known bugs.**
   - `tagHandler` filters by the request language.
   - `GetTranslationSlug` does one query (drop the redundant attempts and the
     `translation_of` dance unless it earns its keep).
   - hreflang never links a non-existent translation: omit the link or point
     it at the canonical page when there is no counterpart.
   - PT article summaries use the same cache strategy as English.

5. **Validate and sanitize at the write point.**
   - Every admin write validates the language via `i18n.IsValid` before
     touching the DB.
   - Sanitize HTML content once, covering every language (see SECURITY.md
     Medium #1 — stored XSS on raw `{{safe}}` rendering).

6. **Add a visible language switcher** in the navigation so visitors can
   change language without editing the URL.

7. **Keep `docs/translations.md` in sync** with the final design, including a
   short "adding a new language" checklist that names every place a new
   language touches.

## Acceptance criteria

- Adding a new language requires only one entry in `i18n.Supported` plus the
  admin UI fields — routes, middleware, hreflang, and cache loops follow
  automatically.
- A Portuguese article translation can be created and edited from the admin
  UI; the PT page renders the translated title/body; the EN page is
  unaffected.
- `/pt/tags/{tag}` lists only Portuguese articles.
- hreflang output is valid: every alternate points to a real published
  page; no empty or home-page URLs.
- Articles and experiments use the same translation mechanism.
- No duplicated or dead i18n code paths remain (`go vet` clean, no unused
  helpers, no duplicated SQL branches).
- All article bodies (every language) are sanitized at the single save point,
  or the residual risk is documented and accepted in SECURITY.md.
- A third language can be added by following `docs/translations.md` and works
  for home, articles, tags, experiments, palette data, and the admin UI.

## Progress

- 2026-08-17: Task opened from the translation-system report. Audit of the
  current i18n implementation (docs/translations.md, internal/i18n, content,
  experiments, routes, templates) completed; findings captured above.
