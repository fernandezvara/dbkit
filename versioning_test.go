package dbkit

import (
	"context"
	"errors"
	"testing"
)

func TestIsConflict(t *testing.T) {
	// Test with ErrConflict
	if !IsConflict(ErrConflict) {
		t.Error("IsConflict should return true for ErrConflict")
	}

	// Test with wrapped conflict error
	wrappedErr := &Error{
		Code:  CodeConflict,
		Cause: ErrConflict,
	}
	if !IsConflict(wrappedErr) {
		t.Error("IsConflict should return true for wrapped conflict error")
	}

	// Test with other error
	otherErr := errors.New("other error")
	if IsConflict(otherErr) {
		t.Error("IsConflict should return false for other errors")
	}
}

func TestRetryOnConflict_Success(t *testing.T) {
	attempts := 0
	err := RetryOnConflict(context.Background(), 3, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryOnConflict_RetrySuccess(t *testing.T) {
	attempts := 0
	err := RetryOnConflict(context.Background(), 3, func() error {
		attempts++
		if attempts < 2 {
			return ErrConflict
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryOnConflict_MaxRetries(t *testing.T) {
	attempts := 0
	err := RetryOnConflict(context.Background(), 3, func() error {
		attempts++
		return ErrConflict
	})

	if !IsConflict(err) {
		t.Errorf("Expected conflict error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryOnConflict_NonConflictError(t *testing.T) {
	otherErr := errors.New("other error")
	attempts := 0
	err := RetryOnConflict(context.Background(), 3, func() error {
		attempts++
		return otherErr
	})

	if err != otherErr {
		t.Errorf("Expected other error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry for non-conflict), got %d", attempts)
	}
}

func TestVersionedUpdate_Builder(t *testing.T) {
	// Test builder pattern
	type TestModel struct {
		ID      string `bun:"id,pk"`
		Name    string `bun:"name"`
		Version int64  `bun:"version"`
	}

	model := &TestModel{ID: "test", Name: "test", Version: 1}

	// Just test that the builder pattern works without executing
	builder := NewVersionedUpdate(nil, model, 1)
	if builder == nil {
		t.Error("NewVersionedUpdate should return a non-nil builder")
	}

	builder = builder.Columns("name")
	if builder == nil {
		t.Error("Columns should return the builder")
	}
}
