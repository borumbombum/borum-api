-- Migration 0009: Create CardDAV tables
-- Creates address_books and contacts tables for sovereign contact backup.
-- Default address book is pre-created for single-user use.

CREATE TABLE IF NOT EXISTS address_books (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL DEFAULT 'default',
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS contacts (
    id TEXT PRIMARY KEY,
    address_book_id TEXT NOT NULL,
    path TEXT NOT NULL,
    vcard_data TEXT NOT NULL,
    etag TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    deleted_at TEXT,
    FOREIGN KEY (address_book_id) REFERENCES address_books(id),
    UNIQUE(address_book_id, path)
);

CREATE INDEX IF NOT EXISTS idx_contacts_address_book_id ON contacts(address_book_id);
CREATE INDEX IF NOT EXISTS idx_contacts_path ON contacts(path);
CREATE INDEX IF NOT EXISTS idx_address_books_user_id ON address_books(user_id);

-- Pre-create default address book for single-user sovereign backup
INSERT OR IGNORE INTO address_books (id, user_id, name, description)
VALUES ('default', 'default', 'Contacts', 'Default address book for sovereign contact backup');
