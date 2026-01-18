package dbkit

import (
	"context"

	"github.com/uptrace/bun"
)

// FindByID finds a record by its primary key
func FindByID[T any](ctx context.Context, db IDB, id any) (*T, error) {
	model := new(T)

	err := db.NewSelect().
		Model(model).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		return nil, wrapError(err, "FindByID")
	}

	return model, nil
}

// FindByPK finds a record by its primary key (works with composite PKs)
func FindByPK[T any](ctx context.Context, db IDB, model *T) error {
	err := db.NewSelect().
		Model(model).
		WherePK().
		Scan(ctx)

	if err != nil {
		return wrapError(err, "FindByPK")
	}

	return nil
}

// FindOne finds a single record matching the query
func FindOne[T any](ctx context.Context, db IDB, query func(q *bun.SelectQuery) *bun.SelectQuery) (*T, error) {
	model := new(T)

	q := db.NewSelect().Model(model)
	if query != nil {
		q = query(q)
	}

	err := q.Limit(1).Scan(ctx)
	if err != nil {
		return nil, wrapError(err, "FindOne")
	}

	return model, nil
}

// FindAll finds all records matching the query
func FindAll[T any](ctx context.Context, db IDB, query func(q *bun.SelectQuery) *bun.SelectQuery) ([]T, error) {
	var models []T

	q := db.NewSelect().Model(&models)
	if query != nil {
		q = query(q)
	}

	err := q.Scan(ctx)
	if err != nil {
		return nil, wrapError(err, "FindAll")
	}

	return models, nil
}

// Create inserts a new record
func Create[T any](ctx context.Context, db IDB, model *T) error {
	_, err := db.NewInsert().
		Model(model).
		Exec(ctx)

	if err != nil {
		return wrapError(err, "Create")
	}

	return nil
}

// CreateReturning inserts a new record and scans returned values
func CreateReturning[T any](ctx context.Context, db IDB, model *T, columns ...string) error {
	q := db.NewInsert().Model(model)

	if len(columns) > 0 {
		q = q.Returning(columns[0])
		for _, col := range columns[1:] {
			q = q.Returning(col)
		}
	} else {
		q = q.Returning("*")
	}

	_, err := q.Exec(ctx)
	if err != nil {
		return wrapError(err, "CreateReturning")
	}

	return nil
}

// CreateMany inserts multiple records
func CreateMany[T any](ctx context.Context, db IDB, models []T) error {
	if len(models) == 0 {
		return nil
	}

	_, err := db.NewInsert().
		Model(&models).
		Exec(ctx)

	if err != nil {
		return wrapError(err, "CreateMany")
	}

	return nil
}

// CreateManyReturning inserts multiple records and scans returned values
func CreateManyReturning[T any](ctx context.Context, db IDB, models *[]T, columns ...string) error {
	if len(*models) == 0 {
		return nil
	}

	q := db.NewInsert().Model(models)

	if len(columns) > 0 {
		q = q.Returning(columns[0])
		for _, col := range columns[1:] {
			q = q.Returning(col)
		}
	} else {
		q = q.Returning("*")
	}

	_, err := q.Exec(ctx)
	if err != nil {
		return wrapError(err, "CreateManyReturning")
	}

	return nil
}

// Update updates an existing record (by primary key)
func Update[T any](ctx context.Context, db IDB, model *T) error {
	result, err := db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)

	if err != nil {
		return wrapError(err, "Update")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &Error{
			Code:    CodeNotFound,
			Message: "record not found for update",
			Op:      "Update",
		}
	}

	return nil
}

// UpdateReturning updates a record and scans returned values
func UpdateReturning[T any](ctx context.Context, db IDB, model *T, columns ...string) error {
	q := db.NewUpdate().Model(model).WherePK()

	if len(columns) > 0 {
		q = q.Returning(columns[0])
		for _, col := range columns[1:] {
			q = q.Returning(col)
		}
	} else {
		q = q.Returning("*")
	}

	_, err := q.Exec(ctx)
	if err != nil {
		return wrapError(err, "UpdateReturning")
	}

	return nil
}

// UpdateColumns updates only specific columns
func UpdateColumns[T any](ctx context.Context, db IDB, model *T, columns ...string) error {
	result, err := db.NewUpdate().
		Model(model).
		Column(columns...).
		WherePK().
		Exec(ctx)

	if err != nil {
		return wrapError(err, "UpdateColumns")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &Error{
			Code:    CodeNotFound,
			Message: "record not found for update",
			Op:      "UpdateColumns",
		}
	}

	return nil
}

// UpdateWhere updates records matching the query
func UpdateWhere[T any](ctx context.Context, db IDB, model *T, query func(q *bun.UpdateQuery) *bun.UpdateQuery) (int64, error) {
	q := db.NewUpdate().Model(model).OmitZero()
	if query != nil {
		q = query(q)
	}

	result, err := q.Exec(ctx)
	if err != nil {
		return 0, wrapError(err, "UpdateWhere")
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

// Delete deletes a record by primary key
func Delete[T any](ctx context.Context, db IDB, model *T) error {
	result, err := db.NewDelete().
		Model(model).
		WherePK().
		Exec(ctx)

	if err != nil {
		return wrapError(err, "Delete")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &Error{
			Code:    CodeNotFound,
			Message: "record not found for deletion",
			Op:      "Delete",
		}
	}

	return nil
}

// DeleteByID deletes a record by its primary key value
func DeleteByID[T any](ctx context.Context, db IDB, id any) error {
	model := new(T)

	result, err := db.NewDelete().
		Model(model).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return wrapError(err, "DeleteByID")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &Error{
			Code:    CodeNotFound,
			Message: "record not found for deletion",
			Op:      "DeleteByID",
		}
	}

	return nil
}

// DeleteWhere deletes records matching the query
func DeleteWhere[T any](ctx context.Context, db IDB, query func(q *bun.DeleteQuery) *bun.DeleteQuery) (int64, error) {
	model := new(T)

	q := db.NewDelete().Model(model)
	if query != nil {
		q = query(q)
	}

	result, err := q.Exec(ctx)
	if err != nil {
		return 0, wrapError(err, "DeleteWhere")
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

// Exists checks if a record exists matching the query
func Exists[T any](ctx context.Context, db IDB, query func(q *bun.SelectQuery) *bun.SelectQuery) (bool, error) {
	model := new(T)

	q := db.NewSelect().Model(model)
	if query != nil {
		q = query(q)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		return false, wrapError(err, "Exists")
	}

	return exists, nil
}

// ExistsByID checks if a record with the given ID exists
func ExistsByID[T any](ctx context.Context, db IDB, id any) (bool, error) {
	model := new(T)

	exists, err := db.NewSelect().
		Model(model).
		Where("id = ?", id).
		Exists(ctx)

	if err != nil {
		return false, wrapError(err, "ExistsByID")
	}

	return exists, nil
}

// Count counts records matching the query
func Count[T any](ctx context.Context, db IDB, query func(q *bun.SelectQuery) *bun.SelectQuery) (int, error) {
	model := new(T)

	q := db.NewSelect().Model(model)
	if query != nil {
		q = query(q)
	}

	count, err := q.Count(ctx)
	if err != nil {
		return 0, wrapError(err, "Count")
	}

	return count, nil
}

// CountAll counts all records in the table
func CountAll[T any](ctx context.Context, db IDB) (int, error) {
	return Count[T](ctx, db, nil)
}

// Upsert inserts or updates a record based on conflict
func Upsert[T any](ctx context.Context, db IDB, model *T, conflictColumns []string, updateColumns []string) error {
	q := db.NewInsert().
		Model(model).
		On("CONFLICT (" + joinColumns(conflictColumns) + ") DO UPDATE")

	for _, col := range updateColumns {
		q = q.Set(col + " = EXCLUDED." + col)
	}

	_, err := q.Exec(ctx)
	if err != nil {
		return wrapError(err, "Upsert")
	}

	return nil
}

// UpsertMany inserts or updates multiple records based on conflict
func UpsertMany[T any](ctx context.Context, db IDB, models []T, conflictColumns []string, updateColumns []string) error {
	if len(models) == 0 {
		return nil
	}

	q := db.NewInsert().
		Model(&models).
		On("CONFLICT (" + joinColumns(conflictColumns) + ") DO UPDATE")

	for _, col := range updateColumns {
		q = q.Set(col + " = EXCLUDED." + col)
	}

	_, err := q.Exec(ctx)
	if err != nil {
		return wrapError(err, "UpsertMany")
	}

	return nil
}

// Reload refreshes a model from the database
func Reload[T any](ctx context.Context, db IDB, model *T) error {
	err := db.NewSelect().
		Model(model).
		WherePK().
		Scan(ctx)

	if err != nil {
		return wrapError(err, "Reload")
	}

	return nil
}

// Raw executes a raw SQL query and scans results into dest
func Raw[T any](ctx context.Context, db IDB, dest *[]T, query string, args ...any) error {
	err := db.NewRaw(query, args...).Scan(ctx, dest)
	if err != nil {
		return wrapError(err, "Raw")
	}
	return nil
}

// RawOne executes a raw SQL query and scans a single result
func RawOne[T any](ctx context.Context, db IDB, query string, args ...any) (*T, error) {
	model := new(T)
	err := db.NewRaw(query, args...).Scan(ctx, model)
	if err != nil {
		return nil, wrapError(err, "RawOne")
	}
	return model, nil
}

// Exec executes a query without returning results
func Exec(ctx context.Context, db IDB, query string, args ...any) (int64, error) {
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, wrapError(err, "Exec")
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

// joinColumns joins column names with commas
func joinColumns(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	result := cols[0]
	for _, col := range cols[1:] {
		result += ", " + col
	}
	return result
}
