-- Experiments are hardcoded in code (internal/experiments). This table only
-- stores the admin-tunable state: whether each experiment is visible and its
-- display order. Rows are seeded from the code registry at startup (INSERT OR
-- IGNORE), so this table never defines what experiments exist.
CREATE TABLE experiments (
    slug    TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 0,
    sort    INTEGER NOT NULL DEFAULT 0
);
