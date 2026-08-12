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

# DB

- Concurrent writes are safe: `*sql.DB` is thread-safe and pools connections.
- SQLite (Turso) allows one writer at a time. Writes queue inside the DB.
- Multi-statement writes: use `db.BeginTx` transactions, or interleaving requests can corrupt operations halfway through.
- Watch for `database is locked` errors under write contention.

# CSS

- Styles live split by subject in `static/css/` and are concatenated at startup by `concatCSS()` in `cmd/web/web.go` (see the README "Styling" section).
- `static/css/breakpoints.css` holds the responsive media queries — the mobile/desktop breakpoints.
- Reuse the established breakpoints. Do not invent new ones:
  - `768px` — the mobile/desktop split (`max-width: 768px` for mobile overrides, `min-width: 768px` for desktop overrides).
  - `640px` — a smaller tier, used by the `.sm\:inline` utility.
- If a media query only affects one component, co-locate it in that component's file (e.g. `.article-table` lives in `prose.css`). Put global responsive rules in `breakpoints.css`.
- Remember the concat order — theme, base, components, prose, animations, breakpoints, utilities — when a rule must win. Equal-specificity rules later in the list win.

# Versioning

- The app version lives in the root `VERSION` file (`x.y.z`) and is shown in the site footer.
- "bump" / "commit and push" / "push" / "push to remote" / "release" → bump the **patch** version (`0.1.0 → 0.1.1`) **before** committing, and include it in that commit.
- A plain "commit" (no push) does **not** bump, unless explicitly stated.
- Always bump **patch** unless the user explicitly asks for minor or major.
- Always stage all changes with `git add -A` before every commit.
