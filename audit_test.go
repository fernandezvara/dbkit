package dbkit

import (
	"context"
	"testing"
)

func TestAuditEntry_Fields(t *testing.T) {
	entry := &AuditEntry{
		Action:    AuditActionCreate,
		TableName: "users",
		RecordID:  "test-id",
	}

	if entry.Action != AuditActionCreate {
		t.Errorf("Expected action CREATE, got %s", entry.Action)
	}

	if entry.TableName != "users" {
		t.Errorf("Expected table users, got %s", entry.TableName)
	}
}

func TestAuditHook_ShouldAudit(t *testing.T) {
	// Test with empty tables (audit all)
	hook := NewAuditHook(AuditConfig{})
	if !hook.shouldAudit("users") {
		t.Error("Should audit all tables when Tables is empty")
	}

	// Test with specific tables
	hook = NewAuditHook(AuditConfig{
		Tables: []string{"users", "orders"},
	})
	if !hook.shouldAudit("users") {
		t.Error("Should audit users table")
	}
	if hook.shouldAudit("products") {
		t.Error("Should not audit products table")
	}

	// Test with exclusions
	hook = NewAuditHook(AuditConfig{
		ExcludeTables: []string{"sessions"},
	})
	if hook.shouldAudit("sessions") {
		t.Error("Should not audit excluded sessions table")
	}
	if !hook.shouldAudit("users") {
		t.Error("Should audit non-excluded users table")
	}
}

func TestAuditCreate(t *testing.T) {
	var capturedEntry *AuditEntry

	handler := func(ctx context.Context, entry *AuditEntry) error {
		capturedEntry = entry
		return nil
	}

	err := AuditCreate(context.Background(), handler, "users", "user-123", map[string]string{"name": "test"})
	if err != nil {
		t.Fatalf("AuditCreate failed: %v", err)
	}

	if capturedEntry == nil {
		t.Fatal("Handler was not called")
	}

	if capturedEntry.Action != AuditActionCreate {
		t.Errorf("Expected action CREATE, got %s", capturedEntry.Action)
	}

	if capturedEntry.TableName != "users" {
		t.Errorf("Expected table users, got %s", capturedEntry.TableName)
	}

	if capturedEntry.RecordID != "user-123" {
		t.Errorf("Expected record ID user-123, got %s", capturedEntry.RecordID)
	}
}

func TestAuditUpdate(t *testing.T) {
	var capturedEntry *AuditEntry

	handler := func(ctx context.Context, entry *AuditEntry) error {
		capturedEntry = entry
		return nil
	}

	oldData := map[string]string{"name": "old"}
	newData := map[string]string{"name": "new"}

	err := AuditUpdate(context.Background(), handler, "users", "user-123", oldData, newData)
	if err != nil {
		t.Fatalf("AuditUpdate failed: %v", err)
	}

	if capturedEntry == nil {
		t.Fatal("Handler was not called")
	}

	if capturedEntry.Action != AuditActionUpdate {
		t.Errorf("Expected action UPDATE, got %s", capturedEntry.Action)
	}
}

func TestAuditDelete(t *testing.T) {
	var capturedEntry *AuditEntry

	handler := func(ctx context.Context, entry *AuditEntry) error {
		capturedEntry = entry
		return nil
	}

	oldData := map[string]string{"name": "deleted"}

	err := AuditDelete(context.Background(), handler, "users", "user-123", oldData)
	if err != nil {
		t.Fatalf("AuditDelete failed: %v", err)
	}

	if capturedEntry == nil {
		t.Fatal("Handler was not called")
	}

	if capturedEntry.Action != AuditActionDelete {
		t.Errorf("Expected action DELETE, got %s", capturedEntry.Action)
	}
}

func TestAuditCreate_NilHandler(t *testing.T) {
	// Should not panic with nil handler
	err := AuditCreate(context.Background(), nil, "users", "user-123", nil)
	if err != nil {
		t.Errorf("Expected no error with nil handler, got %v", err)
	}
}

func TestWithAuditContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithAuditContext(ctx, "user-123", "192.168.1.1", "Mozilla/5.0")

	userID := ctx.Value(ContextKeyUserID)
	if userID != "user-123" {
		t.Errorf("Expected user ID user-123, got %v", userID)
	}

	ipAddress := ctx.Value(ContextKeyIPAddress)
	if ipAddress != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %v", ipAddress)
	}

	userAgent := ctx.Value(ContextKeyUserAgent)
	if userAgent != "Mozilla/5.0" {
		t.Errorf("Expected user agent Mozilla/5.0, got %v", userAgent)
	}
}

func TestDefaultUserIDExtractor(t *testing.T) {
	// With user ID in context
	ctx := context.WithValue(context.Background(), ContextKeyUserID, "user-123")
	userID := DefaultUserIDExtractor(ctx)
	if userID != "user-123" {
		t.Errorf("Expected user-123, got %s", userID)
	}

	// Without user ID in context
	ctx = context.Background()
	userID = DefaultUserIDExtractor(ctx)
	if userID != "" {
		t.Errorf("Expected empty string, got %s", userID)
	}
}

func TestAuditLog_Model(t *testing.T) {
	log := &AuditLog{
		Action:    AuditActionCreate,
		TableName: "users",
		RecordID:  "user-123",
	}

	if log.Action != AuditActionCreate {
		t.Errorf("Expected action CREATE, got %s", log.Action)
	}
}
