// Package backup — VCF contact export.
// Exports all contacts from the database as a single VCF file, following the
// same atomic-write and retention patterns as the SQL backup.
package backup

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupVCF exports all contacts as a VCF file in BACKUP_DIR.
// It is safe to call from the task scheduler; errors are logged but never
// crash the server.
func BackupVCF(db *sql.DB) error {
	dir := backupDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	rows, err := db.Query(`SELECT vcard_data FROM contacts WHERE deleted_at IS NULL`)
	if err != nil {
		return fmt.Errorf("query contacts: %w", err)
	}
	defer rows.Close()

	var sb strings.Builder
	count := 0
	for rows.Next() {
		var vcardData string
		if err := rows.Scan(&vcardData); err != nil {
			log.Printf("vcf backup: scan: %v", err)
			continue
		}
		trimmed := strings.TrimSpace(vcardData)
		if trimmed == "" {
			continue
		}
		sb.WriteString(trimmed)
		if !strings.HasSuffix(trimmed, "\n") {
			sb.WriteString("\n")
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("vcf backup rows: %w", err)
	}

	if count == 0 {
		log.Printf("vcf backup: no contacts to export")
		return nil
	}

	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("borum-contacts-%s.vcf", date)
	outPath := filepath.Join(dir, filename)

	tmpPath := outPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(sb.String()), 0644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("write vcf: %w", err)
	}

	if err := os.Rename(tmpPath, outPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename vcf: %w", err)
	}

	info, err := os.Stat(outPath)
	if err == nil {
		log.Printf("vcf backup: writing to %s (%s, %d contacts)", outPath, humanSize(info.Size()), count)
	}

	if err := deleteOldVCF(dir); err != nil {
		log.Printf("vcf backup: warning: failed to delete old vcf backups: %v", err)
	}

	return nil
}

// deleteOldVCF removes .vcf backup files older than retention days.
func deleteOldVCF(dir string) error {
	retention := retentionDays()
	cutoff := time.Now().AddDate(0, 0, -retention)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".vcf") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, e.Name())
			if err := os.Remove(path); err != nil {
				log.Printf("vcf backup: warning: could not delete %s: %v", path, err)
			} else {
				log.Printf("vcf backup: deleted old backup %s", e.Name())
			}
		}
	}
	return nil
}
