/*
Package dbkit provides a consistent database layer for Go applications.

DBKit wraps Bun ORM with additional features:
  - Connection pooling with full configuration
  - Migration execution with checksum verification
  - Transaction support with auto commit/rollback and savepoints
  - Generic CRUD helpers using Go generics
  - Rich error handling with PostgreSQL error parsing
  - Configurable observability (logging, metrics, tracing)
  - Health check utilities

# Basic Usage

	cfg := dbkit.DefaultConfig(os.Getenv("DATABASE_URL"))
	cfg.Logger = slog.Default()
	cfg.LogSlowQueries = 100 * time.Millisecond

	db, err := dbkit.New(cfg)
	if err != nil {
	    log.Fatal(err)
	}
	defer db.Close()

# Migrations

	migrations := []dbkit.Migration{
	    {ID: "001", Description: "Create users", SQL: "CREATE TABLE users (...)"},
	    {ID: "002", Description: "Add index", SQL: "CREATE INDEX ..."},
	}

	result, err := db.Migrate(ctx, migrations)

# Generic CRUD

	// Find by ID
	user, err := dbkit.FindByID[User](ctx, db, "uuid")

	// Find with query
	users, err := dbkit.FindAll[User](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
	    return q.Where("active = ?", true).OrderExpr("created_at DESC")
	})

	// Create
	err := dbkit.Create(ctx, db, &user)

	// Update
	err := dbkit.Update(ctx, db, &user)

	// Delete
	err := dbkit.Delete(ctx, db, &user)

# Transactions

Callback-based (auto commit/rollback):

	err := db.Transaction(ctx, func(tx *dbkit.Tx) error {
	    if err := dbkit.Create(ctx, tx, &user); err != nil {
	        return err // rollback
	    }
	    return nil // commit
	})

Manual control:

	tx, err := db.Begin(ctx)
	if err != nil {
	    return err
	}
	defer tx.Rollback()

	// ... operations ...

	return tx.Commit()

Nested transactions (savepoints):

	err := db.Transaction(ctx, func(tx *dbkit.Tx) error {
	    dbkit.Create(ctx, tx, &outer)

	    err := tx.Transaction(ctx, func(tx2 *dbkit.Tx) error {
	        return errors.New("fail") // only rolls back inner
	    })

	    return nil // outer commits
	})

# Error Handling

DBKit provides rich error types:

	if err := dbkit.Create(ctx, db, &user); err != nil {
	    if dbkit.IsDuplicate(err) {
	        // Handle duplicate key
	    }

	    var dbErr *dbkit.Error
	    if errors.As(err, &dbErr) {
	        fmt.Println(dbErr.Code)       // DUPLICATE
	        fmt.Println(dbErr.Constraint) // users_email_key
	        fmt.Println(dbErr.Detail)     // Key (email)=(test@example.com) already exists
	    }
	}
*/
package dbkit
