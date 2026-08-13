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

- Styles live split by subject in `static/css/` and are concatenated at startup by `concatCSS()` in `cmd/web/web.go` (see the README "Styling" section).
- `static/css/breakpoints.css` holds the responsive media queries — the mobile/desktop breakpoints.
- Reuse the established breakpoints. Do not invent new ones:
  - `768px` — the mobile/desktop split (`max-width: 768px` for mobile overrides, `min-width: 768px` for desktop overrides).
  - `640px` — a smaller tier, used by the `.sm\:inline` utility.
- If a media query only affects one component, co-locate it in that component's file (e.g. `.article-table` lives in `prose.css`). Put global responsive rules in `breakpoints.css`.
- Remember the concat order — theme, base, components, prose, animations, breakpoints, utilities — when a rule must win. Equal-specificity rules later in the list win.

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

# Versioning

- The app version lives in the root `VERSION` file (`x.y.z`) and is shown in the site footer.
- "bump" / "commit and push" / "push" / "push to remote" / "release" → bump the **patch** version (`0.1.0 → 0.1.1`), then commit, then push — always together.
- A plain "commit" (no push) does **not** bump, unless explicitly stated.
- Always bump **patch** unless the user explicitly asks for minor or major.
- Always stage all changes with `git add -A` before every commit.
- **Never** ask, propose, or auto-run a bump/commit/push after finishing a task. The user tests every change first and orders bump/commit/push explicitly when ready.
