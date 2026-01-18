package dbkit

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"

	"github.com/uptrace/bun"
)

// Tx wraps bun.Tx with additional functionality
type Tx struct {
	bun.Tx
	db           *DBKit
	savepointID  int64
	savepointSeq *int64 // Shared across nested transactions
}

// Ensure Tx implements IDB
var _ IDB = (*Tx)(nil)

// TxOptions configures transaction behavior
type TxOptions struct {
	Isolation sql.IsolationLevel
	ReadOnly  bool
}

// DefaultTxOptions returns default transaction options
func DefaultTxOptions() TxOptions {
	return TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	}
}

// ReadOnlyTxOptions returns options for read-only transactions
func ReadOnlyTxOptions() TxOptions {
	return TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  true,
	}
}

// SerializableTxOptions returns options for serializable transactions
func SerializableTxOptions() TxOptions {
	return TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}
}

// TxFunc is a function executed within a transaction
type TxFunc func(tx *Tx) error

// Transaction executes fn within a transaction with automatic commit/rollback
func (db *DBKit) Transaction(ctx context.Context, fn TxFunc) error {
	return db.TransactionWithOptions(ctx, DefaultTxOptions(), fn)
}

// TransactionWithOptions executes fn within a transaction with custom options
func (db *DBKit) TransactionWithOptions(ctx context.Context, opts TxOptions, fn TxFunc) error {
	bunTx, err := db.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: opts.Isolation,
		ReadOnly:  opts.ReadOnly,
	})
	if err != nil {
		return wrapError(err, "Transaction.Begin")
	}

	seq := int64(0)
	tx := &Tx{
		Tx:           bunTx,
		db:           db,
		savepointSeq: &seq,
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("dbkit: rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return wrapError(err, "Transaction.Commit")
	}

	return nil
}

// ReadOnlyTransaction executes fn within a read-only transaction
func (db *DBKit) ReadOnlyTransaction(ctx context.Context, fn TxFunc) error {
	return db.TransactionWithOptions(ctx, ReadOnlyTxOptions(), fn)
}

// Begin starts a new transaction (manual control)
func (db *DBKit) Begin(ctx context.Context) (*Tx, error) {
	return db.BeginWithOptions(ctx, DefaultTxOptions())
}

// BeginWithOptions starts a new transaction with custom options
func (db *DBKit) BeginWithOptions(ctx context.Context, opts TxOptions) (*Tx, error) {
	bunTx, err := db.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: opts.Isolation,
		ReadOnly:  opts.ReadOnly,
	})
	if err != nil {
		return nil, wrapError(err, "Begin")
	}

	seq := int64(0)
	return &Tx{
		Tx:           bunTx,
		db:           db,
		savepointSeq: &seq,
	}, nil
}

// Commit commits the transaction
func (tx *Tx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return wrapError(err, "Commit")
	}
	return nil
}

// Rollback aborts the transaction
func (tx *Tx) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil {
		// Ignore "already committed" or "already rolled back" errors
		if err == sql.ErrTxDone {
			return nil
		}
		return wrapError(err, "Rollback")
	}
	return nil
}

// Transaction creates a savepoint for nested transaction support
func (tx *Tx) Transaction(ctx context.Context, fn TxFunc) error {
	// Generate unique savepoint name
	id := atomic.AddInt64(tx.savepointSeq, 1)
	savepoint := fmt.Sprintf("sp_%d", id)

	// Create savepoint
	if _, err := tx.ExecContext(ctx, "SAVEPOINT "+savepoint); err != nil {
		return wrapError(err, "Transaction.Savepoint")
	}

	nestedTx := &Tx{
		Tx:           tx.Tx,
		db:           tx.db,
		savepointID:  id,
		savepointSeq: tx.savepointSeq,
	}

	if err := fn(nestedTx); err != nil {
		// Rollback to savepoint
		if _, rbErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT "+savepoint); rbErr != nil {
			return fmt.Errorf("dbkit: savepoint rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	// Release savepoint (commit)
	if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT "+savepoint); err != nil {
		return wrapError(err, "Transaction.ReleaseSavepoint")
	}

	return nil
}

// Savepoint creates a named savepoint for manual control
func (tx *Tx) Savepoint(ctx context.Context, name string) error {
	_, err := tx.ExecContext(ctx, "SAVEPOINT "+name)
	return wrapError(err, "Savepoint")
}

// RollbackTo rolls back to a named savepoint
func (tx *Tx) RollbackTo(ctx context.Context, name string) error {
	_, err := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT "+name)
	return wrapError(err, "RollbackTo")
}

// ReleaseSavepoint releases a named savepoint
func (tx *Tx) ReleaseSavepoint(ctx context.Context, name string) error {
	_, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT "+name)
	return wrapError(err, "ReleaseSavepoint")
}

// DB returns the parent database
func (tx *Tx) DBKit() *DBKit {
	return tx.db
}
