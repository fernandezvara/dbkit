package dbkit

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

// BatchSize is the default batch size for batch operations.
const BatchSize = 100

// BatchInsert inserts records in batches to avoid exceeding PostgreSQL limits.
// Returns the total number of rows affected.
//
// Usage:
//
//	users := []User{{Name: "A"}, {Name: "B"}, ...}
//	count, err := dbkit.BatchInsert(ctx, db, users, 100)
func BatchInsert[T any](ctx context.Context, db bun.IDB, items []T, batchSize int) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}

	if batchSize <= 0 {
		batchSize = BatchSize
	}

	var totalRows int64

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		result, err := db.NewInsert().Model(&batch).Exec(ctx)
		if err != nil {
			return totalRows, wrapError(err, "BatchInsert")
		}

		rows, _ := result.RowsAffected()
		totalRows += rows
	}

	return totalRows, nil
}

// BatchUpdate updates records in batches.
// Returns the total number of rows affected.
//
// Usage:
//
//	users := []User{{ID: "1", Name: "Updated1"}, {ID: "2", Name: "Updated2"}}
//	count, err := dbkit.BatchUpdate(ctx, db, users, 100)
func BatchUpdate[T any](ctx context.Context, db bun.IDB, items []T, batchSize int) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}

	if batchSize <= 0 {
		batchSize = BatchSize
	}

	var totalRows int64

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		for j := range batch {
			result, err := db.NewUpdate().Model(&batch[j]).WherePK().Exec(ctx)
			if err != nil {
				return totalRows, wrapError(err, "BatchUpdate")
			}
			rows, _ := result.RowsAffected()
			totalRows += rows
		}
	}

	return totalRows, nil
}

// BatchDelete deletes records in batches by their IDs.
// Returns the total number of rows affected.
//
// Usage:
//
//	ids := []string{"id1", "id2", "id3"}
//	count, err := dbkit.BatchDelete[User](ctx, db, ids, 100)
func BatchDelete[T any](ctx context.Context, db bun.IDB, ids []string, batchSize int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	if batchSize <= 0 {
		batchSize = BatchSize
	}

	var totalRows int64

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		batch := ids[i:end]
		var model T
		result, err := db.NewDelete().
			Model(&model).
			Where("id IN (?)", bun.In(batch)).
			Exec(ctx)
		if err != nil {
			return totalRows, wrapError(err, "BatchDelete")
		}

		rows, _ := result.RowsAffected()
		totalRows += rows
	}

	return totalRows, nil
}

// BatchUpsert performs upsert (insert or update) in batches.
// conflictColumns specifies which columns to check for conflicts.
// updateColumns specifies which columns to update on conflict.
//
// Usage:
//
//	users := []User{{Email: "a@example.com", Name: "A"}, ...}
//	count, err := dbkit.BatchUpsert(ctx, db, users, []string{"email"}, []string{"name", "updated_at"}, 100)
func BatchUpsert[T any](ctx context.Context, db bun.IDB, items []T, conflictColumns, updateColumns []string, batchSize int) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}

	if batchSize <= 0 {
		batchSize = BatchSize
	}

	var totalRows int64

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		q := db.NewInsert().Model(&batch).On("CONFLICT (" + joinColumns(conflictColumns) + ") DO UPDATE")

		for _, col := range updateColumns {
			q = q.Set(col + " = EXCLUDED." + col)
		}

		result, err := q.Exec(ctx)
		if err != nil {
			return totalRows, wrapError(err, "BatchUpsert")
		}

		rows, _ := result.RowsAffected()
		totalRows += rows
	}

	return totalRows, nil
}

// InTransaction executes a function within a transaction.
// This is an alias for DBKit.Transaction for use with plain bun.IDB.
//
// Usage:
//
//	err := dbkit.InTransaction(ctx, db, func(ctx context.Context, tx bun.Tx) error {
//	    // do work
//	    return nil
//	})
func InTransaction(ctx context.Context, db *bun.DB, fn func(ctx context.Context, tx bun.Tx) error) error {
	return db.RunInTx(ctx, nil, fn)
}

// BulkInsertReturning inserts records and returns the inserted rows with generated values.
//
// Usage:
//
//	users := []User{{Name: "A"}, {Name: "B"}}
//	inserted, err := dbkit.BulkInsertReturning(ctx, db, users)
//	// inserted now has IDs filled in
func BulkInsertReturning[T any](ctx context.Context, db bun.IDB, items []T) ([]T, error) {
	if len(items) == 0 {
		return items, nil
	}

	_, err := db.NewInsert().Model(&items).Returning("*").Exec(ctx)
	if err != nil {
		return nil, wrapError(err, "BulkInsertReturning")
	}

	return items, nil
}

// Exists checks if any record matches the query.
//
// Usage:
//
//	exists, err := dbkit.Exists[User](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
//	    return q.Where("email = ?", email)
//	})
func Exists[T any](ctx context.Context, db bun.IDB, queryFn func(*bun.SelectQuery) *bun.SelectQuery) (bool, error) {
	var model T
	q := db.NewSelect().Model(&model)
	if queryFn != nil {
		q = queryFn(q)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		return false, wrapError(err, "Exists")
	}

	return exists, nil
}

// Count returns the count of records matching the query.
//
// Usage:
//
//	count, err := dbkit.Count[User](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
//	    return q.Where("active = ?", true)
//	})
func Count[T any](ctx context.Context, db bun.IDB, queryFn func(*bun.SelectQuery) *bun.SelectQuery) (int, error) {
	var model T
	q := db.NewSelect().Model(&model)
	if queryFn != nil {
		q = queryFn(q)
	}

	count, err := q.Count(ctx)
	if err != nil {
		return 0, wrapError(err, "Count")
	}

	return count, nil
}

// Pluck extracts a single column from matching records.
//
// Usage:
//
//	emails, err := dbkit.Pluck[User, string](ctx, db, "email", func(q *bun.SelectQuery) *bun.SelectQuery {
//	    return q.Where("active = ?", true)
//	})
func Pluck[T any, V any](ctx context.Context, db bun.IDB, column string, queryFn func(*bun.SelectQuery) *bun.SelectQuery) ([]V, error) {
	var model T
	var values []V

	q := db.NewSelect().Model(&model).Column(column)
	if queryFn != nil {
		q = queryFn(q)
	}

	err := q.Scan(ctx, &values)
	if err != nil {
		return nil, wrapError(err, "Pluck")
	}

	return values, nil
}

// UpdateReturning updates a record and returns the updated row.
//
// Usage:
//
//	user.Name = "Updated"
//	updated, err := dbkit.UpdateReturning(ctx, db, &user)
func UpdateReturning[T any](ctx context.Context, db bun.IDB, model *T) (*T, error) {
	_, err := db.NewUpdate().Model(model).WherePK().Returning("*").Exec(ctx)
	if err != nil {
		return nil, wrapError(err, "UpdateReturning")
	}
	return model, nil
}

// DeleteReturning deletes a record and returns the deleted row.
//
// Usage:
//
//	deleted, err := dbkit.DeleteReturning(ctx, db, &user)
func DeleteReturning[T any](ctx context.Context, db bun.IDB, model *T) (*T, error) {
	_, err := db.NewDelete().Model(model).WherePK().Returning("*").Exec(ctx)
	if err != nil {
		return nil, wrapError(err, "DeleteReturning")
	}
	return model, nil
}

// FindOrCreate finds a record or creates it if it doesn't exist.
// Returns the record and a boolean indicating if it was created.
//
// Usage:
//
//	user, created, err := dbkit.FindOrCreate(ctx, db, &User{Email: "test@example.com"},
//	    func(q *bun.SelectQuery) *bun.SelectQuery {
//	        return q.Where("email = ?", "test@example.com")
//	    })
func FindOrCreate[T any](ctx context.Context, db bun.IDB, model *T, findFn func(*bun.SelectQuery) *bun.SelectQuery) (*T, bool, error) {
	// Try to find first
	var found T
	q := db.NewSelect().Model(&found)
	if findFn != nil {
		q = findFn(q)
	}

	err := q.Scan(ctx)
	if err == nil {
		return &found, false, nil
	}

	if !IsNotFound(err) {
		return nil, false, wrapError(err, "FindOrCreate.Find")
	}

	// Not found, create it
	_, err = db.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		// Check if it was created by another process (race condition)
		if IsDuplicate(err) {
			var retry T
			retryQ := db.NewSelect().Model(&retry)
			if findFn != nil {
				retryQ = findFn(retryQ)
			}
			if retryErr := retryQ.Scan(ctx); retryErr == nil {
				return &retry, false, nil
			}
		}
		return nil, false, wrapError(err, "FindOrCreate.Create")
	}

	return model, true, nil
}

// RawQuery executes a raw SQL query and scans results into the destination.
//
// Usage:
//
//	var results []map[string]interface{}
//	err := dbkit.RawQuery(ctx, db, &results, "SELECT * FROM users WHERE age > ?", 18)
func RawQuery(ctx context.Context, db bun.IDB, dest interface{}, query string, args ...interface{}) error {
	return db.NewRaw(query, args...).Scan(ctx, dest)
}

// RawExec executes a raw SQL statement.
//
// Usage:
//
//	result, err := dbkit.RawExec(ctx, db, "UPDATE users SET active = ? WHERE last_login < ?", false, cutoffDate)
func RawExec(ctx context.Context, db bun.IDB, query string, args ...interface{}) (sql.Result, error) {
	return db.ExecContext(ctx, query, args...)
}
