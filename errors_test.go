package dbkit

import (
	"database/sql"
	"errors"
	"testing"
)

func TestWithErr_Success(t *testing.T) {
	// Simulate a successful operation
	mockResult := sql.Result(nil)
	qr := WithErr(mockResult, nil, "CreateUser")

	if qr.HasError() {
		t.Error("Expected no error")
	}

	if qr.Err() != nil {
		t.Error("Expected Err() to return nil")
	}

	result, err := qr.Unwrap()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestWithErr_Error(t *testing.T) {
	// Simulate a failed operation
	mockResult := sql.Result(nil)
	originalErr := errors.New("some database error")
	qr := WithErr(mockResult, originalErr, "CreateUser")

	if !qr.HasError() {
		t.Error("Expected error")
	}

	err := qr.Err()
	if err == nil {
		t.Error("Expected Err() to return an error")
	}

	// Check that the error is wrapped with the operation name
	var dbErr *Error
	if !errors.As(err, &dbErr) {
		t.Error("Expected error to be wrapped as *Error")
	}

	if dbErr.Op != "CreateUser" {
		t.Errorf("Expected Op to be 'CreateUser', got %s", dbErr.Op)
	}
}

func TestWithErr_NotFound(t *testing.T) {
	// Simulate a not found error
	mockResult := sql.Result(nil)
	notFoundErr := errors.New("sql: no rows in result set")
	qr := WithErr(mockResult, notFoundErr, "FindUser")

	err := qr.Err()
	if !IsNotFound(err) {
		t.Errorf("Expected not found error, got %v", err)
	}

	var dbErr *Error
	if !errors.As(err, &dbErr) {
		t.Error("Expected error to be wrapped as *Error")
	}

	if dbErr.Code != CodeNotFound {
		t.Errorf("Expected CodeNotFound, got %s", dbErr.Code)
	}
}

func TestWithErr1_Success(t *testing.T) {
	// Simulate a successful Scan operation
	qr := WithErr1(nil, "FindByID")

	if qr.HasError() {
		t.Error("Expected no error")
	}

	if qr.Err() != nil {
		t.Error("Expected Err() to return nil")
	}
}

func TestWithErr1_Error(t *testing.T) {
	// Simulate a failed Scan operation
	originalErr := errors.New("scan failed")
	qr := WithErr1(originalErr, "FindByID")

	if !qr.HasError() {
		t.Error("Expected error")
	}

	err := qr.Err()
	if err == nil {
		t.Error("Expected Err() to return an error")
	}

	var dbErr *Error
	if !errors.As(err, &dbErr) {
		t.Error("Expected error to be wrapped as *Error")
	}

	if dbErr.Op != "FindByID" {
		t.Errorf("Expected Op to be 'FindByID', got %s", dbErr.Op)
	}
}

func TestWithErr1_NotFound(t *testing.T) {
	// Simulate a not found error from Scan
	notFoundErr := errors.New("sql: no rows in result set")
	qr := WithErr1(notFoundErr, "FindByID")

	err := qr.Err()
	if !IsNotFound(err) {
		t.Errorf("Expected not found error, got %v", err)
	}
}

func TestQueryResult_Result(t *testing.T) {
	// Test that Result() returns the value
	type mockResult struct {
		value int
	}
	result := mockResult{value: 42}
	qr := WithErr(result, nil, "Test")

	if qr.Result().value != 42 {
		t.Errorf("Expected result value 42, got %d", qr.Result().value)
	}
}

func TestQueryResult_Unwrap(t *testing.T) {
	// Test that Unwrap() returns both result and error
	type mockResult struct {
		value int
	}
	result := mockResult{value: 42}
	originalErr := errors.New("test error")
	qr := WithErr(result, originalErr, "Test")

	unwrappedResult, err := qr.Unwrap()
	if unwrappedResult.value != 42 {
		t.Errorf("Expected result value 42, got %d", unwrappedResult.value)
	}
	if err == nil {
		t.Error("Expected error from Unwrap()")
	}
}
