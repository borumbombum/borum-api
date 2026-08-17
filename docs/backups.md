# Backups

This document explains how the database backup system works.

## Overview

Backups export the Turso database to gzipped SQL files. They use the existing
`tursogo-serverless` SDK connection — no external binaries required.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_ACTIVE` | `0` (disabled) | Set to `1` to enable daily backups |
| `BACKUP_RUN_ON_STARTUP` | `0` (disabled) | Set to `1` to run backup on server start |
| `BACKUP_DIR` | `backups` | Directory to store backups |
| `BACKUP_RETENTION_DAYS` | `7` | Days to keep backups |

**Important:** Backups are disabled by default. You must set `BACKUP_ACTIVE=1`
to enable them.

## How It Works

1. Query all tables from the database
2. For each table, export schema (CREATE TABLE) and data (INSERT statements)
3. Write SQL dump to a temporary file
4. Gzip and rename to `borum-YYYY-MM-DD.sql.gz`
5. Delete backups older than retention period

## Schedule

- **Daily backups:** Run every 24 hours when `BACKUP_ACTIVE=1`
- **Startup backup:** Run once on server start when `BACKUP_RUN_ON_STARTUP=1`

## File Structure

```
backups/
├── borum-2026-08-17.sql.gz
├── borum-2026-08-16.sql.gz
├── borum-2026-08-15.sql.gz
└── ...
```

## Restoring a Backup

```bash
# Decompress
gunzip backups/borum-2026-08-17.sql.gz

# Import into SQLite
sqlite3 borum.db < backups/borum-2026-08-17.sql

# Or import into Turso
turso db shell my-database < backups/borum-2026-08-17.sql
```

## Tables Backed Up

| Table | Description |
|-------|-------------|
| `articles` | Blog posts (English base) |
| `article_translations` | Portuguese translations |
| `experiments` | Admin state (enabled, sort) |
| `experiment_translations` | Portuguese translations |
| `sessions` | User sessions |
| `schema_migrations` | Migration tracking |

## Safety

- Backups write to a temp file first, then atomic rename
- Errors are logged but never crash the server
- Startup backup runs in a goroutine (non-blocking)
- Old backups are deleted before creating new ones

## Console Output

```
backup: starting daily backup
backup: querying schema for 6 tables
backup: exporting articles (8 rows)
backup: exporting article_translations (0 rows)
backup: exporting experiments (1 rows)
backup: exporting experiment_translations (0 rows)
backup: exporting sessions (0 rows)
backup: exporting schema_migrations (8 rows)
backup: writing to backups/borum-2026-08-17.sql.gz (12.3 KB)
backup: completed successfully
```
