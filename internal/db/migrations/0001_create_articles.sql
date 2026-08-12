CREATE TABLE articles (
    slug          TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    subtitle      TEXT NOT NULL DEFAULT '',
    date          TEXT NOT NULL,
    tags          TEXT NOT NULL DEFAULT '[]',
    excerpt       TEXT NOT NULL DEFAULT '',
    image         TEXT NOT NULL DEFAULT '',
    image_caption TEXT NOT NULL DEFAULT '',
    initial_love  INTEGER NOT NULL DEFAULT 0,
    star          INTEGER NOT NULL DEFAULT 0,
    featured      INTEGER NOT NULL DEFAULT 0,
    body          TEXT NOT NULL
);

CREATE INDEX idx_articles_date ON articles(date);
