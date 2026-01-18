package dbkit

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{CodeNotFound, "NOT_FOUND"},
		{CodeDuplicate, "DUPLICATE"},
		{CodeForeignKey, "FOREIGN_KEY"},
	}

	for _, tt := range tests {
		if string(tt.code) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.code)
		}
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		err      *Error
		expected string
	}{
		{
			err:      &Error{Message: "test error"},
			expected: "dbkit: test error",
		},
		{
			err:      &Error{Op: "Create", Message: "failed"},
			expected: "dbkit.Create: failed",
		},
		{
			err:      &Error{Op: "Create", Message: "failed", Table: "users"},
			expected: "dbkit.Create: failed (table: users)",
		},
		{
			err:      &Error{Op: "Create", Message: "failed", Table: "users", Constraint: "users_email_key"},
			expected: "dbkit.Create: failed (table: users) (constraint: users_email_key)",
		},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.err.Error())
		}
	}
}

func TestError_Is(t *testing.T) {
	tests := []struct {
		err    *Error
		target error
		match  bool
	}{
		{&Error{Code: CodeNotFound}, ErrNotFound, true},
		{&Error{Code: CodeDuplicate}, ErrDuplicate, true},
		{&Error{Code: CodeForeignKey}, ErrForeignKey, true},
		{&Error{Code: CodeNotFound}, ErrDuplicate, false},
		{&Error{Code: CodeUnknown}, ErrNotFound, false},
	}

	for _, tt := range tests {
		if errors.Is(tt.err, tt.target) != tt.match {
			t.Errorf("expected Is(%v, %v) = %v", tt.err.Code, tt.target, tt.match)
		}
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &Error{Code: CodeUnknown, Cause: cause}

	if err.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestWrapError_Nil(t *testing.T) {
	if wrapError(nil, "Test") != nil {
		t.Error("wrapError(nil) should return nil")
	}
}

func TestWrapError_AlreadyWrapped(t *testing.T) {
	original := &Error{Code: CodeNotFound, Message: "original"}
	wrapped := wrapError(original, "Test")

	if wrapped != original {
		t.Error("already wrapped error should be returned as-is")
	}
}

func TestWrapError_NoRows(t *testing.T) {
	err := errors.New("sql: no rows in result set")
	wrapped := wrapError(err, "FindByID")

	var dbErr *Error
	if !errors.As(wrapped, &dbErr) {
		t.Fatal("expected *Error")
	}

	if dbErr.Code != CodeNotFound {
		t.Errorf("expected CodeNotFound, got %s", dbErr.Code)
	}
	if dbErr.Op != "FindByID" {
		t.Errorf("expected FindByID, got %s", dbErr.Op)
	}
}

func TestWrapPgError(t *testing.T) {
	tests := []struct {
		pgCode   string
		expected ErrorCode
	}{
		{"23505", CodeDuplicate},
		{"23503", CodeForeignKey},
		{"23502", CodeNotNullViolation},
		{"23514", CodeCheckViolation},
		{"40001", CodeSerialization},
		{"40P01", CodeDeadlock},
		{"57014", CodeTimeout},
		{"08000", CodeConnectionFailed},
		{"99999", CodeUnknown},
	}

	for _, tt := range tests {
		pgErr := &pgconn.PgError{
			Code:           tt.pgCode,
			Message:        "test",
			TableName:      "users",
			ColumnName:     "email",
			ConstraintName: "users_email_key",
		}

		wrapped := wrapPgError(pgErr, "Create")

		if wrapped.Code != tt.expected {
			t.Errorf("pgCode %s: expected %s, got %s", tt.pgCode, tt.expected, wrapped.Code)
		}
		if wrapped.Table != "users" {
			t.Errorf("expected table users, got %s", wrapped.Table)
		}
		if wrapped.Column != "email" {
			t.Errorf("expected column email, got %s", wrapped.Column)
		}
		if wrapped.Constraint != "users_email_key" {
			t.Errorf("expected constraint users_email_key, got %s", wrapped.Constraint)
		}
	}
}

func TestIsNotFound(t *testing.T) {
	err := &Error{Code: CodeNotFound}
	if !IsNotFound(err) {
		t.Error("IsNotFound should return true")
	}

	err2 := &Error{Code: CodeDuplicate}
	if IsNotFound(err2) {
		t.Error("IsNotFound should return false for non-NotFound errors")
	}
}

func TestIsDuplicate(t *testing.T) {
	err := &Error{Code: CodeDuplicate}
	if !IsDuplicate(err) {
		t.Error("IsDuplicate should return true")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected bool
	}{
		{CodeSerialization, true},
		{CodeDeadlock, true},
		{CodeNotFound, false},
		{CodeDuplicate, false},
	}

	for _, tt := range tests {
		err := &Error{Code: tt.code}
		if IsRetryable(err) != tt.expected {
			t.Errorf("IsRetryable(%s) = %v, expected %v", tt.code, !tt.expected, tt.expected)
		}
	}
}

func TestGetErrorCode(t *testing.T) {
	err := &Error{Code: CodeDuplicate}
	code, ok := GetErrorCode(err)
	if !ok {
		t.Error("expected ok=true")
	}
	if code != CodeDuplicate {
		t.Errorf("expected CodeDuplicate, got %s", code)
	}

	_, ok = GetErrorCode(errors.New("plain error"))
	if ok {
		t.Error("expected ok=false for plain error")
	}
}

func TestGetConstraint(t *testing.T) {
	err := &Error{Code: CodeDuplicate, Constraint: "users_email_key"}
	constraint, ok := GetConstraint(err)
	if !ok {
		t.Error("expected ok=true")
	}
	if constraint != "users_email_key" {
		t.Errorf("expected users_email_key, got %s", constraint)
	}

	err2 := &Error{Code: CodeNotFound}
	_, ok = GetConstraint(err2)
	if ok {
		t.Error("expected ok=false when no constraint")
	}
}

func TestChecksumSQL(t *testing.T) {
	sql := "CREATE TABLE users (id UUID PRIMARY KEY)"
	checksum := checksumSQL(sql)

	if len(checksum) != 64 {
		t.Errorf("expected 64 char hex string, got %d chars", len(checksum))
	}

	// Same SQL should produce same checksum
	if checksumSQL(sql) != checksum {
		t.Error("same SQL should produce same checksum")
	}

	// Different SQL should produce different checksum
	if checksumSQL(sql+"x") == checksum {
		t.Error("different SQL should produce different checksum")
	}
}

func TestTruncateSQL(t *testing.T) {
	short := "SELECT * FROM users"
	if truncateSQL(short, 100) != short {
		t.Error("short SQL should not be truncated")
	}

	long := "SELECT * FROM users WHERE " + string(make([]byte, 200))
	truncated := truncateSQL(long, 50)
	if len(truncated) != 53 { // 50 + "..."
		t.Errorf("expected 53 chars, got %d", len(truncated))
	}
}

func TestJoinColumns(t *testing.T) {
	tests := []struct {
		cols     []string
		expected string
	}{
		{[]string{}, ""},
		{[]string{"id"}, "id"},
		{[]string{"id", "email"}, "id, email"},
		{[]string{"id", "email", "name"}, "id, email, name"},
	}

	for _, tt := range tests {
		result := joinColumns(tt.cols)
		if result != tt.expected {
			t.Errorf("joinColumns(%v) = %s, expected %s", tt.cols, result, tt.expected)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("postgres://localhost/test")

	if cfg.URL != "postgres://localhost/test" {
		t.Error("URL not set")
	}
	if cfg.MaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns=25, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("expected MaxIdleConns=5, got %d", cfg.MaxIdleConns)
	}
}

func TestConfig_ApplyDefaults(t *testing.T) {
	cfg := Config{URL: "postgres://localhost/test"}
	cfg.applyDefaults()

	if cfg.MaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns=25, got %d", cfg.MaxOpenConns)
	}
	if cfg.DialTimeout.Seconds() != 5 {
		t.Errorf("expected DialTimeout=5s, got %v", cfg.DialTimeout)
	}
}

func TestDefaultTxOptions(t *testing.T) {
	opts := DefaultTxOptions()

	if opts.ReadOnly {
		t.Error("default should not be read-only")
	}
}

func TestReadOnlyTxOptions(t *testing.T) {
	opts := ReadOnlyTxOptions()

	if !opts.ReadOnly {
		t.Error("should be read-only")
	}
}
