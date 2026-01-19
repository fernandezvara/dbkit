package dbkit

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
)

// TenantContextKey is the context key for tenant ID.
type TenantContextKey struct{}

// ErrNoTenant is returned when tenant ID is required but not found in context.
var ErrNoTenant = errors.New("dbkit: tenant ID not found in context")

// TenantModel provides tenant isolation for models.
// Embed this in your model structs to add tenant_id field.
//
// Usage:
//
//	type User struct {
//	    bun.BaseModel `bun:"table:users,alias:u"`
//	    dbkit.BaseModel
//	    dbkit.TenantModel
//	    Email string `bun:"email,notnull"`
//	}
type TenantModel struct {
	TenantID string `bun:"tenant_id,notnull"`
}

// WithTenant adds tenant ID to the context.
//
// Usage:
//
//	ctx = dbkit.WithTenant(ctx, "tenant-123")
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantContextKey{}, tenantID)
}

// GetTenant extracts tenant ID from the context.
// Returns empty string if not found.
//
// Usage:
//
//	tenantID := dbkit.GetTenant(ctx)
func GetTenant(ctx context.Context) string {
	if v := ctx.Value(TenantContextKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// RequireTenant extracts tenant ID from context or returns an error.
//
// Usage:
//
//	tenantID, err := dbkit.RequireTenant(ctx)
func RequireTenant(ctx context.Context) (string, error) {
	tenantID := GetTenant(ctx)
	if tenantID == "" {
		return "", ErrNoTenant
	}
	return tenantID, nil
}

// TenantScope returns a query modifier that filters by tenant ID from context.
// Use this with Bun's query builder to scope queries to the current tenant.
//
// Usage:
//
//	var users []User
//	db.NewSelect().Model(&users).Apply(dbkit.TenantScope(ctx)).Scan(ctx)
func TenantScope(ctx context.Context) func(*bun.SelectQuery) *bun.SelectQuery {
	tenantID := GetTenant(ctx)
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		if tenantID != "" {
			return q.Where("tenant_id = ?", tenantID)
		}
		return q
	}
}

// TenantUpdateScope returns a query modifier for update queries.
//
// Usage:
//
//	db.NewUpdate().Model(&user).Apply(dbkit.TenantUpdateScope(ctx)).WherePK().Exec(ctx)
func TenantUpdateScope(ctx context.Context) func(*bun.UpdateQuery) *bun.UpdateQuery {
	tenantID := GetTenant(ctx)
	return func(q *bun.UpdateQuery) *bun.UpdateQuery {
		if tenantID != "" {
			return q.Where("tenant_id = ?", tenantID)
		}
		return q
	}
}

// TenantDeleteScope returns a query modifier for delete queries.
//
// Usage:
//
//	db.NewDelete().Model(&user).Apply(dbkit.TenantDeleteScope(ctx)).WherePK().Exec(ctx)
func TenantDeleteScope(ctx context.Context) func(*bun.DeleteQuery) *bun.DeleteQuery {
	tenantID := GetTenant(ctx)
	return func(q *bun.DeleteQuery) *bun.DeleteQuery {
		if tenantID != "" {
			return q.Where("tenant_id = ?", tenantID)
		}
		return q
	}
}

// SetTenantID sets the tenant ID on a model from context.
// The model must have a TenantID field.
//
// Usage:
//
//	user := &User{Email: "test@example.com"}
//	dbkit.SetTenantID(ctx, user)
//	db.NewInsert().Model(user).Exec(ctx)
func SetTenantID(ctx context.Context, model interface{}) error {
	tenantID := GetTenant(ctx)
	if tenantID == "" {
		return ErrNoTenant
	}

	// Use type assertion to set TenantID
	if tm, ok := model.(interface{ SetTenantID(string) }); ok {
		tm.SetTenantID(tenantID)
		return nil
	}

	// Try to set via pointer to TenantModel
	if tm, ok := model.(*TenantModel); ok {
		tm.TenantID = tenantID
		return nil
	}

	return nil
}

// SetTenantID sets the tenant ID on the model.
func (m *TenantModel) SetTenantID(tenantID string) {
	m.TenantID = tenantID
}

// TenantHook is a Bun query hook that automatically applies tenant filtering.
type TenantHook struct {
	// Column is the tenant ID column name (default: "tenant_id")
	Column string
}

// NewTenantHook creates a new tenant hook.
func NewTenantHook(column string) *TenantHook {
	if column == "" {
		column = "tenant_id"
	}
	return &TenantHook{Column: column}
}

// Tenant represents a tenant entity.
type Tenant struct {
	bun.BaseModel `bun:"table:tenants,alias:t"`

	ID        string `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Name      string `bun:"name,notnull"`
	Subdomain string `bun:"subdomain,notnull,unique"`
	Active    bool   `bun:"active,notnull,default:true"`
	Metadata  string `bun:"metadata,type:jsonb"`

	TimestampedModel
}

// TenantConfig configures multi-tenancy behavior.
type TenantConfig struct {
	// Column is the tenant ID column name
	Column string

	// EnforceOnSelect automatically filters SELECT queries by tenant
	EnforceOnSelect bool

	// EnforceOnUpdate automatically filters UPDATE queries by tenant
	EnforceOnUpdate bool

	// EnforceOnDelete automatically filters DELETE queries by tenant
	EnforceOnDelete bool

	// SetOnInsert automatically sets tenant ID on INSERT
	SetOnInsert bool
}

// DefaultTenantConfig returns the default tenant configuration.
func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		Column:          "tenant_id",
		EnforceOnSelect: true,
		EnforceOnUpdate: true,
		EnforceOnDelete: true,
		SetOnInsert:     true,
	}
}

// TenantIsolation provides methods for tenant-isolated operations.
type TenantIsolation struct {
	db     bun.IDB
	config TenantConfig
}

// NewTenantIsolation creates a new tenant isolation helper.
func NewTenantIsolation(db bun.IDB, config TenantConfig) *TenantIsolation {
	return &TenantIsolation{
		db:     db,
		config: config,
	}
}

// Select creates a tenant-scoped SELECT query.
//
// Usage:
//
//	var users []User
//	ti.Select(ctx).Model(&users).Scan(ctx)
func (ti *TenantIsolation) Select(ctx context.Context) *bun.SelectQuery {
	q := ti.db.NewSelect()
	tenantID := GetTenant(ctx)
	if tenantID != "" && ti.config.EnforceOnSelect {
		q = q.Where(ti.config.Column+" = ?", tenantID)
	}
	return q
}

// Insert creates a query and can set tenant ID automatically.
// Note: You should still set the tenant ID on the model before insert.
func (ti *TenantIsolation) Insert(ctx context.Context) *bun.InsertQuery {
	return ti.db.NewInsert()
}

// Update creates a tenant-scoped UPDATE query.
//
// Usage:
//
//	ti.Update(ctx).Model(&user).WherePK().Exec(ctx)
func (ti *TenantIsolation) Update(ctx context.Context) *bun.UpdateQuery {
	q := ti.db.NewUpdate()
	tenantID := GetTenant(ctx)
	if tenantID != "" && ti.config.EnforceOnUpdate {
		q = q.Where(ti.config.Column+" = ?", tenantID)
	}
	return q
}

// Delete creates a tenant-scoped DELETE query.
//
// Usage:
//
//	ti.Delete(ctx).Model(&user).WherePK().Exec(ctx)
func (ti *TenantIsolation) Delete(ctx context.Context) *bun.DeleteQuery {
	q := ti.db.NewDelete()
	tenantID := GetTenant(ctx)
	if tenantID != "" && ti.config.EnforceOnDelete {
		q = q.Where(ti.config.Column+" = ?", tenantID)
	}
	return q
}

// ValidateTenant checks if the tenant ID in context is valid.
// Returns error if tenant doesn't exist or is not active.
func ValidateTenant(ctx context.Context, db bun.IDB) error {
	tenantID := GetTenant(ctx)
	if tenantID == "" {
		return ErrNoTenant
	}

	var tenant Tenant
	err := db.NewSelect().
		Model(&tenant).
		Where("id = ?", tenantID).
		Where("active = ?", true).
		Scan(ctx)

	if err != nil {
		if IsNotFound(err) {
			return &Error{
				Code:    CodeNotFound,
				Message: "tenant not found or inactive",
				Op:      "ValidateTenant",
				Cause:   ErrNoTenant,
			}
		}
		return wrapError(err, "ValidateTenant")
	}

	return nil
}
