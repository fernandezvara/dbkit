package dbkit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// AuditAction represents the type of action being audited.
type AuditAction string

const (
	AuditActionCreate AuditAction = "CREATE"
	AuditActionUpdate AuditAction = "UPDATE"
	AuditActionDelete AuditAction = "DELETE"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	ID        string          `json:"id,omitempty"`
	Action    AuditAction     `json:"action"`
	TableName string          `json:"table_name"`
	RecordID  string          `json:"record_id"`
	OldData   json.RawMessage `json:"old_data,omitempty"`
	NewData   json.RawMessage `json:"new_data,omitempty"`
	UserID    string          `json:"user_id,omitempty"`
	IPAddress string          `json:"ip_address,omitempty"`
	UserAgent string          `json:"user_agent,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// AuditHandler is a function that handles audit entries.
// Implement this to store audit logs in your preferred backend.
type AuditHandler func(ctx context.Context, entry *AuditEntry) error

// AuditConfig configures the audit system.
type AuditConfig struct {
	// Handler is called for each audit entry.
	Handler AuditHandler

	// Tables specifies which tables to audit. If empty, all tables are audited.
	Tables []string

	// ExcludeTables specifies tables to exclude from auditing.
	ExcludeTables []string

	// IncludeOldData includes the old data in update/delete operations.
	IncludeOldData bool

	// IncludeNewData includes the new data in create/update operations.
	IncludeNewData bool

	// UserIDExtractor extracts the user ID from the context.
	UserIDExtractor func(ctx context.Context) string

	// MetadataExtractor extracts additional metadata from the context.
	MetadataExtractor func(ctx context.Context) map[string]interface{}
}

// AuditHook is a Bun query hook that creates audit log entries.
type AuditHook struct {
	config AuditConfig
}

// NewAuditHook creates a new audit hook with the given configuration.
func NewAuditHook(config AuditConfig) *AuditHook {
	return &AuditHook{config: config}
}

// shouldAudit checks if a table should be audited.
func (h *AuditHook) shouldAudit(tableName string) bool {
	// Check exclusions first
	for _, t := range h.config.ExcludeTables {
		if t == tableName {
			return false
		}
	}

	// If Tables is empty, audit all tables
	if len(h.config.Tables) == 0 {
		return true
	}

	// Check if table is in the audit list
	for _, t := range h.config.Tables {
		if t == tableName {
			return true
		}
	}

	return false
}

// CreateEntry creates an audit entry from the context and query.
func (h *AuditHook) CreateEntry(ctx context.Context, action AuditAction, tableName, recordID string, oldData, newData interface{}) *AuditEntry {
	_ = ctx // Suppress unused warning
	entry := &AuditEntry{
		Action:    action,
		TableName: tableName,
		RecordID:  recordID,
		CreatedAt: time.Now(),
	}

	if h.config.UserIDExtractor != nil {
		entry.UserID = h.config.UserIDExtractor(ctx)
	}

	if h.config.MetadataExtractor != nil {
		metadata := h.config.MetadataExtractor(ctx)
		if len(metadata) > 0 {
			entry.Metadata, _ = json.Marshal(metadata)
		}
	}

	if h.config.IncludeOldData && oldData != nil {
		entry.OldData, _ = json.Marshal(oldData)
	}

	if h.config.IncludeNewData && newData != nil {
		entry.NewData, _ = json.Marshal(newData)
	}

	return entry
}

// Auditable is an interface that models can implement to provide audit information.
type Auditable interface {
	// AuditID returns the ID of the record for audit purposes.
	AuditID() string
	// AuditTableName returns the table name for audit purposes.
	AuditTableName() string
}

// AuditableModel is a base model that implements the Auditable interface.
// Embed this in your models to enable audit logging.
//
// Usage:
//
//	type User struct {
//	    bun.BaseModel `bun:"table:users,alias:u"`
//	    dbkit.BaseModel
//	    dbkit.AuditableModel
//	    Email string `bun:"email,notnull,unique"`
//	}
type AuditableModel struct{}

// AuditCreate logs a create action for a model.
// Call this after inserting a record.
//
// Usage:
//
//	_, err := db.NewInsert().Model(&user).Exec(ctx)
//	if err == nil {
//	    dbkit.AuditCreate(ctx, auditor, "users", user.ID, &user)
//	}
func AuditCreate(ctx context.Context, handler AuditHandler, tableName, recordID string, newData interface{}) error {
	if handler == nil {
		return nil
	}

	entry := &AuditEntry{
		Action:    AuditActionCreate,
		TableName: tableName,
		RecordID:  recordID,
		CreatedAt: time.Now(),
	}

	if newData != nil {
		entry.NewData, _ = json.Marshal(newData)
	}

	return handler(ctx, entry)
}

// AuditUpdate logs an update action for a model.
// Call this after updating a record.
//
// Usage:
//
//	oldUser := user  // Copy before update
//	_, err := db.NewUpdate().Model(&user).WherePK().Exec(ctx)
//	if err == nil {
//	    dbkit.AuditUpdate(ctx, auditor, "users", user.ID, &oldUser, &user)
//	}
func AuditUpdate(ctx context.Context, handler AuditHandler, tableName, recordID string, oldData, newData interface{}) error {
	if handler == nil {
		return nil
	}

	entry := &AuditEntry{
		Action:    AuditActionUpdate,
		TableName: tableName,
		RecordID:  recordID,
		CreatedAt: time.Now(),
	}

	if oldData != nil {
		entry.OldData, _ = json.Marshal(oldData)
	}

	if newData != nil {
		entry.NewData, _ = json.Marshal(newData)
	}

	return handler(ctx, entry)
}

// AuditDelete logs a delete action for a model.
// Call this after deleting a record.
//
// Usage:
//
//	dbkit.AuditDelete(ctx, auditor, "users", user.ID, &user)
//	_, err := db.NewDelete().Model(&user).WherePK().Exec(ctx)
func AuditDelete(ctx context.Context, handler AuditHandler, tableName, recordID string, oldData interface{}) error {
	if handler == nil {
		return nil
	}

	entry := &AuditEntry{
		Action:    AuditActionDelete,
		TableName: tableName,
		RecordID:  recordID,
		CreatedAt: time.Now(),
	}

	if oldData != nil {
		entry.OldData, _ = json.Marshal(oldData)
	}

	return handler(ctx, entry)
}

// AuditLog is a database model for storing audit entries.
// Use this if you want to store audit logs in the database.
//
// Create the table with:
//
//	CREATE TABLE audit_logs (
//	    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
//	    action VARCHAR(20) NOT NULL,
//	    table_name VARCHAR(255) NOT NULL,
//	    record_id VARCHAR(255) NOT NULL,
//	    old_data JSONB,
//	    new_data JSONB,
//	    user_id VARCHAR(255),
//	    ip_address VARCHAR(45),
//	    user_agent TEXT,
//	    metadata JSONB,
//	    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
//	);
//	CREATE INDEX idx_audit_logs_table_record ON audit_logs(table_name, record_id);
//	CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
//	CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
type AuditLog struct {
	bun.BaseModel `bun:"table:audit_logs,alias:al"`

	ID        string          `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Action    AuditAction     `bun:"action,notnull"`
	TableName string          `bun:"table_name,notnull"`
	RecordID  string          `bun:"record_id,notnull"`
	OldData   json.RawMessage `bun:"old_data,type:jsonb"`
	NewData   json.RawMessage `bun:"new_data,type:jsonb"`
	UserID    string          `bun:"user_id"`
	IPAddress string          `bun:"ip_address"`
	UserAgent string          `bun:"user_agent"`
	Metadata  json.RawMessage `bun:"metadata,type:jsonb"`
	CreatedAt time.Time       `bun:"created_at,notnull,default:current_timestamp"`
}

// NewDatabaseAuditHandler creates an AuditHandler that stores entries in the database.
//
// Usage:
//
//	handler := dbkit.NewDatabaseAuditHandler(db)
//	dbkit.AuditCreate(ctx, handler, "users", user.ID, &user)
func NewDatabaseAuditHandler(db bun.IDB) AuditHandler {
	return func(ctx context.Context, entry *AuditEntry) error {
		log := &AuditLog{
			Action:    entry.Action,
			TableName: entry.TableName,
			RecordID:  entry.RecordID,
			OldData:   entry.OldData,
			NewData:   entry.NewData,
			UserID:    entry.UserID,
			IPAddress: entry.IPAddress,
			UserAgent: entry.UserAgent,
			Metadata:  entry.Metadata,
			CreatedAt: entry.CreatedAt,
		}

		_, err := db.NewInsert().Model(log).Exec(ctx)
		return err
	}
}

// ContextKey is a type for context keys used by the audit system.
type ContextKey string

const (
	// ContextKeyUserID is the context key for the user ID.
	ContextKeyUserID ContextKey = "dbkit_user_id"
	// ContextKeyIPAddress is the context key for the IP address.
	ContextKeyIPAddress ContextKey = "dbkit_ip_address"
	// ContextKeyUserAgent is the context key for the user agent.
	ContextKeyUserAgent ContextKey = "dbkit_user_agent"
)

// WithAuditContext adds audit context information to a context.
//
// Usage:
//
//	ctx = dbkit.WithAuditContext(ctx, userID, ipAddress, userAgent)
func WithAuditContext(ctx context.Context, userID, ipAddress, userAgent string) context.Context {
	ctx = context.WithValue(ctx, ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, ContextKeyIPAddress, ipAddress)
	ctx = context.WithValue(ctx, ContextKeyUserAgent, userAgent)
	return ctx
}

// DefaultUserIDExtractor extracts the user ID from the context.
func DefaultUserIDExtractor(ctx context.Context) string {
	if v := ctx.Value(ContextKeyUserID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
