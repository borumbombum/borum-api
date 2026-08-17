# ⚠️ CRITICAL WARNING ⚠️

**NEVER COMMIT, PUSH, OR BUMP VERSION WITHOUT EXPLICIT PERMISSION!**

- Do NOT run `git commit`
- Do NOT run `git push`
- Do NOT bump VERSION
- Do NOT run `git add -A` unless explicitly told to commit

**YOU MUST ASK FIRST. WAIT FOR EXPLICIT "commit" or "push" ORDER.**

This rule overrides everything else. Breaking it loses trust.

**ALWAYS VERIFY CODE COMPILES:**
- Run `go build ./...` after every code change
- Never proceed to next task until code compiles cleanly
- If compilation fails, fix it immediately before continuing

---

# Security

- Read `SECURITY.md` before any security work. It is the audit report and to-do list. Remove items only after they are implemented and verified, and never remove unfixed items.
- **NEVER commit WebDAV credentials, Nostr private keys, or any auth tokens** — store in `.env` or environment variables only.
- **NEVER log or expose sensitive data** — no API keys, passwords, or private keys in logs, error messages, or responses.

# Migrations

- **NEVER put sensitive data in migrations** — no passwords, API keys, tokens, personal data, or real article content.
- Migrations are for schema only: CREATE TABLE, ALTER TABLE, indexes.
- Seed data (articles, experiments) goes in separate scripts or is created via the admin UI.
- If a migration contains sensitive data, it's a security breach — fix immediately.

## Response Style

- Use Plain Language for your answers.
- Keep your sentences to 20 words or fewer.
- Use simple, everyday words instead of complex or technical ones where possible.
- Active voice. Address the reader directly ("you").
- Keep necessary technical terms, but explain them briefly on first use.
- State actions, constraints, scope, and expected results explicitly.
- Cut filler, hedging, jargon, and repetition.
- Use short paragraphs and lists to break up dense information.
- If precision and natural phrasing conflict, precision wins.

- IMPORTANT: We need to keep things simple and efficient. Create reusable effcient code whenever possible, following the Twelve-Factor App methodology.

# DB

- Concurrent writes are safe: `*sql.DB` is thread-safe and pools connections.
- SQLite (Turso) allows one writer at a time. Writes queue inside the DB.
- Multi-statement writes: use `db.BeginTx` transactions, or interleaving requests can corrupt operations halfway through.
- Watch for `database is locked` errors under write contention.

# CSS

- **ONLY USE TAILWIND for front-end CSS. Do not add or edit hand-written CSS in `static/css/` (theme, base, components, prose, admin, editor, animations, breakpoints, utilities). Style everything with Tailwind utility classes in templates.**
- Styles live split by subject in `static/css/` and are concatenated at startup by `concatCSS()` in `cmd/web/web.go` (see the README "Styling" section).
- `static/css/breakpoints.css` holds the responsive media queries — the mobile/desktop breakpoints.
- Reuse the established breakpoints. Do not invent new ones:
  - `768px` — the mobile/desktop split (`max-width: 768px` for mobile overrides, `min-width: 768px` for desktop overrides).
  - `640px` — a smaller tier, used by the `.sm\:inline` utility.
- If a media query only affects one component, co-locate it in that component's file (e.g. `.article-table` lives in `prose.css`). Put global responsive rules in `breakpoints.css`.
- Remember the concat order — theme, base, components, prose, animations, breakpoints, utilities — when a rule must win. Equal-specificity rules later in the list win.

# Editor

- Rich-HTML content fields (article body, experiment intro) must use the TipTap editor component, never a plain textarea. Copy the `editor-shell` markup from `god_article_form.html` and load `/vendor/tiptap/tiptap.bundle.js` + `/god-editor.js` in the page's `page_scripts` block.
- `god-editor.js` wires every `[data-editor]` mount on a page. Each mount needs a unique `data-editor` id that matches the id of the hidden textarea it syncs to (e.g. `intro-{{.Slug}}` for per-row experiment intros).
- Plain-text metadata fields (e.g. the article excerpt, shown as a plain subtitle) stay plain inputs; only HTML-rendered content gets an editor.

# Command Palette

- The palette opens with the `:` key. The input starts prefilled with `:`, so users type `:home`, `:q`, etc. directly.
- Commands live in the `paletteCommands` array in `static/app.js`. There is no other registry.
- Each command is one object:
  - `hint` — one line shown in `:help` (keep it short, e.g. `:help / :h \u2014 show command list`).
  - `keys` — strings Tab completion matches against (prefix match, array order = priority).
  - `match(v)` — returns true when the input `v` (already trimmed, starts with `:`) triggers this command.
  - `run()` — what happens on Enter.
  - `instant: true` — optional; runs the command as you type instead of on Enter (only `:q` uses this).
- `:help` is auto-generated from `paletteCommands` + `paletteModes`. Never hand-edit the `:help` list; add a command/mode and it appears.
- The palette always closes after a command runs (`runCommand()` clears the input and calls `closePalette()`).
- Search modes live in `paletteModes` (e.g. `#tag` search, article title search). Each mode is `{ hint, test(v), search(v) }`; the first mode whose `test` passes fills the results.
- Commands are checked before modes, so a command match always wins (e.g. `:#5` is the principle jump, not a tag search).
- The palette reads article/tag data from `/data/articles.json` at load; tags are compiled into `paletteTags`. No new data file for tags.

# Tasks

- Task details live in `tasks/` — one file per task, named `NNN-short-slug.md`.
- This list mirrors those files and must stay in sync. See the `tasks` skill for the full workflow.
- Status markers: `[TODO]` (not started), `[IN_PROGRESS]` (agent working on it), `[DONE]` (verified complete).
- Set `[IN_PROGRESS]` when you start a task, `[DONE]` when done — in both this list and the task file.

- 001 [DONE] Top loading indicator between page navigations
- 002 [DONE] Perfect the language (i18n) implementation
- 003 [TODO] CardDAV contact sync (sovereign backup)

# Documentation

- Detailed docs live in `docs/` — one file per topic.
- Check `docs/` when you need to understand how a subsystem works.
- Current topics: translations (i18n), database schema, admin workflows.

# Languages

- Supported languages: `en` (base), `pt`.
- English is the default language. No URL prefix.
- Other languages use subpath prefix: `/pt/blog/{slug}`, `/pt/experiments/{slug}`.
- Articles store translations as separate rows with `lang` and `translation_of` columns.
- Experiments store translations in `experiment_translations` table keyed by `(slug, lang)`.
- To add a new language: add code to `supportedLangs` list, add route prefixes, add admin fields.
- `hreflang` tags link all translations on every page. `x-default` points to English.
- `html lang` attribute is dynamic, set per request based on URL prefix.
- Admin forms show all languages side by side. Empty fields = no translation.

# Versioning

- The app version lives in the root `VERSION` file (`x.y.z`) and is shown in the site footer.
- "bump" / "commit and push" / "push" / "push to remote" / "release" → bump the **patch** version (`0.1.0 → 0.1.1`), then commit, then push — always together.
- A plain "commit" (no push) does **not** bump, unless explicitly stated.
- Always bump **patch** unless the user explicitly asks for minor or major.
- Always stage all changes with `git add -A` before every commit.
- **Never** ask, propose, or auto-run a bump/commit/push after finishing a task. The user tests every change first and orders bump/commit/push explicitly when ready.
