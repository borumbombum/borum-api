// Package backup exports the database to gzipped SQL files.
// It uses the existing database/sql connection (tursogo-serverless) to query
// schema and data, generating a portable SQL dump that can be imported into
// any SQLite-compatible database.
package backup

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Backup exports the database to a gzipped SQL file in BACKUP_DIR.
// It is safe to call from the task scheduler; errors are logged but never
// crash the server. Returns nil immediately if BACKUP_ACTIVE is not "1".
func Backup(db *sql.DB) error {
	if !isActive() {
		return nil
	}

	dir := backupDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	// Query all user tables (exclude internal SQLite tables)
	tables, err := queryTables(db)
	if err != nil {
		return fmt.Errorf("query tables: %w", err)
	}
	log.Printf("backup: querying schema for %d tables", len(tables))

	// Build SQL dump in memory
	var sb strings.Builder
	for _, table := range tables {
		schema, err := querySchema(db, table)
		if err != nil {
			return fmt.Errorf("query schema for %s: %w", table, err)
		}
		sb.WriteString(schema)
		sb.WriteString(";\n\n")

		count, err := exportTable(db, &sb, table)
		if err != nil {
			return fmt.Errorf("export table %s: %w", table, err)
		}
		log.Printf("backup: exporting %s (%d rows)", table, count)
	}

	// Write to temp file, then gzip
	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("borum-%s.sql.gz", date)
	outPath := filepath.Join(dir, filename)

	tmpPath := outPath + ".tmp"
	if err := writeGzipped(tmpPath, sb.String()); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("write gzipped: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, outPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename backup: %w", err)
	}

	// Log file size
	info, err := os.Stat(outPath)
	if err == nil {
		log.Printf("backup: writing to %s (%s)", outPath, humanSize(info.Size()))
	}

	// Delete old backups
	if err := deleteOld(dir); err != nil {
		log.Printf("backup: warning: failed to delete old backups: %v", err)
	}

	return nil
}

// queryTables returns all user-created tables (excludes sqlite_internal_*).
func queryTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// querySchema returns the CREATE TABLE statement for a table.
func querySchema(db *sql.DB, table string) (string, error) {
	var sqlStr string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&sqlStr)
	if err != nil {
		return "", err
	}
	return sqlStr, nil
}

// exportTable writes INSERT statements for all rows in a table.
func exportTable(db *sql.DB, w io.Writer, table string) (int, error) {
	rows, err := db.Query(fmt.Sprintf(`SELECT * FROM %s`, table))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	count := 0
	for rows.Next() {
		// Scan into []interface{} to handle any type
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return count, err
		}

		// Build INSERT statement
		escaped := make([]string, len(cols))
		for i, v := range values {
			escaped[i] = escapeValue(v)
		}

		fmt.Fprintf(w, "INSERT INTO %s VALUES (%s);\n", table, strings.Join(escaped, ", "))
		count++
	}
	return count, rows.Err()
}

// escapeValue converts a value to its SQL literal representation.
func escapeValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case []byte:
		return "'" + escapeString(string(val)) + "'"
	case string:
		return "'" + escapeString(val) + "'"
	case int, int64, float64:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	default:
		return "'" + escapeString(fmt.Sprintf("%v", val)) + "'"
	}
}

// escapeString escapes single quotes for SQL.
func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// writeGzipped writes content to a gzipped file.
func writeGzipped(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	if _, err := gz.Write([]byte(content)); err != nil {
		return err
	}
	return gz.Close()
}

// deleteOld removes backup files older than retention days.
func deleteOld(dir string) error {
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
		if !strings.HasSuffix(e.Name(), ".sql.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, e.Name())
			if err := os.Remove(path); err != nil {
				log.Printf("backup: warning: could not delete %s: %v", path, err)
			} else {
				log.Printf("backup: deleted old backup %s", e.Name())
			}
		}
	}
	return nil
}

// backupDir returns the backup directory from env or default.
func backupDir() string {
	if d := os.Getenv("BACKUP_DIR"); d != "" {
		return d
	}
	return "backups"
}

// isActive reports whether backups are enabled.
// Must be explicitly enabled via BACKUP_ACTIVE=1.
func isActive() bool {
	return os.Getenv("BACKUP_ACTIVE") == "1"
}

// RunOnStartup reports whether a backup should run immediately on server start.
func RunOnStartup() bool {
	return os.Getenv("BACKUP_RUN_ON_STARTUP") == "1"
}

// retentionDays returns the retention period from env or default.
func retentionDays() int {
	if d := os.Getenv("BACKUP_RETENTION_DAYS"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			return n
		}
	}
	return 7
}

// humanSize formats bytes as human-readable string.
func humanSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
