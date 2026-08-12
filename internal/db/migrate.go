// Package db owns the SQLite (Turso) schema and applies it at startup.
// Migrations are plain .sql files embedded from migrations/ and applied in
// filename order, tracked in the schema_migrations table so each runs once.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate applies every embedded migration that has not run yet, in filename
// order. Each migration runs inside its own transaction, statement by
// statement, because the remote libsql driver does not accept a multi-statement
// Exec. It is safe to call repeatedly: applied versions are skipped.
func Migrate(ctx context.Context, d *sql.DB) error {
	if _, err := d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	names, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	sort.Slice(names, func(i, j int) bool { return names[i].Name() < names[j].Name() })

	for _, entry := range names {
		version := entry.Name()
		var applied bool
		err := d.QueryRowContext(ctx,
			`SELECT 1 FROM schema_migrations WHERE version = ?`, version,
		).Scan(&applied)
		switch {
		case err == sql.ErrNoRows:
		case err != nil:
			return fmt.Errorf("check migration %s: %w", version, err)
		default:
			continue
		}

		b, err := migrationFS.ReadFile("migrations/" + version)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", version, err)
		}
		for _, stmt := range splitStatements(string(b)) {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("apply migration %s: %w", version, err)
			}
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version) VALUES (?)`, version,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
	}
	return nil
}

// splitStatements splits a migration file into individual SQL statements. It
// tracks single-quoted string literals (including '' escapes) and -- comments
// so semicolons inside the seed data do not split a statement mid-literal.
func splitStatements(sqlText string) []string {
	var stmts []string
	var cur []rune
	inStr := false
	runes := []rune(sqlText)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if inStr {
			cur = append(cur, c)
			if c == '\'' {
				if i+1 < len(runes) && runes[i+1] == '\'' {
					cur = append(cur, '\'')
					i++
				} else {
					inStr = false
				}
			}
			continue
		}
		switch c {
		case '\'':
			inStr = true
			cur = append(cur, c)
		case ';':
			if t := strings.TrimSpace(string(cur)); t != "" {
				stmts = append(stmts, t)
			}
			cur = nil
		case '-':
			if i+1 < len(runes) && runes[i+1] == '-' {
				for i < len(runes) && runes[i] != '\n' {
					i++
				}
			} else {
				cur = append(cur, c)
			}
		default:
			cur = append(cur, c)
		}
	}
	if t := strings.TrimSpace(string(cur)); t != "" {
		stmts = append(stmts, t)
	}
	return stmts
}
