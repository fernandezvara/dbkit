/*
Package dbkit provides a consistent database layer for Go applications.

DBKit wraps Bun ORM with additional features:
  - Connection pooling with full configuration
  - Migration execution with checksum verification
  - Transaction support with auto commit/rollback and savepoints
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

# Database Operations

Use Bun ORM directly for all CRUD operations:

	// Find by ID
	var user User
	err := db.NewSelect().Model(&user).Where("id = ?", id).Scan(ctx)

	// Find with query
	var users []User
	err := db.NewSelect().Model(&users).Where("active = ?", true).Order("created_at DESC").Scan(ctx)

	// Create
	_, err := db.NewInsert().Model(&user).Exec(ctx)

	// Update
	_, err := db.NewUpdate().Model(&user).WherePK().Exec(ctx)

	// Delete
	_, err := db.NewDelete().Model(&user).WherePK().Exec(ctx)

# Transactions

Callback-based (auto commit/rollback):

	err := db.Transaction(ctx, func(tx *dbkit.Tx) error {
	    if _, err := tx.NewInsert().Model(&user).Exec(ctx); err != nil {
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
	    tx.NewInsert().Model(&outer).Exec(ctx)

	    err := tx.Transaction(ctx, func(tx2 *dbkit.Tx) error {
	        return errors.New("fail") // only rolls back inner
	    })

	    return nil // outer commits
	})

# Error Handling

DBKit provides rich error types for PostgreSQL errors:

	if _, err := db.NewInsert().Model(&user).Exec(ctx); err != nil {
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
