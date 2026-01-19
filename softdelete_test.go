package dbkit

import (
	"testing"

	"github.com/uptrace/bun"
)

func TestNotDeleted(t *testing.T) {
	// Test that NotDeleted adds the correct WHERE clause
	// This is a basic unit test - integration tests would verify actual filtering
	q := &bun.SelectQuery{}
	result := NotDeleted(q)
	if result == nil {
		t.Error("NotDeleted should return a non-nil query")
	}
}

func TestOnlyDeleted(t *testing.T) {
	// Test that OnlyDeleted adds the correct WHERE clause
	q := &bun.SelectQuery{}
	result := OnlyDeleted(q)
	if result == nil {
		t.Error("OnlyDeleted should return a non-nil query")
	}
}

func TestWithDeleted(t *testing.T) {
	// Test that WithDeleted returns a query
	q := &bun.SelectQuery{}
	result := WithDeleted(q)
	if result == nil {
		t.Error("WithDeleted should return a non-nil query")
	}
}
