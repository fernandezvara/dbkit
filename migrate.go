package dbkit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Migration represents a single migration to execute
type Migration struct {
	ID          string // Unique identifier (e.g., "001", "20240115120000", or any string)
	Description string // Human-readable description
	SQL         string // SQL statements to execute
}

// MigrationResult represents the result of running migrations
type MigrationResult struct {
	Applied   []AppliedMigration
	Skipped   []string // IDs that were already applied
	TotalTime time.Duration
}

// AppliedMigration represents a successfully applied migration
type AppliedMigration struct {
	ID          string
	Description string
	AppliedAt   time.Time
	Duration    time.Duration
	Checksum    string
}

// migrationsTable is the schema for tracking migrations
const migrationsTable = `
CREATE TABLE IF NOT EXISTS _dbkit_migrations (
    id VARCHAR(255) PRIMARY KEY,
    description TEXT,
    checksum VARCHAR(64) NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms BIGINT NOT NULL
);
`

// Migrate executes migrations in order, skipping already-applied ones
func (db *DBKit) Migrate(ctx context.Context, migrations []Migration) (*MigrationResult, error) {
	start := time.Now()
	result := &MigrationResult{
		Applied: make([]AppliedMigration, 0),
		Skipped: make([]string, 0),
	}

	// Ensure migrations table exists
	if _, err := db.ExecContext(ctx, migrationsTable); err != nil {
		return nil, &Error{
			Code:    CodeUnknown,
			Message: "failed to create migrations table",
			Op:      "Migrate",
			Cause:   err,
		}
	}

	// Get already applied migrations
	applied, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Apply each migration
	for _, m := range migrations {
		checksum := checksumSQL(m.SQL)

		// Check if already applied
		if existing, ok := applied[m.ID]; ok {
			// Verify checksum matches
			if existing != checksum {
				return nil, &Error{
					Code:    CodeUnknown,
					Message: fmt.Sprintf("migration %s has changed (checksum mismatch: expected %s, got %s)", m.ID, existing, checksum),
					Op:      "Migrate",
				}
			}
			result.Skipped = append(result.Skipped, m.ID)
			continue
		}

		// Apply migration
		migrationStart := time.Now()
		if err := db.applyMigration(ctx, m, checksum, migrationStart); err != nil {
			return nil, err
		}
		duration := time.Since(migrationStart)

		result.Applied = append(result.Applied, AppliedMigration{
			ID:          m.ID,
			Description: m.Description,
			AppliedAt:   time.Now(),
			Duration:    duration,
			Checksum:    checksum,
		})
	}

	result.TotalTime = time.Since(start)
	return result, nil
}

// getAppliedMigrations returns a map of migration ID to checksum
func (db *DBKit) getAppliedMigrations(ctx context.Context) (map[string]string, error) {
	var rows []struct {
		ID       string `bun:"id"`
		Checksum string `bun:"checksum"`
	}

	err := db.NewSelect().
		TableExpr("_dbkit_migrations").
		Column("id", "checksum").
		Scan(ctx, &rows)

	if err != nil {
		return nil, wrapError(err, "Migrate.GetApplied")
	}

	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.ID] = row.Checksum
	}
	return result, nil
}

// applyMigration executes a single migration within a transaction
func (db *DBKit) applyMigration(ctx context.Context, m Migration, checksum string, startTime time.Time) error {
	return db.Transaction(ctx, func(tx *Tx) error {
		// Execute migration SQL
		if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
			return &Error{
				Code:    CodeUnknown,
				Message: fmt.Sprintf("migration %s failed: %v", m.ID, err),
				Op:      "Migrate.Apply",
				Query:   truncateSQL(m.SQL, 200),
				Cause:   err,
			}
		}

		// Calculate duration
		durationMs := time.Since(startTime).Milliseconds()

		// Record migration
		_, err := tx.NewRaw(`
            INSERT INTO _dbkit_migrations (id, description, checksum, duration_ms)
            VALUES (?, ?, ?, ?)
        `, m.ID, m.Description, checksum, durationMs).Exec(ctx)

		if err != nil {
			return wrapError(err, "Migrate.Record")
		}

		return nil
	})
}

// MigrationStatus returns the status of all known migrations
func (db *DBKit) MigrationStatus(ctx context.Context, migrations []Migration) ([]MigrationStatusEntry, error) {
	// Ensure migrations table exists
	if _, err := db.ExecContext(ctx, migrationsTable); err != nil {
		return nil, &Error{
			Code:    CodeUnknown,
			Message: "failed to create migrations table",
			Op:      "MigrationStatus",
			Cause:   err,
		}
	}

	applied, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	var result []MigrationStatusEntry
	for _, m := range migrations {
		checksum := checksumSQL(m.SQL)
		entry := MigrationStatusEntry{
			ID:          m.ID,
			Description: m.Description,
			Checksum:    checksum,
		}

		if appliedChecksum, ok := applied[m.ID]; ok {
			entry.Applied = true
			entry.ChecksumMatch = appliedChecksum == checksum
		}

		result = append(result, entry)
	}

	return result, nil
}

// MigrationStatusEntry represents the status of a single migration
type MigrationStatusEntry struct {
	ID            string
	Description   string
	Checksum      string
	Applied       bool
	ChecksumMatch bool // Only relevant if Applied is true
}

// GetAppliedMigrations returns all migrations that have been applied
func (db *DBKit) GetAppliedMigrations(ctx context.Context) ([]AppliedMigration, error) {
	// Ensure migrations table exists
	if _, err := db.ExecContext(ctx, migrationsTable); err != nil {
		return nil, &Error{
			Code:    CodeUnknown,
			Message: "failed to create migrations table",
			Op:      "GetAppliedMigrations",
			Cause:   err,
		}
	}

	var rows []struct {
		ID          string    `bun:"id"`
		Description string    `bun:"description"`
		Checksum    string    `bun:"checksum"`
		AppliedAt   time.Time `bun:"applied_at"`
		DurationMs  int64     `bun:"duration_ms"`
	}

	err := db.NewSelect().
		TableExpr("_dbkit_migrations").
		Column("id", "description", "checksum", "applied_at", "duration_ms").
		OrderExpr("applied_at ASC").
		Scan(ctx, &rows)

	if err != nil {
		return nil, wrapError(err, "GetAppliedMigrations")
	}

	result := make([]AppliedMigration, len(rows))
	for i, row := range rows {
		result[i] = AppliedMigration{
			ID:          row.ID,
			Description: row.Description,
			AppliedAt:   row.AppliedAt,
			Duration:    time.Duration(row.DurationMs) * time.Millisecond,
			Checksum:    row.Checksum,
		}
	}

	return result, nil
}

// checksumSQL creates a SHA256 checksum of SQL content
func checksumSQL(sql string) string {
	hash := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(hash[:])
}

// truncateSQL truncates SQL for error messages
func truncateSQL(sql string, maxLen int) string {
	if len(sql) <= maxLen {
		return sql
	}
	return sql[:maxLen] + "..."
}
