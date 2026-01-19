# dbkit Enhancement Plan

## Overview

This plan outlines the strategic evolution of dbkit from a "wrapper of wrapper" to a value-added database layer that provides meaningful abstractions and common patterns for Go applications using PostgreSQL and Bun ORM.

## Feature Changes Summary

| Feature                                                 | Action     | Rationale                                                       |
| ------------------------------------------------------- | ---------- | --------------------------------------------------------------- |
| Generic CRUD Helpers (FindByID, Create, Update, Delete) | **REMOVE** | These are thin wrappers that add no value over direct Bun usage |
| Generic Query Helpers (FindAll, Count, Exists)          | **REMOVE** | Simple one-liners that don't justify the abstraction layer      |
| Chainable Error Wrapping                                | **ADD**    | Provides meaningful errors from the start, not just at the end  |
| Base Models (with soft delete, versioning)              | **ADD**    | Eliminates boilerplate and enforces consistent patterns         |
| Pagination Helpers                                      | **ADD**    | Common pattern that requires repetitive code                    |
| Soft Delete Filtering                                   | **ADD**    | Automatic filtering reduces errors and boilerplate              |
| Audit Trail Functionality                               | **ADD**    | Common requirement in enterprise applications                   |
| Enhanced Value-Added Helpers                            | **ADD**    | Helpers that provide real value beyond simple wrapping          |
| Multi-tenancy Patterns                                  | **ADD**    | Critical pattern for SaaS applications                          |

## Detailed Implementation Plan

### 1. Remove Low-Value Wrapper Functions

**User Stories:**

- As a developer, I want to use Bun's query builder directly without unnecessary abstraction layers
- As a maintainer, I want to reduce code complexity by removing functions that don't add value

**Functions to Remove:**

- `FindByID[T]` - Use `db.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx)` directly
- `FindByPK[T]` - Use `db.NewSelect().Model(model).WherePK().Scan(ctx)` directly
- `FindOne[T]` - Use `db.NewSelect().Model(&model).Limit(1).Scan(ctx)` directly
- `FindAll[T]` - Use `db.NewSelect().Model(&models).Scan(ctx)` directly
- `Create[T]` - Use `db.NewInsert().Model(&model).Exec(ctx)` directly
- `CreateReturning[T]` - Use `db.NewInsert().Model(&model).Returning("*").Exec(ctx)` directly
- `CreateMany[T]` - Use `db.NewInsert().Model(&models).Exec(ctx)` directly
- `Update[T]` - Use `db.NewUpdate().Model(&model).WherePK().Exec(ctx)` directly
- `UpdateColumns[T]` - Use `db.NewUpdate().Model(&model).Column(cols...).WherePK().Exec(ctx)` directly
- `Delete[T]` - Use `db.NewDelete().Model(&model).WherePK().Exec(ctx)` directly
- `DeleteByID[T]` - Use `db.NewDelete().Model(&model).Where("id = ?", id).Exec(ctx)` directly
- `Exists[T]` - Use `db.NewSelect().Model(&model).Exists(ctx)` directly
- `ExistsByID[T]` - Use `db.NewSelect().Model(&model).Where("id = ?", id).Exists(ctx)` directly
- `Count[T]` - Use `db.NewSelect().Model(&model).Count(ctx)` directly
- `CountAll[T]` - Use `db.NewSelect().Model(&model).Count(ctx)` directly
- `Reload[T]` - Use `db.NewSelect().Model(&model).WherePK().Scan(ctx)` directly
- `Raw[T]` - Use `db.NewRaw(query, args...).Scan(ctx, &dest)` directly
- `RawOne[T]` - Use `db.NewRaw(query, args...).Scan(ctx, &model)` directly
- `Exec` - Use `db.ExecContext(ctx, query, args...)` directly

**Documentation Update:**

- Add common patterns table to README showing Bun equivalents
- No deprecation warnings needed (library not in use)
- Clean removal of wrapper functions

### 2. Chainable Error Wrapping System

**User Stories:**

- As a developer, I want meaningful errors from the start of my query chain
- As a developer, I want a simple chainable method that doesn't depend on Bun internals
- As a developer, I want errors that include operation context, table names, and constraint information

**Implementation - Chainable Method (Selected Approach):**

```go
// Error wrapper that can be chained to any query
type QueryResult[T any] struct {
    result T
    err    error
    op     string
    db     *DBKit
}

func (qr *QueryResult[T]) Err() error {
    return wrapError(qr.err, qr.op)
}

func (qr *QueryResult[T]) Unwrap() (T, error) {
    return qr.result, qr.err
}

// Helper functions to wrap queries with error context
func WithErr[T any](result T, err error, op string) *QueryResult[T] {
    return &QueryResult[T]{
        result: result,
        err:    err,
        op:     op,
    }
}

// Usage examples:
// Insert
result, err := WithErr(db.NewInsert().Model(&user).Exec(ctx), "Create").Unwrap()
if err != nil {
    // Handle meaningful error with context
}

// Select
user, err := WithErr(db.NewSelect().Model(&userModel).Where("id = ?", id).Scan(ctx), "FindByID").Unwrap()
if err != nil {
    // Handle meaningful error
}

// Update
result, err := WithErr(db.NewUpdate().Model(&user).WherePK().Exec(ctx), "Update").Unwrap()
if err != nil {
    // Handle meaningful error
}

// Alternative chainable syntax
err := WithErr(db.NewInsert().Model(&user).Exec(ctx), "Create").Err()
```

**Benefits of this approach:**

- Simple implementation without depending on Bun internals
- Works with any Bun query operation
- Clear separation between query execution and error handling
- Easy to maintain if Bun API changes
- Provides both unwrapped results and enhanced errors

### 3. Base Models with Soft Deletes and Versioning

**User Stories:**

- As a developer, I want base models that provide common fields and functionality
- As a developer, I want soft delete functionality that automatically filters deleted records
- As a developer, I want optimistic locking to prevent concurrent modification conflicts

**Implementation:**

```go
// Base model with common fields
type BaseModel struct {
    bun.BaseModel `bun:"table:{{.TableName}},alias:{{.Alias}}"`
    ID        string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
    CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
    UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

// Soft deletable model - can be embedded in any model
type SoftDeletableModel struct {
    BaseModel
    DeletedAt *time.Time `bun:"deleted_at,soft_delete"`
}

// Versioned model with optimistic locking - can be embedded in any model
type VersionedModel struct {
    BaseModel
    Version int `bun:"version,notnull,default:0"`
}

// Composable models - you can mix and match
type AuditableSoftDeletableModel struct {
    SoftDeletableModel
    CreatedBy string `bun:"created_by"`
    UpdatedBy string `bun:"updated_by"`
}

type VersionedSoftDeletableModel struct {
    SoftDeletableModel
    Version int `bun:"version,notnull,default:0"`
}

// Usage examples - choose what you need:

// Example 1: Just base model
type Category struct {
    BaseModel
    Name string `bun:"name,notnull"`
}

// Example 2: Soft deletable
type Product struct {
    SoftDeletableModel
    Name  string `bun:"name,notnull"`
    Price float64 `bun:"price,notnull"`
}

// Example 3: Versioned (optimistic locking)
type Configuration struct {
    VersionedModel
    Key   string `bun:"key,notnull,unique"`
    Value string `bun:"value,notnull"`
}

// Example 4: Both soft delete and versioning
type Article struct {
    VersionedSoftDeletableModel
    Title   string `bun:"title,notnull"`
    Content string `bun:"content,notnull"`
}

// Example 5: Full audit + soft delete
type User struct {
    AuditableSoftDeletableModel
    Name     string `bun:"name,notnull"`
    Email    string `bun:"email,notnull,unique"`
    Password string `bun:"password,notnull"`
}

// Helper methods for base models
func (m *BaseModel) BeforeInsert(ctx context.Context, query *bun.InsertQuery) error {
    if m.CreatedAt.IsZero() {
        m.CreatedAt = time.Now()
    }
    if m.UpdatedAt.IsZero() {
        m.UpdatedAt = time.Now()
    }
    return nil
}

func (m *BaseModel) BeforeUpdate(ctx context.Context, query *bun.UpdateQuery) error {
    m.UpdatedAt = time.Now()
    return nil
}

func (m *VersionedModel) BeforeUpdate(ctx context.Context, query *bun.UpdateQuery) error {
    m.BaseModel.BeforeUpdate(ctx, query)
    query.Where("version = ?", m.Version)
    query.Set("version = version + 1")
    return nil
}
```

**Key Benefits:**

- **Composable**: Mix and match base models as needed
- **Flexible**: Use only what you need (soft delete, versioning, or both)
- **Clean**: No inheritance, just Go embedding
- **Type-safe**: Compile-time composition

### 4. Pagination Helpers

**User Stories:**

- As a developer, I want consistent pagination across all queries
- As a developer, I want pagination metadata (total count, page info) automatically calculated
- As a developer, I want cursor-based pagination for large datasets

**Implementation:**

```go
type PaginationRequest struct {
    Page     int    `json:"page" query:"page"`
    PageSize int    `json:"page_size" query:"page_size"`
    Sort     string `json:"sort" query:"sort"`
}

type PaginationResponse struct {
    Data       interface{} `json:"data"`
    Pagination Pagination  `json:"pagination"`
}

type Pagination struct {
    Page       int   `json:"page"`
    PageSize   int   `json:"page_size"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
    HasNext    bool  `json:"has_next"`
    HasPrev    bool  `json:"has_prev"`
}

type CursorPagination struct {
    Cursor string `json:"cursor"`
    Limit  int    `json:"limit"`
    HasNext bool  `json:"has_next"`
}

// Offset-based pagination
func FindWithPagination[T any](ctx context.Context, db IDB, req PaginationRequest, query func(q *bun.SelectQuery) *bun.SelectQuery) (*PaginationResponse, error) {
    var models []T

    // Count query
    countQuery := db.NewSelect().Model((*T)(nil))
    if query != nil {
        // Apply filters without pagination
        tempQuery := countQuery
        query(tempQuery)
        countQuery = tempQuery
    }
    total, err := countQuery.Count(ctx)
    if err != nil {
        return nil, wrapError(err, "FindWithPagination.Count")
    }

    // Data query
    dataQuery := db.NewSelect().Model(&models)
    if query != nil {
        dataQuery = query(dataQuery)
    }

    offset := (req.Page - 1) * req.PageSize
    dataQuery = dataQuery.Offset(offset).Limit(req.PageSize)

    if req.Sort != "" {
        dataQuery = dataQuery.Order(req.Sort)
    }

    err = dataQuery.Scan(ctx)
    if err != nil {
        return nil, wrapError(err, "FindWithPagination.Scan")
    }

    totalPages := int(total) / req.PageSize
    if int(total)%req.PageSize != 0 {
        totalPages++
    }

    return &PaginationResponse{
        Data: models,
        Pagination: Pagination{
            Page:       req.Page,
            PageSize:   req.PageSize,
            Total:      total,
            TotalPages: totalPages,
            HasNext:    req.Page < totalPages,
            HasPrev:    req.Page > 1,
        },
    }, nil
}

// Cursor-based pagination
func FindWithCursorPagination[T any](ctx context.Context, db IDB, cursor string, limit int, query func(q *bun.SelectQuery) *bun.SelectQuery) (*[]T, *CursorPagination, error) {
    var models []T

    dataQuery := db.NewSelect().Model(&models).Limit(limit + 1) // +1 to check if there's next

    if cursor != "" {
        dataQuery = dataQuery.Where("id > ?", cursor)
    }

    if query != nil {
        dataQuery = query(dataQuery)
    }

    dataQuery = dataQuery.Order("id ASC")

    err := dataQuery.Scan(ctx)
    if err != nil {
        return nil, nil, wrapError(err, "FindWithCursorPagination")
    }

    hasNext := len(models) > limit
    if hasNext {
        models = models[:limit]
    }

    var nextCursor string
    if len(models) > 0 {
        nextCursor = models[len(models)-1].(interface{ GetID() string }).GetID()
    }

    pagination := &CursorPagination{
        Cursor: nextCursor,
        Limit:  limit,
        HasNext: hasNext,
    }

    return &models, pagination, nil
}
```

### 5. Soft Delete Filtering

**User Stories:**

- As a developer, I want soft delete filtering to be automatic for soft-deletable models
- As a developer, I want to be able to include deleted records when needed
- As a developer, I want soft delete operations to be reversible

**Implementation:**

```go
// Soft delete query helper
type SoftDeleteQueryHelper struct {
    db IDB
}

func (s *SoftDeleteQueryHelper) NewSelect() *bun.SelectQuery {
    return s.db.NewSelect().Where("deleted_at IS NULL")
}

func (s *SoftDeleteQueryHelper) NewSelectWithDeleted() *bun.SelectQuery {
    return s.db.NewSelect()
}

// Soft delete operations
func SoftDelete[T SoftDeletableModel](ctx context.Context, db IDB, model *T) error {
    now := time.Now()
    result, err := db.NewUpdate().
        Model(model).
        Set("deleted_at = ?", now).
        Set("updated_at = ?", now).
        WherePK().
        Exec(ctx)

    if err != nil {
        return wrapError(err, "SoftDelete")
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return &Error{
            Code:    CodeNotFound,
            Message: "record not found for soft delete",
            Op:      "SoftDelete",
        }
    }

    model.DeletedAt = &now
    return nil
}

func Restore[T SoftDeletableModel](ctx context.Context, db IDB, model *T) error {
    now := time.Now()
    result, err := db.NewUpdate().
        Model(model).
        Set("deleted_at = NULL").
        Set("updated_at = ?", now).
        Where("id = ?", model.GetID()).
        Where("deleted_at IS NOT NULL").
        Exec(ctx)

    if err != nil {
        return wrapError(err, "Restore")
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return &Error{
            Code:    CodeNotFound,
            Message: "record not found for restore",
            Op:      "Restore",
        }
    }

    model.DeletedAt = nil
    return nil
}

// Automatic filtering through query hooks
func (m *SoftDeletableModel) BeforeSelect(ctx context.Context, query *bun.SelectQuery) error {
    // Only add soft delete filter if not explicitly including deleted
    if !hasIncludeDeletedFlag(ctx) {
        query.Where("deleted_at IS NULL")
    }
    return nil
}

// Context helper for including deleted records
func WithDeleted(ctx context.Context) context.Context {
    return context.WithValue(ctx, "include_deleted", true)
}

func hasIncludeDeletedFlag(ctx context.Context) bool {
    flag, ok := ctx.Value("include_deleted").(bool)
    return ok && flag
}
```

### 7. Enhanced Value-Added Helpers

**User Stories:**

- As a developer, I want helpers that solve complex problems, not just wrap simple operations
- As a developer, I want batch operations that are efficient and safe
- As a developer, I want search functionality that handles common patterns

**Implementation:**

```go
// Batch operations with transaction safety
func BatchCreate[T any](ctx context.Context, db IDB, models []T, batchSize int) error {
    if len(models) == 0 {
        return nil
    }

    return db.Transaction(ctx, func(tx *Tx) error {
        for i := 0; i < len(models); i += batchSize {
            end := i + batchSize
            if end > len(models) {
                end = len(models)
            }

            batch := models[i:end]
            _, err := tx.NewInsert().Model(&batch).Exec(ctx)
            if err != nil {
                return wrapError(err, "BatchCreate")
            }
        }
        return nil
    })
}

// Upsert with conflict resolution strategies
type UpsertStrategy int

const (
    UpsertIgnore UpsertStrategy = iota
    UpsertUpdate
    UpsertMerge
)

func Upsert[T any](ctx context.Context, db IDB, model *T, conflictColumns []string, strategy UpsertStrategy, updateColumns ...string) error {
    query := db.NewInsert().Model(model)

    switch strategy {
    case UpsertIgnore:
        query = query.On("CONFLICT DO NOTHING")
    case UpsertUpdate:
        query = query.On("CONFLICT (" + joinColumns(conflictColumns) + ") DO UPDATE")
        for _, col := range updateColumns {
            query = query.Set(col + " = EXCLUDED." + col)
        }
    case UpsertMerge:
        query = query.On("CONFLICT (" + joinColumns(conflictColumns) + ") DO UPDATE")
        query = query.Set("updated_at = EXCLUDED.updated_at")
        for _, col := range updateColumns {
            query = query.Set(col + " = COALESCE(EXCLUDED." + col + ", " + col + ")")
        }
    }

    _, err := query.Exec(ctx)
    return wrapError(err, "Upsert")
}

// Search helper with full-text search support
type SearchQuery struct {
    Term       string                 `json:"term"`
    Fields     []string               `json:"fields"`
    Filters    map[string]interface{} `json:"filters"`
    Sort       string                 `json:"sort"`
    Pagination PaginationRequest      `json:"pagination"`
}

type SearchResult[T any] struct {
    Data       []T         `json:"data"`
    Pagination Pagination  `json:"pagination"`
    SearchMeta SearchMeta  `json:"search_meta"`
}

type SearchMeta struct {
    Term        string `json:"term"`
    Total       int64  `json:"total"`
    SearchTime  string `json:"search_time"`
    Suggestions []string `json:"suggestions,omitempty"`
}

func Search[T any](ctx context.Context, db IDB, req SearchQuery) (*SearchResult[T], error) {
    var models []T

    start := time.Now()

    // Build search query
    query := db.NewSelect().Model(&models)

    // Add text search
    if req.Term != "" {
        if len(req.Fields) > 0 {
            // Search in specific fields
            var searchConditions []string
            for _, field := range req.Fields {
                searchConditions = append(searchConditions, field+" ILIKE ?")
            }
            searchTerm := "%" + req.Term + "%"
            args := make([]interface{}, len(req.Fields))
            for i := range args {
                args[i] = searchTerm
            }
            query = query.Where(strings.Join(searchConditions, " OR "), args...)
        } else {
            // Full-text search (requires PostgreSQL tsvector)
            query = query.Where("to_tsvector('english', *) @@ to_tsquery('english', ?)", req.Term)
        }
    }

    // Add filters
    for field, value := range req.Filters {
        query = query.Where(field+" = ?", value)
    }

    // Count total
    total, err := query.Count(ctx)
    if err != nil {
        return nil, wrapError(err, "Search.Count")
    }

    // Add pagination and sorting
    offset := (req.Pagination.Page - 1) * req.Pagination.PageSize
    query = query.Offset(offset).Limit(req.Pagination.PageSize)

    if req.Sort != "" {
        query = query.Order(req.Sort)
    }

    err = query.Scan(ctx)
    if err != nil {
        return nil, wrapError(err, "Search.Scan")
    }

    totalPages := int(total) / req.Pagination.PageSize
    if int(total)%req.Pagination.PageSize != 0 {
        totalPages++
    }

    return &SearchResult[T]{
        Data: models,
        Pagination: Pagination{
            Page:       req.Pagination.Page,
            PageSize:   req.Pagination.PageSize,
            Total:      total,
            TotalPages: totalPages,
            HasNext:    req.Pagination.Page < totalPages,
            HasPrev:    req.Pagination.Page > 1,
        },
        SearchMeta: SearchMeta{
            Term:       req.Term,
            Total:      total,
            SearchTime: time.Since(start).String(),
        },
    }, nil
}
```

### 8. Multi-tenancy Patterns

**User Stories:**

- As a SaaS developer, I want automatic tenant isolation in all queries
- As a developer, I want to be able to query across tenants when needed
- As a developer, I want tenant-aware migrations and audit logs

**What Multi-tenancy Means:**

Multi-tenancy is an architecture where a single instance of software serves multiple customers (tenants) while keeping their data isolated and secure. There are three main approaches:

1. **Database-per-tenant**: Each tenant gets their own database
2. **Schema-per-tenant**: Each tenant gets their own schema within a database
3. **Shared-database with tenant_id**: All tenants share tables but rows are filtered by tenant_id

**Implementation for Shared-Database Approach:**

````go
// Tenant context
type TenantContext struct {
    ID     string
    Name   string
    Domain string
    Plan   string // free, pro, enterprise
}

// Tenant model
type Tenant struct {
    bun.BaseModel
    ID          string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
    Name        string    `bun:"name,notnull"`
    Domain      string    `bun:"domain,notnull,unique"`
    Plan        string    `bun:"plan,notnull,default:'free'"`
    Settings    JSON      `bun:"settings"`
    CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp"`
    UpdatedAt   time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}

// Tenant-aware base model
type TenantModel struct {
    BaseModel
    TenantID string `bun:"tenant_id,notnull"`
}

// Tenant middleware
type TenantMiddleware struct {
    db *DBKit
}

func NewTenantMiddleware(db *DBKit) *TenantMiddleware {
    return &TenantMiddleware{db: db}
}

func (tm *TenantMiddleware) WithTenant(ctx context.Context, tenantID string) context.Context {
    return context.WithValue(ctx, "tenant_id", tenantID)
}

func (tm *TenantMiddleware) GetTenant(ctx context.Context) (string, error) {
    tenantID, ok := ctx.Value("tenant_id").(string)
    if !ok {
        return "", &Error{
            Code:    CodeConnectionFailed,
            Message: "tenant context not found",
            Op:      "GetTenant",
        }
    }
    return tenantID, nil
}

// Tenant-aware query helper
type TenantQueryHelper struct {
    db    *DBKit
    tm    *TenantMiddleware
    ctx   context.Context
}

func (tq *TenantQueryHelper) NewSelect() *bun.SelectQuery {
    tenantID, err := tq.tm.GetTenant(tq.ctx)
    if err != nil {
        // This will be caught by the error wrapping
        return tq.db.NewSelect().Where("1=0") // Invalid query
    }
    return tq.db.NewSelect().Where("tenant_id = ?", tenantID)
}

func (tq *TenantQueryHelper) NewSelectAllTenants() *bun.SelectQuery {
    return tq.db.NewSelect() // Bypass tenant filtering
}

// Tenant-aware operations
func FindTenant[T TenantModel](ctx context.Context, db *DBKit, id string) (*T, error) {
    tm := NewTenantMiddleware(db)
    tq := &TenantQueryHelper{db: db, tm: tm, ctx: ctx}

    model := new(T)
    err := tq.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
    if err != nil {
        return nil, wrapError(err, "FindTenant")
    }

    return model, nil
}

func CreateTenant[T TenantModel](ctx context.Context, db *DBKit, model *T) error {
    tm := NewTenantMiddleware(db)
    tenantID, err := tm.GetTenant(ctx)
    if err != nil {
        return err
    }

    model.TenantID = tenantID
    model.CreatedAt = time.Now()
    model.UpdatedAt = time.Now()

    _, err = db.NewInsert().Model(model).Exec(ctx)
    return wrapError(err, "CreateTenant")
}

// Tenant migration support
func (db *DBKit) MigrateWithTenants(ctx context.Context, migrations []Migration, tenantIDs []string) error {
    for _, tenantID := range tenantIDs {
        tenantCtx := NewTenantMiddleware(db).WithTenant(ctx, tenantID)

        err := db.Migrate(tenantCtx, migrations)
        if err != nil {
            return fmt.Errorf("migration failed for tenant %s: %w", tenantID, err)
        }
    }
    return nil
}

// Tenant-aware audit logs
func (h *AuditHook) logTenantEvent(event *bun.QueryEvent) {
    tenantID, _ := h.context.Value("tenant_id").(string)
    // Include tenant_id in all audit logs
}

### Multi-Tenancy Usage Guide

**1. Setting Up Multi-Tenancy**

```go
// Initialize dbkit with tenant support
db, err := dbkit.New(cfg)
if err != nil {
    return err
}

// Create tenant middleware
tenantMiddleware := dbkit.NewTenantMiddleware(db)
````

**2. Creating Tenant-Aware Models**

```go
// Tenant-aware model
type TenantUser struct {
    dbkit.TenantModel  // Embed tenant support
    Name  string `bun:"name,notnull"`
    Email string `bun:"email,notnull"`
}

// Non-tenant model (shared across all tenants)
type SystemConfig struct {
    dbkit.BaseModel
    Key   string `bun:"key,notnull,unique"`
    Value string `bun:"value,notnull"`
}
```

**3. Working with Tenant Context**

```go
// Middleware to extract tenant from request
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract tenant from JWT, subdomain, or header
        tenantID := extractTenantFromRequest(r)

        // Add tenant to context
        ctx := tenantMiddleware.WithTenant(r.Context(), tenantID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// In your handlers
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("id")

    // Automatically filtered by tenant
    user, err := dbkit.FindTenant[TenantUser](r.Context(), db, userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(user)
}
```

**4. Advanced Tenant Operations**

```go
// Query across all tenants (admin operations)
func GetAllUsersAcrossTenants(ctx context.Context, db *dbkit.DBKit) ([]TenantUser, error) {
    tm := dbkit.NewTenantMiddleware(db)
    tq := &dbkit.TenantQueryHelper{db: db, tm: tm, ctx: ctx}

    var users []TenantUser
    err := tq.NewSelectAllTenants().Model(&users).Scan(ctx)
    return users, err
}

// Tenant-specific migrations
func RunMigrationsForTenant(ctx context.Context, db *dbkit.DBKit, tenantID string) error {
    migrations := []dbkit.Migration{
        {ID: "001", Description: "Create tenant users", SQL: "..."},
    }

    tenantCtx := dbkit.NewTenantMiddleware(db).WithTenant(ctx, tenantID)
    return db.Migrate(tenantCtx, migrations)
}

// Tenant isolation in transactions
func TransferUserData(ctx context.Context, db *dbkit.DBKit, fromTenant, toTenant, userID string) error {
    return db.Transaction(ctx, func(tx *dbkit.Tx) error {
        // Source tenant context
        fromCtx := dbkit.NewTenantMiddleware(db).WithTenant(ctx, fromTenant)

        // Get user from source tenant
        var user TenantUser
        err := tx.NewSelect().Model(&user).Where("id = ?", userID).Scan(fromCtx)
        if err != nil {
            return err
        }

        // Target tenant context
        toCtx := dbkit.NewTenantMiddleware(db).WithTenant(ctx, toTenant)

        // Create in target tenant
        user.ID = "" // Reset ID for new record
        _, err = tx.NewInsert().Model(&user).Exec(toCtx)
        if err != nil {
            return err
        }

        // Delete from source tenant
        _, err = tx.NewDelete().Model(&user).Where("id = ?", userID).Exec(fromCtx)
        return err
    })
}
```

**5. Tenant Management**

```go
// Create new tenant
func CreateTenant(ctx context.Context, db *dbkit.DBKit, tenant *Tenant) error {
    // Create tenant record
    _, err := db.NewInsert().Model(tenant).Exec(ctx)
    if err != nil {
        return err
    }

    // Run migrations for new tenant
    return RunMigrationsForTenant(ctx, db, tenant.ID)
}

// Tenant health check
func CheckTenantHealth(ctx context.Context, db *dbkit.DBKit, tenantID string) error {
    tenantCtx := dbkit.NewTenantMiddleware(db).WithTenant(ctx, tenantID)

    // Test tenant-specific connection
    return db.Ping(tenantCtx)
}

// Tenant statistics
func GetTenantStats(ctx context.Context, db *dbkit.DBKit, tenantID string) (*TenantStats, error) {
    tenantCtx := dbkit.NewTenantMiddleware(db).WithTenant(ctx, tenantID)

    stats := &TenantStats{}

    // Count users in tenant
    userCount, err := db.NewSelect().Model((*TenantUser)(nil)).Where("tenant_id = ?", tenantID).Count(tenantCtx)
    if err != nil {
        return nil, err
    }
    stats.UserCount = userCount

    // Get storage usage
    // ... additional stats

    return stats, nil
}
```

**6. Best Practices**

- **Always validate tenant context** before operations
- **Use tenant-aware models** for tenant-specific data
- **Use regular models** for shared/system data
- **Implement proper tenant isolation** at the application level
- **Add tenant_id to all foreign key relationships**
- **Consider row-level security** for additional protection
- **Monitor tenant performance** and resource usage

**7. Security Considerations**

```go
// Tenant validation middleware
func ValidateTenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := extractTenantFromRequest(r)

        // Validate tenant exists and is active
        if !isTenantActive(tenantID) {
            http.Error(w, "Tenant not found", http.StatusNotFound)
            return
        }

        // Check user has access to this tenant
        if !userHasAccessToTenant(getUserID(r), tenantID) {
            http.Error(w, "Access denied", http.StatusForbidden)
            return
        }

        ctx := tenantMiddleware.WithTenant(r.Context(), tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Technical Specifications

### **Go and PostgreSQL Requirements**

- **Go Version**: 1.25.5 or higher
- **PostgreSQL Versions**: 13, 14, 15, 16, 17, 18 (as defined in @[scripts/versions.mk])
- **Bun ORM**: v1.2.16 (current dependency)
- **pgx Driver**: v5.8.0 (current dependency)

### **API Design - Interface Definitions**

```go
// Core interfaces for dbkit v1.0.0

// IDB represents a database connection (main DB or transaction)
type IDB interface {
    // Core Bun operations
    NewSelect() *bun.SelectQuery
    NewInsert() *bun.InsertQuery
    NewUpdate() *bun.UpdateQuery
    NewDelete() *bun.DeleteQuery
    NewRaw(query string, args ...interface{}) *bun.RawQuery

    // Transaction operations
    Transaction(ctx context.Context, fn func(tx *Tx) error) error
    BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error)

    // Utility operations
    Ping(ctx context.Context) error
    Close() error
}

// Error wrapping interface
type ErrorWrapper interface {
    Err() error
    Unwrap() (interface{}, error)
}

// Soft deletable interface
type SoftDeletable interface {
    GetDeletedAt() *time.Time
    SetDeletedAt(*time.Time)
}

// Versionable interface for optimistic locking
type Versionable interface {
    GetVersion() int
    SetVersion(int)
}

// Auditable interface for audit trail
type Auditable interface {
    GetAuditConfig() AuditConfig
    GetAuditTableName() string
    GetAuditRecordID() string
}

// Tenant-aware interface
type TenantModel interface {
    GetTenantID() string
    SetTenantID(string)
}

// Pagination interfaces
type PaginatedResult[T any] interface {
    GetItems() []T
    GetPagination() Pagination
}

type CursorPaginatedResult[T any] interface {
    GetItems() []T
    GetCursorPagination() CursorPagination
}
```

### **No Backward Compatibility**

- This is **version 1.0.0** of the library
- Breaking changes are acceptable
- Clean API design without legacy constraints
- Target initial release: **v0.1.0**

### **Development Environment**

- **Local Development**: Already configured in @[Makefile]
- **CI/CD Pipeline**: GitHub Actions (to be added)
- **Code Formatting**: Standard Go standards (gofmt, golangci-lint)
- **Testing**: PostgreSQL versions 13-18 via Docker Compose

### **Performance Benchmarks**

Focus on benchmarks that matter for dbkit's value-added features:

```go
// Benchmark categories to implement:

// 1. Error Wrapping Performance
func BenchmarkWithErrorWrapping(b *testing.B)
func BenchmarkWithoutErrorWrapping(b *testing.B)

// 2. Soft Delete Filtering Performance
func BenchmarkSoftDeleteFiltering(b *testing.B)
func BenchmarkManualSoftDeleteFiltering(b *testing.B)

// 3. Pagination Performance
func BenchmarkOffsetPagination(b *testing.B)
func BenchmarkCursorPagination(b *testing.B)

// 4. Batch Operations Performance
func BenchmarkBatchInsert(b *testing.B)
func BenchmarkIndividualInserts(b *testing.B)

// 5. Multi-tenancy Overhead
func BenchmarkTenantQuery(b *testing.B)
func BenchmarkNonTenantQuery(b *testing.B)

// 6. Audit Trail Performance
func BenchmarkWithAudit(b *testing.B)
func BenchmarkWithoutAudit(b *testing.B)
```

### **Security Considerations**

- **Bun ORM Protection**: Bun handles SQL injection prevention through parameterized queries
- **Input Validation**: Validate all user inputs in base models and helpers
- **Tenant Isolation**: Ensure tenant_id filtering prevents cross-tenant data access
- **Audit Trail**: Log all data modifications for security auditing

### **Release Strategy**

- **Versioning**: Semantic versioning (v0.1.0 initial target)
- **Release Timeline**:
  - Phase 1: v0.1.0-alpha (cleanup and error wrapping)
  - Phase 2: v0.1.0-beta (base models and core features)
  - Phase 3: v0.1.0-rc (advanced features)
  - Phase 4: v0.1.0 (stable release)
- **Communication**: GitHub releases and README updates

## Implementation Rules

For each user story, the following tasks MUST be completed:

### **Implementation Rules:**

1. **Implement the user story** - Write the actual code functionality
2. **Document all functions** - Add comprehensive documentation for all implemented functions to ensure correct library usage
3. **Update README.md** - Add the required documentation to @[README.md] with examples and usage patterns
4. **Add tests** - Write comprehensive tests for all implemented files
5. **Make a commit** - Create a git commit with format `feature: <what it does>` and detailed description. **Do not include @plan.md in commits**, only new and updated files

### **Quality Standards:**

- All public functions must have Go doc comments
- All functions must have unit tests with >80% coverage
- README examples must be tested and working
- Commits must follow conventional commit format
- No TODO comments or placeholder code

## Implementation Phases

### Phase 1: Cleanup and Foundation (Week 1-2)

**User Story 1.1: Remove Low-Value Wrapper Functions**

- **Implement**: Remove all thin wrapper functions from helpers.go
- **Document**: Update README.md with common patterns table
- **Test**: Ensure no tests reference removed functions
- **Commit**: `feature: remove thin wrapper functions`

**User Story 1.2: Chainable Error Wrapping System**

- **Implement**: Add WithErr() function and QueryResult struct
- **Document**: Document error wrapping patterns in README.md
- **Test**: Comprehensive tests for error wrapping
- **Commit**: `feature: implement chainable error wrapping`

**User Story 1.3: Foundation Tests and Documentation**

- **Implement**: Test infrastructure and documentation templates
- **Document**: Testing guidelines and documentation standards
- **Test**: Test the testing framework itself
- **Commit**: `feature: add testing infrastructure`

### Phase 2: Base Models and Core Features (Week 3-4)

**User Story 2.1: Base Models Implementation**

- **Implement**: BaseModel, SoftDeletableModel, VersionedModel
- **Document**: Base model composition guide in README.md
- **Test**: Tests for all base model combinations
- **Commit**: `feature: implement composable base models`

**User Story 2.2: Soft Delete Functionality**

- **Implement**: Automatic filtering, restore operations
- **Document**: Soft delete usage patterns in README.md
- **Test**: Soft delete filtering and restoration tests
- **Commit**: `feature: add soft delete functionality`

**User Story 2.3: Optimistic Locking**

- **Implement**: Version field handling and conflict detection
- **Document**: Optimistic locking patterns in README.md
- **Test**: Concurrent modification tests
- **Commit**: `feature: implement optimistic locking`

**User Story 2.4: Audit Trail System**

- **Implement**: Configurable audit logging with hooks
- **Document**: Audit configuration and usage in README.md
- **Test**: Audit logging for all operations
- **Commit**: `feature: add configurable audit trail`

### Phase 3: Advanced Features (Week 5-6)

**User Story 3.1: Pagination Helpers**

- **Implement**: Offset-based and cursor-based pagination
- **Document**: Pagination patterns and examples in README.md
- **Test**: Pagination edge cases and performance
- **Commit**: `feature: implement pagination helpers`

**User Story 3.2: Enhanced Value-Added Helpers**

- **Implement**: Batch operations, search functionality, upsert strategies
- **Document**: Advanced helper patterns in README.md
- **Test**: Batch operations and search functionality
- **Commit**: `feature: add enhanced database helpers`

**User Story 3.3: Multi-tenancy Patterns**

- **Implement**: Tenant context, isolation, and management
- **Document**: Complete multi-tenancy guide in README.md
- **Test**: Tenant isolation and security tests
- **Commit**: `feature: implement multi-tenancy support`

### Phase 4: Polish and Documentation (Week 7-8)

**User Story 4.1: Performance Optimizations**

- **Implement**: Query optimization and caching strategies
- **Document**: Performance best practices in README.md
- **Test**: Benchmark tests and performance validation
- **Commit**: `feature: add performance optimizations`

**User Story 4.2: Examples and Documentation**

- **Implement**: Complete example applications
- **Document**: Full documentation with real-world examples
- **Test**: Example applications must be functional
- **Commit**: `feature: add examples and complete documentation`

**User Story 4.3: Release Preparation**

- **Implement**: Version tagging and release automation
- **Document**: Release notes and upgrade guides
- **Test**: End-to-end integration tests
- **Commit**: `feature: prepare for release`

## Commit Guidelines

### **Commit Message Format:**

```

feature: <brief description>

Detailed description of what was implemented:

- Added function X that does Y
- Updated documentation for Z
- Added tests covering edge cases A, B, C
- Updated README.md with usage examples

Files changed:

- helpers.go (removed functions)
- errors.go (added WithErr)
- README.md (updated patterns)
- \*\_test.go (comprehensive tests)

```

### **Files to Exclude from Commits:**

- **@plan.md** - Never commit the plan file
- Temporary files
- Test coverage files (coverage.out, coverage.html)
- IDE configuration files

### **Documentation Requirements:**

- All public functions must have Go doc comments
- README.md must have working examples
- Complex features need dedicated documentation sections
- Performance characteristics must be documented

## Success Metrics

1. **Code Reduction**: Remove ~50% of wrapper functions while maintaining functionality
2. **Developer Experience**: Reduce boilerplate code by 60% for common patterns
3. **Error Handling**: 100% of database operations return meaningful errors
4. **Performance**: No performance regression in core operations
5. **Adoption**: Clear documentation and examples for new users

## Risks and Mitigations

| Risk                                    | Impact | Mitigation                                                   |
| --------------------------------------- | ------ | ------------------------------------------------------------ |
| Performance overhead from new features  | Medium | Benchmark and optimize critical paths                        |
| Complexity increase                     | Medium | Keep simple operations simple, make advanced features opt-in |
| Multi-tenancy implementation complexity | High   | Start with simple patterns, evolve based on feedback         |

## Conclusion

This enhancement plan transforms dbkit from a thin wrapper to a value-added database layer that solves real problems for Go developers. The focus is on providing meaningful abstractions for common patterns while maintaining the flexibility and power of the underlying Bun ORM.

The key success factors are:

1. Removing unnecessary abstractions
2. Adding value where it matters most (error handling, common patterns)
3. Providing escape hatches for advanced use cases
4. Maintaining backward compatibility where possible

This approach positions dbkit as a productivity tool that enhances developer experience without sacrificing flexibility or performance.

```

```

```

```

```

```
