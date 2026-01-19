package dbkit

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

// SoftDelete marks a model as deleted by setting the DeletedAt field.
// The model must embed SoftDeletableModel or have a DeletedAt field.
//
// Usage:
//
//	err := dbkit.SoftDelete(ctx, db, &user)
func SoftDelete[T any](ctx context.Context, db bun.IDB, model *T) (sql.Result, error) {
	now := time.Now()
	return db.NewUpdate().
		Model(model).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		WherePK().
		Exec(ctx)
}

// SoftDeleteByID marks a record as deleted by its ID.
//
// Usage:
//
//	err := dbkit.SoftDeleteByID[User](ctx, db, userID)
func SoftDeleteByID[T any](ctx context.Context, db bun.IDB, id string) (sql.Result, error) {
	now := time.Now()
	var model T
	return db.NewUpdate().
		Model(&model).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
}

// Restore removes the soft delete mark from a model.
//
// Usage:
//
//	err := dbkit.Restore(ctx, db, &user)
func Restore[T any](ctx context.Context, db bun.IDB, model *T) (sql.Result, error) {
	now := time.Now()
	return db.NewUpdate().
		Model(model).
		Set("deleted_at = NULL").
		Set("updated_at = ?", now).
		WherePK().
		Exec(ctx)
}

// RestoreByID removes the soft delete mark from a record by its ID.
//
// Usage:
//
//	err := dbkit.RestoreByID[User](ctx, db, userID)
func RestoreByID[T any](ctx context.Context, db bun.IDB, id string) (sql.Result, error) {
	now := time.Now()
	var model T
	return db.NewUpdate().
		Model(&model).
		Set("deleted_at = NULL").
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
}

// HardDelete permanently removes a soft-deleted record.
// This bypasses the soft delete and actually deletes the record.
//
// Usage:
//
//	err := dbkit.HardDelete(ctx, db, &user)
func HardDelete[T any](ctx context.Context, db bun.IDB, model *T) (sql.Result, error) {
	return db.NewDelete().
		Model(model).
		WherePK().
		ForceDelete().
		Exec(ctx)
}

// HardDeleteByID permanently removes a record by its ID.
//
// Usage:
//
//	err := dbkit.HardDeleteByID[User](ctx, db, userID)
func HardDeleteByID[T any](ctx context.Context, db bun.IDB, id string) (sql.Result, error) {
	var model T
	return db.NewDelete().
		Model(&model).
		Where("id = ?", id).
		ForceDelete().
		Exec(ctx)
}

// NotDeleted returns a query modifier that filters out soft-deleted records.
// Use this with Bun's query builder to exclude deleted records.
//
// Usage:
//
//	var users []User
//	db.NewSelect().Model(&users).Apply(dbkit.NotDeleted).Scan(ctx)
func NotDeleted(q *bun.SelectQuery) *bun.SelectQuery {
	return q.Where("deleted_at IS NULL")
}

// OnlyDeleted returns a query modifier that includes only soft-deleted records.
// Use this to find records that have been soft deleted.
//
// Usage:
//
//	var deletedUsers []User
//	db.NewSelect().Model(&deletedUsers).Apply(dbkit.OnlyDeleted).Scan(ctx)
func OnlyDeleted(q *bun.SelectQuery) *bun.SelectQuery {
	return q.Where("deleted_at IS NOT NULL")
}

// WithDeleted returns a query modifier that includes all records (both deleted and not).
// This is useful when you need to see all records regardless of deletion status.
// Note: By default, models with soft_delete tag are automatically filtered.
//
// Usage:
//
//	var allUsers []User
//	db.NewSelect().Model(&allUsers).Apply(dbkit.WithDeleted).Scan(ctx)
func WithDeleted(q *bun.SelectQuery) *bun.SelectQuery {
	return q.WhereAllWithDeleted()
}
