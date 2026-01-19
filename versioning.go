package dbkit

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
)

// ErrConflict is returned when an optimistic locking conflict is detected.
var ErrConflict = errors.New("dbkit: optimistic locking conflict - record was modified")

// UpdateWithVersion performs an optimistic locking update.
// It increments the version and only succeeds if the current version matches.
// Returns ErrConflict if the record was modified by another process.
//
// Usage:
//
//	account.Balance += 100
//	err := dbkit.UpdateWithVersion(ctx, db, &account)
//	if errors.Is(err, dbkit.ErrConflict) {
//	    // Handle conflict - reload and retry
//	}
func UpdateWithVersion[T any](ctx context.Context, db bun.IDB, model *T, version int64) error {
	result, err := db.NewUpdate().
		Model(model).
		Set("version = version + 1").
		Where("version = ?", version).
		WherePK().
		Exec(ctx)
	if err != nil {
		return wrapError(err, "UpdateWithVersion")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return wrapError(err, "UpdateWithVersion")
	}

	if rows == 0 {
		return &Error{
			Code:    CodeConflict,
			Message: "optimistic locking conflict - record was modified",
			Op:      "UpdateWithVersion",
			Cause:   ErrConflict,
		}
	}

	return nil
}

// UpdateColumnsWithVersion performs an optimistic locking update on specific columns.
// It increments the version and only succeeds if the current version matches.
//
// Usage:
//
//	err := dbkit.UpdateColumnsWithVersion(ctx, db, &account, account.Version, "balance", "updated_at")
func UpdateColumnsWithVersion[T any](ctx context.Context, db bun.IDB, model *T, version int64, columns ...string) error {
	q := db.NewUpdate().
		Model(model).
		Set("version = version + 1").
		Where("version = ?", version).
		WherePK()

	for _, col := range columns {
		q = q.Column(col)
	}

	result, err := q.Exec(ctx)
	if err != nil {
		return wrapError(err, "UpdateColumnsWithVersion")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return wrapError(err, "UpdateColumnsWithVersion")
	}

	if rows == 0 {
		return &Error{
			Code:    CodeConflict,
			Message: "optimistic locking conflict - record was modified",
			Op:      "UpdateColumnsWithVersion",
			Cause:   ErrConflict,
		}
	}

	return nil
}

// CheckVersion verifies that a record's version matches the expected version.
// Returns ErrConflict if versions don't match.
//
// Usage:
//
//	if err := dbkit.CheckVersion[Account](ctx, db, accountID, expectedVersion); err != nil {
//	    // Version mismatch - reload required
//	}
func CheckVersion[T any](ctx context.Context, db bun.IDB, id string, expectedVersion int64) error {
	var currentVersion int64
	err := db.NewSelect().
		Model((*T)(nil)).
		Column("version").
		Where("id = ?", id).
		Scan(ctx, &currentVersion)
	if err != nil {
		return wrapError(err, "CheckVersion")
	}

	if currentVersion != expectedVersion {
		return &Error{
			Code:    CodeConflict,
			Message: "version mismatch - record was modified",
			Op:      "CheckVersion",
			Cause:   ErrConflict,
		}
	}

	return nil
}

// IsConflict checks if the error is an optimistic locking conflict.
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// RetryOnConflict executes a function and retries on optimistic locking conflicts.
// The function should reload the model and retry the operation.
//
// Usage:
//
//	err := dbkit.RetryOnConflict(ctx, 3, func() error {
//	    // Reload the record
//	    db.NewSelect().Model(&account).WherePK().Scan(ctx)
//	    // Modify and update
//	    account.Balance += 100
//	    return dbkit.UpdateWithVersion(ctx, db, &account, account.Version)
//	})
func RetryOnConflict(ctx context.Context, maxRetries int, fn func() error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !IsConflict(err) {
			return err
		}
		lastErr = err
	}
	return lastErr
}

// VersionedUpdate is a helper struct for building versioned update queries.
type VersionedUpdate[T any] struct {
	db      bun.IDB
	model   *T
	version int64
	columns []string
}

// NewVersionedUpdate creates a new versioned update builder.
func NewVersionedUpdate[T any](db bun.IDB, model *T, version int64) *VersionedUpdate[T] {
	return &VersionedUpdate[T]{
		db:      db,
		model:   model,
		version: version,
	}
}

// Columns specifies which columns to update.
func (v *VersionedUpdate[T]) Columns(cols ...string) *VersionedUpdate[T] {
	v.columns = cols
	return v
}

// Exec executes the versioned update.
func (v *VersionedUpdate[T]) Exec(ctx context.Context) (sql.Result, error) {
	q := v.db.NewUpdate().
		Model(v.model).
		Set("version = version + 1").
		Where("version = ?", v.version).
		WherePK()

	if len(v.columns) > 0 {
		for _, col := range v.columns {
			q = q.Column(col)
		}
	}

	result, err := q.Exec(ctx)
	if err != nil {
		return nil, wrapError(err, "VersionedUpdate.Exec")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, &Error{
			Code:    CodeConflict,
			Message: "optimistic locking conflict - record was modified",
			Op:      "VersionedUpdate.Exec",
			Cause:   ErrConflict,
		}
	}

	return result, nil
}
