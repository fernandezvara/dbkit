package dbkit

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/uptrace/bun"
)

// TestModel is a simple model for testing
type TestModel struct {
	bun.BaseModel `bun:"table:test_models,alias:tm"`
	ID            string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Name          string    `bun:"name,notnull"`
	Email         string    `bun:"email,notnull,unique"`
	Age           int       `bun:"age"`
	Active        bool      `bun:"active,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

// getTestDB returns a database connection for testing
func getTestDB(t *testing.T) *DBKit {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/dbkit_test?sslmode=disable"
	}

	db, err := New(Config{
		URL:             dbURL,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute,
		Logger:          slog.Default(),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Clean up test table before each test
	_, err = db.NewDropTable().IfExists().TableExpr("test_models").Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to drop test table: %v", err)
	}

	return db
}

func TestIntegration_Create(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Test Create
	model := &TestModel{
		Name:   "John Doe",
		Email:  "john@example.com",
		Age:    30,
		Active: true,
	}

	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if model.ID == "" {
		t.Error("ID should be set after creation")
	}

	// Verify the record was created
	found, err := FindByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.Name != model.Name {
		t.Errorf("Expected name %s, got %s", model.Name, found.Name)
	}

	if found.Email != model.Email {
		t.Errorf("Expected email %s, got %s", model.Email, found.Email)
	}
}

func TestIntegration_CreateMany(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Test CreateMany
	models := []*TestModel{
		{Name: "Alice", Email: "alice@example.com", Age: 25},
		{Name: "Bob", Email: "bob@example.com", Age: 30},
		{Name: "Charlie", Email: "charlie@example.com", Age: 35},
	}

	err = CreateMany(ctx, db, models)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	// Verify all records were created
	for _, model := range models {
		if model.ID == "" {
			t.Error("ID should be set after creation")
		}

		found, err := FindByID[TestModel](ctx, db, model.ID)
		if err != nil {
			t.Fatalf("FindByID failed for %s: %v", model.ID, err)
		}

		if found.Name != model.Name {
			t.Errorf("Expected name %s, got %s", model.Name, found.Name)
		}
	}
}

func TestIntegration_FindOne(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data
	model := &TestModel{Name: "Test User", Email: "test@example.com", Age: 40}
	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test FindOne
	found, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "test@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if found.Name != "Test User" {
		t.Errorf("Expected name Test User, got %s", found.Name)
	}

	// Test FindOne with no results
	_, err = FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "nonexistent@example.com")
	})
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got %v", err)
	}
}

func TestIntegration_FindAll(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data with unique prefix
	models := []*TestModel{
		{Name: "IFA_User1", Email: "ifa_user1@example.com", Age: 25, Active: true},
		{Name: "IFA_User2", Email: "ifa_user2@example.com", Age: 30, Active: false},
		{Name: "IFA_User3", Email: "ifa_user3@example.com", Age: 35, Active: true},
	}

	err = CreateMany(ctx, db, models)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	// Test FindAll with filter - only our test records
	activeUsers, err := FindAll[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("active = ?", true).Where("age > ?", 30).Order("name")
	})
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(activeUsers) != 1 {
		t.Errorf("Expected 1 active users, got %d", len(activeUsers))
	}

	// Test FindAll with filter for our records
	allUsers, err := FindAll[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("name LIKE ?", "IFA_%")
	})
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(allUsers) != 3 {
		t.Errorf("Expected 3 users, got %d", len(allUsers))
	}
}

func TestIntegration_Update(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data
	model := &TestModel{Name: "Original Name", Email: "original@example.com", Age: 25}
	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test Update
	model.Name = "Updated Name"
	model.Age = 30
	err = Update(ctx, db, model)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	found, err := FindByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.Name != "Updated Name" {
		t.Errorf("Expected name Updated Name, got %s", found.Name)
	}

	if found.Age != 30 {
		t.Errorf("Expected age 30, got %d", found.Age)
	}
}

func TestIntegration_UpdateColumns(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data
	model := &TestModel{Name: "Test", Email: "test@example.com", Age: 25}
	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test UpdateColumns
	err = UpdateColumns(ctx, db, model, "name", "age")
	if err != nil {
		t.Fatalf("UpdateColumns failed: %v", err)
	}

	// Verify only specified columns were updated
	found, err := FindByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.Name != "Test" {
		t.Errorf("Expected name Test, got %s", found.Name)
	}

	if found.Age != 25 {
		t.Errorf("Expected age 25, got %d", found.Age)
	}
}

func TestIntegration_UpdateWhere(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data
	models := []*TestModel{
		{Name: "User1", Email: "user1@example.com", Age: 25},
		{Name: "User2", Email: "user2@example.com", Age: 30},
		{Name: "User3", Email: "user3@example.com", Age: 35},
	}

	err = CreateMany(ctx, db, models)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	// Test UpdateWhere
	updateModel := &TestModel{Age: 99}
	rowsAffected, err := UpdateWhere[TestModel](ctx, db, updateModel, func(q *bun.UpdateQuery) *bun.UpdateQuery {
		return q.Where("age < ?", 30)
	})
	if err != nil {
		t.Fatalf("UpdateWhere failed: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}

	// Verify update
	updated, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "user1@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if updated.Age != 99 {
		t.Errorf("Expected age 99, got %d", updated.Age)
	}
}

func TestIntegration_Delete(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data
	model := &TestModel{Name: "To Delete", Email: "delete@example.com", Age: 25}
	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test Delete
	err = Delete(ctx, db, model)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = FindByID[TestModel](ctx, db, model.ID)
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got %v", err)
	}
}

func TestIntegration_DeleteByID(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data
	model := &TestModel{Name: "To Delete", Email: "delete2@example.com", Age: 25}
	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test DeleteByID
	err = DeleteByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("DeleteByID failed: %v", err)
	}

	// Verify deletion
	_, err = FindByID[TestModel](ctx, db, model.ID)
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got %v", err)
	}
}

func TestIntegration_DeleteWhere(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data
	models := []*TestModel{
		{Name: "User1", Email: "user1@example.com", Age: 25},
		{Name: "User2", Email: "user2@example.com", Age: 30},
		{Name: "User3", Email: "user3@example.com", Age: 35},
	}

	err = CreateMany(ctx, db, models)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	// Test DeleteWhere
	rowsAffected, err := DeleteWhere[TestModel](ctx, db, func(q *bun.DeleteQuery) *bun.DeleteQuery {
		return q.Where("age > ?", 30)
	})
	if err != nil {
		t.Fatalf("DeleteWhere failed: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}

	// Verify deletion
	_, err = FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "user3@example.com")
	})
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got %v", err)
	}

	// Verify other records still exist
	remaining, err := FindAll[TestModel](ctx, db, nil)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(remaining) != 2 {
		t.Errorf("Expected 2 remaining users, got %d", len(remaining))
	}
}

func TestIntegration_ExistsByID(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data
	model := &TestModel{Name: "Test", Email: "exists@example.com", Age: 25}
	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test ExistsByID with existing ID
	exists, err := ExistsByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("ExistsByID failed: %v", err)
	}

	if !exists {
		t.Error("Expected true for existing ID")
	}

	// Test ExistsByID with non-existing ID
	nonExistentID := "00000000-0000-0000-0000-000000000000"
	exists, err = ExistsByID[TestModel](ctx, db, nonExistentID)
	if err != nil {
		t.Fatalf("ExistsByID failed: %v", err)
	}

	if exists {
		t.Error("Expected false for non-existing ID")
	}
}

func TestIntegration_Count(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test data
	models := []*TestModel{
		{Name: "IC_User1", Email: "ic_user1@example.com", Age: 25, Active: true},
		{Name: "IC_User2", Email: "ic_user2@example.com", Age: 30, Active: false},
		{Name: "IC_User3", Email: "ic_user3@example.com", Age: 35, Active: true},
	}

	err = CreateMany(ctx, db, models)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	// Test Count all
	count, err := Count[TestModel](ctx, db, nil)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}

	// Test Count with filter
	activeCount, err := Count[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("active = ?", true)
	})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if activeCount != 2 {
		t.Errorf("Expected active count 2, got %d", activeCount)
	}
}

func TestIntegration_Upsert(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Test Upsert (insert)
	model := &TestModel{Name: "Upsert Test", Email: "upsert@example.com", Age: 25}
	err = Upsert(ctx, db, model, []string{"email"}, []string{"name", "age", "updated_at"})
	if err != nil {
		t.Fatalf("Upsert (insert) failed: %v", err)
	}

	// Verify insertion
	inserted, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "upsert@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if inserted.Name != "Upsert Test" {
		t.Errorf("Expected name Upsert Test, got %s", inserted.Name)
	}

	// Test Upsert (update)
	model.Name = "Updated Upsert"
	model.Age = 30
	err = Upsert(ctx, db, model, []string{"email"}, []string{"name", "age", "updated_at"})
	if err != nil {
		t.Fatalf("Upsert (update) failed: %v", err)
	}

	// Verify update
	updated, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "upsert@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if updated.Name != "Updated Upsert" {
		t.Errorf("Expected name Updated Upsert, got %s", updated.Name)
	}

	if updated.Age != 30 {
		t.Errorf("Expected age 30, got %d", updated.Age)
	}

	// Verify ID didn't change
	if updated.ID != inserted.ID {
		t.Errorf("ID should not change during upsert")
	}
}
