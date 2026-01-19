package dbkit

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/fernandezvara/dbkit/hooks"
)

// DBKit wraps bun.DB with additional functionality
type DBKit struct {
	*bun.DB
	config Config
}

// New creates a new database connection with the given configuration
func New(cfg Config) (*DBKit, error) {
	// Apply defaults for zero values
	cfg.applyDefaults()

	if cfg.URL == "" {
		return nil, &Error{
			Code:    CodeConnectionFailed,
			Message: "database URL is required",
			Op:      "New",
		}
	}

	// Create pgdriver connector with timeouts
	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(cfg.URL),
		pgdriver.WithDialTimeout(cfg.DialTimeout),
		pgdriver.WithReadTimeout(cfg.ReadTimeout),
		pgdriver.WithWriteTimeout(cfg.WriteTimeout),
	)

	// Open sql.DB
	sqlDB := sql.OpenDB(connector)

	// Configure pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Create bun.DB
	bunDB := bun.NewDB(sqlDB, pgdialect.New())

	db := &DBKit{
		DB:     bunDB,
		config: cfg,
	}

	// Add observability hooks
	if cfg.Logger != nil && (cfg.LogQueries || cfg.LogSlowQueries > 0) {
		bunDB.AddQueryHook(hooks.NewLoggerHook(cfg.Logger, cfg.LogQueries, cfg.LogSlowQueries))
	}
	if cfg.MetricsRegistry != nil {
		hook, err := hooks.NewMetricsHook(cfg.MetricsRegistry)
		if err != nil {
			return nil, fmt.Errorf("dbkit: failed to create metrics hook: %w", err)
		}
		bunDB.AddQueryHook(hook)
	}
	if cfg.Tracer != nil {
		bunDB.AddQueryHook(hooks.NewTracingHook(cfg.Tracer))
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := bunDB.PingContext(ctx); err != nil {
		return nil, &Error{
			Code:    CodeConnectionFailed,
			Message: "failed to connect to database",
			Op:      "New",
			Cause:   err,
		}
	}

	return db, nil
}

// Close closes the database connection
func (db *DBKit) Close() error {
	return db.DB.Close()
}

// Ping verifies the database connection is alive
func (db *DBKit) Ping(ctx context.Context) error {
	if err := db.PingContext(ctx); err != nil {
		return wrapError(err, "Ping")
	}
	return nil
}

// Stats returns connection pool statistics
func (db *DBKit) Stats() sql.DBStats {
	return db.DB.Stats()
}

// Bun returns the underlying bun.DB for direct access
func (db *DBKit) Bun() *bun.DB {
	return db.DB
}

// Config returns the current configuration
func (db *DBKit) Config() Config {
	return db.config
}

// IDB is the interface for both DB and Tx to enable function reuse
type IDB interface {
	bun.IDB
	NewSelect() *bun.SelectQuery
	NewInsert() *bun.InsertQuery
	NewUpdate() *bun.UpdateQuery
	NewDelete() *bun.DeleteQuery
	NewRaw(query string, args ...any) *bun.RawQuery
	NewCreateTable() *bun.CreateTableQuery
	NewDropTable() *bun.DropTableQuery
	NewCreateIndex() *bun.CreateIndexQuery
	NewDropIndex() *bun.DropIndexQuery
	NewTruncateTable() *bun.TruncateTableQuery
	NewAddColumn() *bun.AddColumnQuery
	NewDropColumn() *bun.DropColumnQuery
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Ensure DB implements IDB
var _ IDB = (*DBKit)(nil)
