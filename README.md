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

    // Create
    user := &User{Email: "john@example.com", Name: "John"}
    if err := dbkit.Create(ctx, db, user); err != nil {
        panic(err)
    }

    // Find
    found, err := dbkit.FindByID[User](ctx, db, user.ID)
    if err != nil {
        if dbkit.IsNotFound(err) {
            println("user not found")
        }
    }

    // Update
    found.Name = "John Doe"
    dbkit.Update(ctx, db, found)

    // Transaction
    err = db.Transaction(ctx, func(tx *dbkit.Tx) error {
        return dbkit.Delete(ctx, tx, found)
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
    if err := dbkit.Create(ctx, tx, &user); err != nil {
        return err // auto rollback
    }
    if err := dbkit.Create(ctx, tx, &profile); err != nil {
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
    dbkit.Create(ctx, tx, &outer)

    // Nested - uses SAVEPOINT
    err := tx.Transaction(ctx, func(tx2 *dbkit.Tx) error {
        dbkit.Create(ctx, tx2, &inner)
        return errors.New("fail") // only rolls back inner
    })

    // outer is still committed
    return nil
})
```

## Generic Helpers

```go
// Find
user, err := dbkit.FindByID[User](ctx, db, "uuid")
user, err := dbkit.FindOne[User](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
    return q.Where("email = ?", email)
})
users, err := dbkit.FindAll[User](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
    return q.Where("active = ?", true).Limit(10)
})

// Create
err := dbkit.Create(ctx, db, &user)
err := dbkit.CreateMany(ctx, db, users)

// Update
err := dbkit.Update(ctx, db, &user)
err := dbkit.UpdateColumns(ctx, db, &user, "name", "updated_at")
rows, err := dbkit.UpdateWhere[User](ctx, db, &user, func(q *bun.UpdateQuery) *bun.UpdateQuery {
    return q.Where("active = ?", false)
})

// Delete
err := dbkit.Delete(ctx, db, &user)
err := dbkit.DeleteByID[User](ctx, db, "uuid")
rows, err := dbkit.DeleteWhere[User](ctx, db, func(q *bun.DeleteQuery) *bun.DeleteQuery {
    return q.Where("created_at < ?", cutoff)
})

// Exists / Count
exists, err := dbkit.ExistsByID[User](ctx, db, "uuid")
count, err := dbkit.Count[User](ctx, db, nil)

// Upsert
err := dbkit.Upsert(ctx, db, &user, []string{"email"}, []string{"name", "updated_at"})
```

## Error Handling

```go
err := dbkit.Create(ctx, db, &user)
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
