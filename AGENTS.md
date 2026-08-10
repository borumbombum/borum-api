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
