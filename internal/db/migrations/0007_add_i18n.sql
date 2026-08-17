ALTER TABLE articles ADD COLUMN lang TEXT NOT NULL DEFAULT 'en';
ALTER TABLE articles ADD COLUMN translation_of TEXT NOT NULL DEFAULT '';

CREATE TABLE experiment_translations (
    slug TEXT NOT NULL,
    lang TEXT NOT NULL DEFAULT 'en',
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    intro TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (slug, lang)
);
