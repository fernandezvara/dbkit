package dbkit

import (
	"testing"
	"time"
)

func TestBaseModel_Fields(t *testing.T) {
	model := BaseModel{
		ID:        "test-uuid",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if model.ID != "test-uuid" {
		t.Errorf("Expected ID 'test-uuid', got %s", model.ID)
	}

	if model.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if model.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestSoftDeletableModel_IsDeleted(t *testing.T) {
	// Not deleted
	model := SoftDeletableModel{}
	if model.IsDeleted() {
		t.Error("Model should not be deleted")
	}

	// Deleted
	now := time.Now()
	model.DeletedAt = &now
	if !model.IsDeleted() {
		t.Error("Model should be deleted")
	}
}

func TestVersionedModel_Fields(t *testing.T) {
	model := VersionedModel{Version: 1}
	if model.Version != 1 {
		t.Errorf("Expected Version 1, got %d", model.Version)
	}
}

func TestFullModel_IsDeleted(t *testing.T) {
	// Not deleted
	model := FullModel{}
	if model.IsDeleted() {
		t.Error("Model should not be deleted")
	}

	// Deleted
	now := time.Now()
	model.DeletedAt = &now
	if !model.IsDeleted() {
		t.Error("Model should be deleted")
	}
}

func TestFullModel_Fields(t *testing.T) {
	now := time.Now()
	model := FullModel{
		ID:        "test-uuid",
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: nil,
		Version:   1,
	}

	if model.ID != "test-uuid" {
		t.Errorf("Expected ID 'test-uuid', got %s", model.ID)
	}

	if model.Version != 1 {
		t.Errorf("Expected Version 1, got %d", model.Version)
	}
}

func TestTimestampedModel_Fields(t *testing.T) {
	now := time.Now()
	model := TimestampedModel{
		CreatedAt: now,
		UpdatedAt: now,
	}

	if model.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if model.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}
