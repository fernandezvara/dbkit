package dbkit

import (
	"context"
	"testing"
)

func TestWithTenant(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenant(ctx, "tenant-123")

	tenantID := GetTenant(ctx)
	if tenantID != "tenant-123" {
		t.Errorf("Expected tenant-123, got %s", tenantID)
	}
}

func TestGetTenant_Empty(t *testing.T) {
	ctx := context.Background()
	tenantID := GetTenant(ctx)
	if tenantID != "" {
		t.Errorf("Expected empty string, got %s", tenantID)
	}
}

func TestRequireTenant_Success(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-123")
	tenantID, err := RequireTenant(ctx)
	if err != nil {
		t.Fatalf("RequireTenant failed: %v", err)
	}
	if tenantID != "tenant-123" {
		t.Errorf("Expected tenant-123, got %s", tenantID)
	}
}

func TestRequireTenant_Error(t *testing.T) {
	ctx := context.Background()
	_, err := RequireTenant(ctx)
	if err != ErrNoTenant {
		t.Errorf("Expected ErrNoTenant, got %v", err)
	}
}

func TestTenantScope(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-123")
	fn := TenantScope(ctx)
	if fn == nil {
		t.Error("TenantScope should return a function")
	}
}

func TestTenantScope_NoTenant(t *testing.T) {
	ctx := context.Background()
	fn := TenantScope(ctx)
	if fn == nil {
		t.Error("TenantScope should return a function even without tenant")
	}
}

func TestTenantUpdateScope(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-123")
	fn := TenantUpdateScope(ctx)
	if fn == nil {
		t.Error("TenantUpdateScope should return a function")
	}
}

func TestTenantDeleteScope(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-123")
	fn := TenantDeleteScope(ctx)
	if fn == nil {
		t.Error("TenantDeleteScope should return a function")
	}
}

func TestTenantModel_SetTenantID(t *testing.T) {
	model := &TenantModel{}
	model.SetTenantID("tenant-123")
	if model.TenantID != "tenant-123" {
		t.Errorf("Expected tenant-123, got %s", model.TenantID)
	}
}

func TestSetTenantID_NoTenant(t *testing.T) {
	ctx := context.Background()
	model := &TenantModel{}
	err := SetTenantID(ctx, model)
	if err != ErrNoTenant {
		t.Errorf("Expected ErrNoTenant, got %v", err)
	}
}

func TestSetTenantID_WithTenant(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-123")
	model := &TenantModel{}
	err := SetTenantID(ctx, model)
	if err != nil {
		t.Fatalf("SetTenantID failed: %v", err)
	}
	if model.TenantID != "tenant-123" {
		t.Errorf("Expected tenant-123, got %s", model.TenantID)
	}
}

func TestNewTenantHook(t *testing.T) {
	hook := NewTenantHook("")
	if hook.Column != "tenant_id" {
		t.Errorf("Expected default column 'tenant_id', got %s", hook.Column)
	}

	hook = NewTenantHook("org_id")
	if hook.Column != "org_id" {
		t.Errorf("Expected column 'org_id', got %s", hook.Column)
	}
}

func TestDefaultTenantConfig(t *testing.T) {
	config := DefaultTenantConfig()
	if config.Column != "tenant_id" {
		t.Errorf("Expected column 'tenant_id', got %s", config.Column)
	}
	if !config.EnforceOnSelect {
		t.Error("Expected EnforceOnSelect to be true")
	}
	if !config.EnforceOnUpdate {
		t.Error("Expected EnforceOnUpdate to be true")
	}
	if !config.EnforceOnDelete {
		t.Error("Expected EnforceOnDelete to be true")
	}
	if !config.SetOnInsert {
		t.Error("Expected SetOnInsert to be true")
	}
}

func TestNewTenantIsolation(t *testing.T) {
	config := DefaultTenantConfig()
	ti := NewTenantIsolation(nil, config)
	if ti == nil {
		t.Error("NewTenantIsolation should return a non-nil value")
	}
}

func TestTenant_Model(t *testing.T) {
	tenant := &Tenant{
		ID:        "tenant-123",
		Name:      "Test Tenant",
		Subdomain: "test",
		Active:    true,
	}

	if tenant.ID != "tenant-123" {
		t.Errorf("Expected ID tenant-123, got %s", tenant.ID)
	}
	if tenant.Name != "Test Tenant" {
		t.Errorf("Expected Name 'Test Tenant', got %s", tenant.Name)
	}
}
