-- Admin-written intro text, rendered above the experiment's own form. Raw
-- HTML, like article bodies, and shown only when non-empty.
ALTER TABLE experiments ADD COLUMN intro TEXT NOT NULL DEFAULT '';
