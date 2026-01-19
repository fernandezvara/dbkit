# DBKit

An opinionated database layer for Go applications built on top of [Bun ORM](https://bun.uptrace.dev/) and [pgx](https://github.com/jackc/pgx).

## Features

- **Connection Pooling**: Full control over pool size, lifetimes, and timeouts
- **Migrations**: Execute SQL migrations with checksum verification
- **Transactions**: Callback-based (auto commit/rollback) + Manual + Savepoints
- **Rich Errors**: Typed errors with Code, Constraint, Table, Column, Detail, Hint
- **Observability**: Structured logging, Prometheus metrics, OpenTelemetry tracing

## Installation

```bash
go get github.com/fernandezvara/dbkit
```

## Common Database Patterns with Bun

Here are common database operations and how to implement them using Bun ORM directly:

| Pattern             | Bun Implementation                                                        |
| ------------------- | ------------------------------------------------------------------------- |
| **FindByID**        | `db.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx)`              |
| **FindByPK**        | `db.NewSelect().Model(model).WherePK().Scan(ctx)`                         |
| **FindOne**         | `db.NewSelect().Model(&model).Limit(1).Scan(ctx)`                         |
| **FindAll**         | `db.NewSelect().Model(&models).Scan(ctx)`                                 |
| **Create**          | `db.NewInsert().Model(&model).Exec(ctx)`                                  |
| **CreateReturning** | `db.NewInsert().Model(&model).Returning("id").Exec(ctx)`                  |
| **CreateMany**      | `db.NewInsert().Model(&models).Exec(ctx)`                                 |
| **Update**          | `db.NewUpdate().Model(&model).WherePK().Exec(ctx)`                        |
| **UpdateColumns**   | `db.NewUpdate().Model(&model).Column("name").WherePK().Exec(ctx)`         |
| **Delete**          | `db.NewDelete().Model(&model).WherePK().Exec(ctx)`                        |
| **DeleteByID**      | `db.NewDelete().Model(&model).Where("id = ?", id).Exec(ctx)`              |
| **Exists**          | `db.NewSelect().Model(&model).Where("name = ?", name).Exists(ctx)`        |
| **ExistsByID**      | `db.NewSelect().Model(&model).Where("id = ?", id).Exists(ctx)`            |
| **Count**           | `db.NewSelect().Model(&model).Where("active = ?", true).Count(ctx)`       |
| **CountAll**        | `db.NewSelect().Model(&model).Count(ctx)`                                 |
| **Reload**          | `db.NewSelect().Model(&model).WherePK().Scan(ctx)`                        |
| **Upsert**          | `db.NewInsert().Model(&model).On("CONFLICT (email) DO UPDATE").Exec(ctx)` |
| **Transaction**     | `db.Transaction(ctx, func(tx *bun.Tx) error { ... })`                     |
| **Raw Query**       | `db.NewRaw("SELECT * FROM users WHERE age > ?", age).Scan(ctx, &users)`   |

**Enhanced Error Handling with dbkit:**

```go
// Wrap any Bun operation with enhanced error context:
user, err := dbkit.WithErr(db.NewSelect().Model(&user).Where("id = ?", id).Scan(ctx), "FindByID").Unwrap()
if err != nil {
    // Get rich error information:
    if dbkit.IsNotFound(err) {
        // Handle not found
    }
    var dbErr *dbkit.Error
    if errors.As(err, &dbErr) {
        fmt.Printf("Table: %s, Constraint: %s\n", dbErr.Table, dbErr.Constraint)
    }
}
```

## Quick Start

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/fernandezvara/dbkit"
    "github.com/uptrace/bun"
)

type User struct {
    bun.BaseModel `bun:"table:users,alias:u"`

    ID    string `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
    Email string `bun:"email,notnull,unique"`
    Name  string `bun:"name,notnull"`
}

func main() {
    ctx := context.Background()

    // Connect
    db, err := dbkit.New(dbkit.Config{
        URL:             os.Getenv("DATABASE_URL"),
        MaxOpenConns:    25,
        MaxIdleConns:    5,
        ConnMaxLifetime: 5 * time.Minute,
        Logger:          slog.Default(),
        LogSlowQueries:  100 * time.Millisecond,
    })
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Run migrations
    _, err = db.Migrate(ctx, []dbkit.Migration{
        {ID: "001", Description: "Create users", SQL: `
            CREATE TABLE users (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                email VARCHAR(255) UNIQUE NOT NULL,
                name VARCHAR(255) NOT NULL
            );
        `},
    })
    if err != nil {
        panic(err)
    }

    // Create using direct Bun call
    user := &User{Email: "john@example.com", Name: "John"}
    if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
        panic(err)
    }

    // Find using direct Bun call
    var found User
    err = db.NewSelect().Model(&found).Where("id = ?", user.ID).Scan(ctx)
    if err != nil {
        if dbkit.IsNotFound(err) {
            println("user not found")
        }
    }

    // Update using direct Bun call
    found.Name = "John Doe"
    db.NewUpdate().Model(&found).WherePK().Exec(ctx)

    // Transaction
    err = db.Transaction(ctx, func(tx *dbkit.Tx) error {
        _, err := tx.NewDelete().Model(&found).WherePK().Exec(ctx)
        return err
    })
}
```

## Migrations

Migrations are provided as a slice and executed in order:

```go
migrations := []dbkit.Migration{
    {
        ID:          "001",
        Description: "Create users table",
        SQL: `
            CREATE TABLE users (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                email VARCHAR(255) UNIQUE NOT NULL
            );
        `,
    },
    {
        ID:          "002",
        Description: "Add name column",
        SQL:         `ALTER TABLE users ADD COLUMN name VARCHAR(255);`,
    },
}

result, err := db.Migrate(ctx, migrations)
// result.Applied: migrations that were applied
// result.Skipped: migrations already in database
```

DBKit tracks migrations in `_dbkit_migrations` table with checksums to detect changes.

### ⚠️ Important: Migration ID Collision Prevention

**Migration IDs must be unique across your entire application to prevent conflicts.**

When using migrations from multiple libraries or packages, ensure that migration IDs do not collide. Each migration ID must be globally unique within your application context.

**Examples of problematic collisions:**

- Library A has migration ID `"001"`
- Library B also has migration ID `"001"`
- This will cause conflicts and unpredictable behavior

**Recommended strategies:**

1. **Use prefixed IDs**: `libraryA_001`, `libraryB_001`
2. **Use timestamps**: `20240115_001_create_users`
3. **Use package names**: `auth_001`, `billing_001`
4. **Use UUIDs**: Generate unique IDs for each migration

```go
// Good - prefixed with library/package name
migrations := []dbkit.Migration{
    {ID: "auth_001", Description: "Create users table", SQL: "..."},
    {ID: "billing_001", Description: "Create invoices table", SQL: "..."},
}

// Bad - potential collision with other libraries
migrations := []dbkit.Migration{
    {ID: "001", Description: "Create users table", SQL: "..."},
    {ID: "002", Description: "Create invoices table", SQL: "..."},
}
```

Failure to ensure unique migration IDs can result in:

- Migrations being skipped or applied out of order
- Database schema inconsistencies
- Difficult-to-debug migration conflicts

## Transactions

### Callback-based (auto commit/rollback)

```go
err := db.Transaction(ctx, func(tx *dbkit.Tx) error {
    if _, err := tx.NewInsert().Model(&user).Exec(ctx); err != nil {
        return err // auto rollback
    }
    if _, err := tx.NewInsert().Model(&profile).Exec(ctx); err != nil {
        return err // auto rollback
    }
    return nil // auto commit
})
```

### Manual control

```go
tx, err := db.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback() // no-op if committed

// ... do work ...

return tx.Commit()
```

### Nested transactions (savepoints)

```go
err := db.Transaction(ctx, func(tx *dbkit.Tx) error {
    tx.NewInsert().Model(&outer).Exec(ctx)

    // Nested - uses SAVEPOINT
    err := tx.Transaction(ctx, func(tx2 *dbkit.Tx) error {
        tx2.NewInsert().Model(&inner).Exec(ctx)
        return errors.New("fail") // only rolls back inner
    })

    // outer is still committed
    return nil
})
```

## Chainable Error Wrapping

DBKit provides chainable error wrapping to add meaningful context to database errors:

```go
// For operations that return (sql.Result, error)
result, err := dbkit.WithErr(db.NewInsert().Model(&user).Exec(ctx), "CreateUser").Unwrap()
if err != nil {
    // err is wrapped with operation context
}

// For operations that return only error (like Scan)
err := dbkit.WithErr1(db.NewSelect().Model(&user).Where("id = ?", id).Scan(ctx), "FindByID").Err()

// Check error directly
if dbkit.WithErr(db.NewInsert().Model(&user).Exec(ctx), "CreateUser").HasError() {
    // handle error
}

// Get result without error check
qr := dbkit.WithErr(db.NewInsert().Model(&user).Exec(ctx), "CreateUser")
if !qr.HasError() {
    result := qr.Result()
}
```

## Error Handling

```go
_, err := db.NewInsert().Model(&user).Exec(ctx)
if err != nil {
    // Quick checks with sentinel errors
    if dbkit.IsNotFound(err) { ... }
    if dbkit.IsDuplicate(err) { ... }
    if dbkit.IsForeignKey(err) { ... }
    if dbkit.IsRetryable(err) { ... } // serialization or deadlock

    // Rich error details
    var dbErr *dbkit.Error
    if errors.As(err, &dbErr) {
        fmt.Println(dbErr.Code)       // DUPLICATE
        fmt.Println(dbErr.Table)      // users
        fmt.Println(dbErr.Column)     // email
        fmt.Println(dbErr.Constraint) // users_email_key
        fmt.Println(dbErr.Detail)     // from PostgreSQL
        fmt.Println(dbErr.Hint)       // from PostgreSQL
    }
}
```

## Base Models

DBKit provides composable base models for common patterns:

```go
// Basic model with ID and timestamps
type User struct {
    bun.BaseModel `bun:"table:users,alias:u"`
    dbkit.BaseModel
    Email string `bun:"email,notnull,unique"`
}

// With soft delete capability
type Post struct {
    bun.BaseModel `bun:"table:posts,alias:p"`
    dbkit.BaseModel
    dbkit.SoftDeletableModel
    Title string `bun:"title,notnull"`
}

// With optimistic locking (versioning)
type Account struct {
    bun.BaseModel `bun:"table:accounts,alias:a"`
    dbkit.BaseModel
    dbkit.VersionedModel
    Balance int64 `bun:"balance,notnull"`
}

// Full model with all features
type Document struct {
    bun.BaseModel `bun:"table:documents,alias:d"`
    dbkit.FullModel  // ID, timestamps, soft delete, version
    Content string `bun:"content"`
}
```

### Available Base Models

| Model                | Fields                   | Use Case                    |
| -------------------- | ------------------------ | --------------------------- |
| `BaseModel`          | ID, CreatedAt, UpdatedAt | Standard models             |
| `SoftDeletableModel` | DeletedAt                | Add soft delete capability  |
| `VersionedModel`     | Version                  | Add optimistic locking      |
| `TimestampedModel`   | CreatedAt, UpdatedAt     | Timestamps without UUID ID  |
| `FullModel`          | All fields combined      | Models needing all features |

### Soft Delete Operations

```go
// Soft delete a record
dbkit.SoftDelete(ctx, db, &user)
dbkit.SoftDeleteByID[User](ctx, db, userID)

// Restore a soft-deleted record
dbkit.Restore(ctx, db, &user)
dbkit.RestoreByID[User](ctx, db, userID)

// Permanently delete (bypass soft delete)
dbkit.HardDelete(ctx, db, &user)
dbkit.HardDeleteByID[User](ctx, db, userID)

// Query modifiers for soft delete
db.NewSelect().Model(&users).Apply(dbkit.NotDeleted).Scan(ctx)   // Exclude deleted
db.NewSelect().Model(&users).Apply(dbkit.OnlyDeleted).Scan(ctx)  // Only deleted
db.NewSelect().Model(&users).Apply(dbkit.WithDeleted).Scan(ctx)  // Include all
```

### Optimistic Locking

```go
// Update with version check (returns ErrConflict on mismatch)
err := dbkit.UpdateWithVersion(ctx, db, &account, account.Version)
if dbkit.IsConflict(err) {
    // Handle conflict - reload and retry
}

// Update specific columns with version check
err := dbkit.UpdateColumnsWithVersion(ctx, db, &account, account.Version, "balance")

// Retry on conflict automatically
err := dbkit.RetryOnConflict(ctx, 3, func() error {
    db.NewSelect().Model(&account).WherePK().Scan(ctx)  // Reload
    account.Balance += 100
    return dbkit.UpdateWithVersion(ctx, db, &account, account.Version)
})

// Builder pattern for versioned updates
result, err := dbkit.NewVersionedUpdate(db, &account, account.Version).
    Columns("balance", "updated_at").
    Exec(ctx)
```

### Audit Trail

```go
// Create a database audit handler
handler := dbkit.NewDatabaseAuditHandler(db)

// Log operations manually
dbkit.AuditCreate(ctx, handler, "users", user.ID, &user)
dbkit.AuditUpdate(ctx, handler, "users", user.ID, &oldUser, &newUser)
dbkit.AuditDelete(ctx, handler, "users", user.ID, &user)

// Add audit context (user ID, IP, user agent)
ctx = dbkit.WithAuditContext(ctx, userID, ipAddress, userAgent)

// Configure audit hook
hook := dbkit.NewAuditHook(dbkit.AuditConfig{
    Handler:        handler,
    Tables:         []string{"users", "orders"},  // Optional: specific tables
    ExcludeTables:  []string{"sessions"},         // Optional: exclude tables
    IncludeOldData: true,
    IncludeNewData: true,
    UserIDExtractor: dbkit.DefaultUserIDExtractor,
})
```

## Pagination

### Offset-based Pagination

```go
// Simple pagination modifier
var users []User
db.NewSelect().Model(&users).Apply(dbkit.Paginate(2, 10)).Scan(ctx)

// With count and metadata
page, err := dbkit.PaginateWithCount[User](ctx, db, 1, 10, func(q *bun.SelectQuery) *bun.SelectQuery {
    return q.Where("active = ?", true).Order("created_at DESC")
})
// page.Items, page.TotalItems, page.TotalPages, page.PageInfo
```

### Cursor-based Pagination

```go
// Forward pagination
var users []User
db.NewSelect().Model(&users).
    Apply(dbkit.CursorPaginate("id", "", afterCursor, 10, true)).
    Scan(ctx)

// Process results
items, pageInfo := dbkit.CursorPaginateResult(users, 10, true, func(u User) string {
    return dbkit.EncodeCursor(u.ID, "")
})
// pageInfo.HasNextPage, pageInfo.EndCursor
```

### Keyset Pagination

```go
// Efficient for large datasets
var users []User
db.NewSelect().Model(&users).
    Apply(dbkit.KeysetPaginate("id", lastID, 10)).
    Order("id ASC").
    Scan(ctx)
```

## Observability

### Logging

```go
db, _ := dbkit.New(dbkit.Config{
    URL:            url,
    Logger:         slog.Default(),
    LogQueries:     true,                    // Log all queries (debug)
    LogSlowQueries: 100 * time.Millisecond,  // Log slow queries (warn)
})
```

### Prometheus Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

db, _ := dbkit.New(dbkit.Config{
    URL:             url,
    MetricsRegistry: prometheus.DefaultRegisterer,
})

// Exposed metrics:
// - dbkit_query_duration_seconds (histogram)
// - dbkit_queries_total (counter)
// - dbkit_query_errors_total (counter)
```

### OpenTelemetry Tracing

```go
import "go.opentelemetry.io/otel"

db, _ := dbkit.New(dbkit.Config{
    URL:    url,
    Tracer: otel.Tracer("dbkit"),
})
```

## Health Checks

```go
// Simple check
if db.IsHealthy(ctx) {
    // OK
}

// Detailed status
status := db.Health(ctx)
fmt.Println(status.Healthy)              // true/false
fmt.Println(status.Latency)              // ping latency
fmt.Println(status.PoolStats.InUse)      // connections in use
fmt.Println(status.PoolStats.Idle)       // idle connections
```

## Direct Bun Access

For complex queries, access Bun directly:

```go
// Raw queries
var results []MyStruct
db.NewRaw("SELECT ... COMPLEX QUERY ...", args...).Scan(ctx, &results)

// Full query builder
db.NewSelect().
    Model(&users).
    Column("email").
    ColumnExpr("COUNT(*) AS count").
    Group("email").
    Having("COUNT(*) > 1").
    Scan(ctx)

// Get underlying bun.DB
bunDB := db.Bun()
```

## License

MIT
