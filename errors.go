package dbkit

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrorCode represents a database error classification
type ErrorCode string

const (
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeDuplicate        ErrorCode = "DUPLICATE"
	CodeForeignKey       ErrorCode = "FOREIGN_KEY"
	CodeCheckViolation   ErrorCode = "CHECK_VIOLATION"
	CodeNotNullViolation ErrorCode = "NOT_NULL"
	CodeConnectionFailed ErrorCode = "CONNECTION_FAILED"
	CodeTimeout          ErrorCode = "TIMEOUT"
	CodeSerialization    ErrorCode = "SERIALIZATION"
	CodeDeadlock         ErrorCode = "DEADLOCK"
	CodeUnknown          ErrorCode = "UNKNOWN"
)

// Sentinel errors for quick checks
var (
	ErrNotFound         = errors.New("dbkit: record not found")
	ErrDuplicate        = errors.New("dbkit: duplicate key violation")
	ErrForeignKey       = errors.New("dbkit: foreign key violation")
	ErrCheckViolation   = errors.New("dbkit: check constraint violation")
	ErrNotNullViolation = errors.New("dbkit: not null violation")
	ErrConnection       = errors.New("dbkit: connection failed")
	ErrTimeout          = errors.New("dbkit: operation timeout")
	ErrSerialization    = errors.New("dbkit: serialization failure")
	ErrDeadlock         = errors.New("dbkit: deadlock detected")
)

// Error is a rich database error with context
type Error struct {
	Code       ErrorCode // Error classification
	Message    string    // Human-readable message
	Op         string    // Operation that failed (e.g., "FindByID", "Create")
	Table      string    // Table name if known
	Column     string    // Column name if known
	Constraint string    // Constraint name if applicable
	Detail     string    // Additional detail from PostgreSQL
	Hint       string    // Hint from PostgreSQL
	Query      string    // Query that failed (may be empty for security)
	Cause      error     // Underlying error
}

func (e *Error) Error() string {
	msg := fmt.Sprintf("dbkit: %s", e.Message)
	if e.Op != "" {
		msg = fmt.Sprintf("dbkit.%s: %s", e.Op, e.Message)
	}
	if e.Table != "" {
		msg += fmt.Sprintf(" (table: %s)", e.Table)
	}
	if e.Constraint != "" {
		msg += fmt.Sprintf(" (constraint: %s)", e.Constraint)
	}
	return msg
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// Is implements errors.Is for sentinel error matching
func (e *Error) Is(target error) bool {
	switch e.Code {
	case CodeNotFound:
		return target == ErrNotFound
	case CodeDuplicate:
		return target == ErrDuplicate
	case CodeForeignKey:
		return target == ErrForeignKey
	case CodeCheckViolation:
		return target == ErrCheckViolation
	case CodeNotNullViolation:
		return target == ErrNotNullViolation
	case CodeConnectionFailed:
		return target == ErrConnection
	case CodeTimeout:
		return target == ErrTimeout
	case CodeSerialization:
		return target == ErrSerialization
	case CodeDeadlock:
		return target == ErrDeadlock
	}
	return false
}

// wrapError converts a raw error to a rich Error
func wrapError(err error, op string) error {
	if err == nil {
		return nil
	}

	// Already wrapped
	var dbErr *Error
	if errors.As(err, &dbErr) {
		return err
	}

	// Check for "no rows" error
	if err.Error() == "sql: no rows in result set" {
		return &Error{
			Code:    CodeNotFound,
			Message: "record not found",
			Op:      op,
			Cause:   err,
		}
	}

	// PostgreSQL specific errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return wrapPgError(pgErr, op)
	}

	// Generic wrapping
	return &Error{
		Code:    CodeUnknown,
		Message: err.Error(),
		Op:      op,
		Cause:   err,
	}
}

// wrapPgError converts PostgreSQL errors to rich errors
func wrapPgError(pgErr *pgconn.PgError, op string) *Error {
	e := &Error{
		Op:         op,
		Table:      pgErr.TableName,
		Column:     pgErr.ColumnName,
		Constraint: pgErr.ConstraintName,
		Detail:     pgErr.Detail,
		Hint:       pgErr.Hint,
		Cause:      pgErr,
	}

	// Map PostgreSQL error codes
	// See: https://www.postgresql.org/docs/current/errcodes-appendix.html
	switch pgErr.Code {
	case "23505": // unique_violation
		e.Code = CodeDuplicate
		e.Message = "duplicate key value violates unique constraint"
	case "23503": // foreign_key_violation
		e.Code = CodeForeignKey
		e.Message = "foreign key constraint violation"
	case "23502": // not_null_violation
		e.Code = CodeNotNullViolation
		e.Message = "null value in column violates not-null constraint"
	case "23514": // check_violation
		e.Code = CodeCheckViolation
		e.Message = "check constraint violation"
	case "40001": // serialization_failure
		e.Code = CodeSerialization
		e.Message = "serialization failure, retry transaction"
	case "40P01": // deadlock_detected
		e.Code = CodeDeadlock
		e.Message = "deadlock detected"
	case "57014": // query_canceled (timeout)
		e.Code = CodeTimeout
		e.Message = "query was cancelled due to timeout"
	case "08000", "08003", "08006": // connection errors
		e.Code = CodeConnectionFailed
		e.Message = "database connection failed"
	default:
		e.Code = CodeUnknown
		e.Message = pgErr.Message
	}

	return e
}

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsDuplicate checks if error is a duplicate key error
func IsDuplicate(err error) bool {
	return errors.Is(err, ErrDuplicate)
}

// IsForeignKey checks if error is a foreign key error
func IsForeignKey(err error) bool {
	return errors.Is(err, ErrForeignKey)
}

// IsCheckViolation checks if error is a check constraint error
func IsCheckViolation(err error) bool {
	return errors.Is(err, ErrCheckViolation)
}

// IsNotNullViolation checks if error is a not null violation error
func IsNotNullViolation(err error) bool {
	return errors.Is(err, ErrNotNullViolation)
}

// IsConnection checks if error is a connection error
func IsConnection(err error) bool {
	return errors.Is(err, ErrConnection)
}

// IsTimeout checks if error is a timeout error
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsRetryable checks if the error is retryable (serialization, deadlock)
func IsRetryable(err error) bool {
	return errors.Is(err, ErrSerialization) || errors.Is(err, ErrDeadlock)
}

// GetErrorCode extracts the error code if it's a dbkit error
func GetErrorCode(err error) (ErrorCode, bool) {
	var dbErr *Error
	if errors.As(err, &dbErr) {
		return dbErr.Code, true
	}
	return "", false
}

// GetConstraint extracts the constraint name if available
func GetConstraint(err error) (string, bool) {
	var dbErr *Error
	if errors.As(err, &dbErr) && dbErr.Constraint != "" {
		return dbErr.Constraint, true
	}
	return "", false
}

// GetTable extracts the table name if available
func GetTable(err error) (string, bool) {
	var dbErr *Error
	if errors.As(err, &dbErr) && dbErr.Table != "" {
		return dbErr.Table, true
	}
	return "", false
}

// GetColumn extracts the column name if available
func GetColumn(err error) (string, bool) {
	var dbErr *Error
	if errors.As(err, &dbErr) && dbErr.Column != "" {
		return dbErr.Column, true
	}
	return "", false
}

// GetDetail extracts the error detail if available
func GetDetail(err error) (string, bool) {
	var dbErr *Error
	if errors.As(err, &dbErr) && dbErr.Detail != "" {
		return dbErr.Detail, true
	}
	return "", false
}

// GetHint extracts the error hint if available
func GetHint(err error) (string, bool) {
	var dbErr *Error
	if errors.As(err, &dbErr) && dbErr.Hint != "" {
		return dbErr.Hint, true
	}
	return "", false
}

// QueryResult wraps a query result with error context for chainable error handling.
// It provides a way to add meaningful context to errors without depending on Bun internals.
type QueryResult[T any] struct {
	result T
	err    error
	op     string
}

// Err returns the wrapped error with enhanced context.
// If there was no error, it returns nil.
func (qr *QueryResult[T]) Err() error {
	return wrapError(qr.err, qr.op)
}

// Unwrap returns the result and the wrapped error.
// Use this when you need both the result and the error.
func (qr *QueryResult[T]) Unwrap() (T, error) {
	return qr.result, wrapError(qr.err, qr.op)
}

// Result returns only the result value.
// Use Err() to check for errors first.
func (qr *QueryResult[T]) Result() T {
	return qr.result
}

// HasError returns true if there was an error.
func (qr *QueryResult[T]) HasError() bool {
	return qr.err != nil
}

// WithErr wraps a result and error with operation context for enhanced error handling.
// This function allows chainable error handling with meaningful context.
//
// Usage:
//
//	// For operations that return (sql.Result, error)
//	result, err := dbkit.WithErr(db.NewInsert().Model(&user).Exec(ctx), "CreateUser").Unwrap()
//
//	// For operations that return only error (like Scan)
//	err := dbkit.WithErr1(db.NewSelect().Model(&user).Where("id = ?", id).Scan(ctx), "FindByID").Err()
//
//	// Check error directly
//	if dbkit.WithErr(db.NewInsert().Model(&user).Exec(ctx), "CreateUser").HasError() {
//	    // handle error
//	}
func WithErr[T any](result T, err error, op string) *QueryResult[T] {
	return &QueryResult[T]{
		result: result,
		err:    err,
		op:     op,
	}
}

// WithErr1 is a convenience function for operations that return only an error.
// This is useful for Scan() operations which don't return a result.
//
// Usage:
//
//	err := dbkit.WithErr1(db.NewSelect().Model(&user).Where("id = ?", id).Scan(ctx), "FindByID").Err()
func WithErr1(err error, op string) *QueryResult[struct{}] {
	return &QueryResult[struct{}]{
		result: struct{}{},
		err:    err,
		op:     op,
	}
}
