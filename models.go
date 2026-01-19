package dbkit

import (
	"context"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

// BaseModel provides common fields for all models: ID and timestamps.
// Embed this in your model structs for standard ID and timestamp handling.
//
// Usage:
//
//	type User struct {
//	    bun.BaseModel `bun:"table:users,alias:u"`
//	    dbkit.BaseModel
//	    Email string `bun:"email,notnull,unique"`
//	}
type BaseModel struct {
	ID        string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

// SoftDeletableModel adds soft delete capability to models.
// Embed this alongside BaseModel for soft delete functionality.
//
// Usage:
//
//	type User struct {
//	    bun.BaseModel `bun:"table:users,alias:u"`
//	    dbkit.BaseModel
//	    dbkit.SoftDeletableModel
//	    Email string `bun:"email,notnull,unique"`
//	}
//
// When querying, add a filter to exclude soft-deleted records:
//
//	db.NewSelect().Model(&users).Where("deleted_at IS NULL").Scan(ctx)
//
// To soft delete:
//
//	db.NewUpdate().Model(&user).Set("deleted_at = ?", time.Now()).WherePK().Exec(ctx)
//
// To restore:
//
//	db.NewUpdate().Model(&user).Set("deleted_at = NULL").WherePK().Exec(ctx)
type SoftDeletableModel struct {
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero"`
}

// IsDeleted returns true if the model has been soft deleted.
func (m *SoftDeletableModel) IsDeleted() bool {
	return m.DeletedAt != nil
}

// VersionedModel adds optimistic locking capability to models.
// Embed this alongside BaseModel for version-based conflict detection.
//
// Usage:
//
//	type User struct {
//	    bun.BaseModel `bun:"table:users,alias:u"`
//	    dbkit.BaseModel
//	    dbkit.VersionedModel
//	    Email string `bun:"email,notnull,unique"`
//	}
//
// When updating, include version check:
//
//	result, err := db.NewUpdate().
//	    Model(&user).
//	    Set("version = version + 1").
//	    Where("id = ?", user.ID).
//	    Where("version = ?", user.Version).
//	    Exec(ctx)
//
// Check if update was successful:
//
//	rows, _ := result.RowsAffected()
//	if rows == 0 {
//	    // Conflict detected - record was modified by another process
//	}
type VersionedModel struct {
	Version int64 `bun:"version,notnull,default:1"`
}

// TimestampedModel is an alias for BaseModel for clarity.
// Use this if you only need timestamps without the ID field.
//
// Usage:
//
//	type AuditLog struct {
//	    bun.BaseModel `bun:"table:audit_logs,alias:al"`
//	    ID            int64 `bun:"id,pk,autoincrement"`
//	    dbkit.TimestampedModel
//	    Action string `bun:"action,notnull"`
//	}
type TimestampedModel struct {
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

// FullModel combines BaseModel, SoftDeletableModel, and VersionedModel.
// Use this for models that need all features.
//
// Usage:
//
//	type User struct {
//	    bun.BaseModel `bun:"table:users,alias:u"`
//	    dbkit.FullModel
//	    Email string `bun:"email,notnull,unique"`
//	}
type FullModel struct {
	ID        string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	CreatedAt time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero"`
	Version   int64      `bun:"version,notnull,default:1"`
}

// IsDeleted returns true if the model has been soft deleted.
func (m *FullModel) IsDeleted() bool {
	return m.DeletedAt != nil
}

// BeforeAppendModel is a Bun hook that updates the UpdatedAt timestamp
// before insert or update operations.
var _ bun.BeforeAppendModelHook = (*BaseModel)(nil)

func (m *BaseModel) BeforeAppendModel(ctx context.Context, query schema.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		now := time.Now()
		if m.CreatedAt.IsZero() {
			m.CreatedAt = now
		}
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = time.Now()
	}
	return nil
}

// BeforeAppendModel is a Bun hook for TimestampedModel.
var _ bun.BeforeAppendModelHook = (*TimestampedModel)(nil)

func (m *TimestampedModel) BeforeAppendModel(ctx context.Context, query schema.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		now := time.Now()
		if m.CreatedAt.IsZero() {
			m.CreatedAt = now
		}
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = time.Now()
	}
	return nil
}

// BeforeAppendModel is a Bun hook for FullModel.
var _ bun.BeforeAppendModelHook = (*FullModel)(nil)

func (m *FullModel) BeforeAppendModel(ctx context.Context, query schema.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		now := time.Now()
		if m.CreatedAt.IsZero() {
			m.CreatedAt = now
		}
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = time.Now()
	}
	return nil
}
