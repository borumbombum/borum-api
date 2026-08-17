Status: [DONE]

# VCF Contact Backup

## Context

The database backup system (`internal/backup/backup.go`) exports all tables as
gzipped SQL dumps on a 24-hour schedule. Contacts stored in the `contacts` table
need the same treatment — a periodic VCF export so contact data is backed up
alongside the rest of the database.

## Requirements

1. **VCF export** — Query all contacts where `deleted_at IS NULL`, re-encode
   each vCard, write to a single `.vcf` file.
2. **Atomic writes** — Write to `.tmp`, then rename. Same pattern as SQL backup.
3. **File naming** — `borum-contacts-YYYY-MM-DD.vcf` (one per day, same-date
   runs overwrite).
4. **Retention** — Delete `.vcf` files older than `BACKUP_RETENTION_DAYS`.
5. **No new scheduler** — Runs inside the existing `backup.Backup()` call.
6. **No new env vars** — Reuses `BACKUP_ACTIVE`, `BACKUP_DIR`,
   `BACKUP_RETENTION_DAYS`.
7. **Graceful degradation** — VCF failure logs a warning but doesn't block the
   SQL backup.

## Files

- `internal/backup/vcf.go` — `BackupVCF(db *sql.DB) error` + `deleteOldVCF()`
- `internal/backup/backup.go` — `Backup()` calls `BackupVCF()` after SQL dump

## Acceptance criteria

- `go build ./...` passes
- With `BACKUP_ACTIVE=1`, `Backup()` produces both `.sql.gz` and `.vcf` files
- VCF file contains valid vCard data for all non-deleted contacts
- Old `.vcf` files are cleaned up after retention period
- VCF failure doesn't break SQL backup
