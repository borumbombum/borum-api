-- Migration 0008: Create article_translations table
-- Uses the same pattern as experiment_translations: base row in articles,
-- translations in article_translations keyed by (slug, lang).
-- No data migration needed: the previous code never wrote PT article rows.

CREATE TABLE IF NOT EXISTS article_translations (
    slug TEXT NOT NULL,
    lang TEXT NOT NULL DEFAULT 'en',
    title TEXT NOT NULL DEFAULT '',
    subtitle TEXT NOT NULL DEFAULT '',
    excerpt TEXT NOT NULL DEFAULT '',
    image_caption TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (slug, lang)
);
