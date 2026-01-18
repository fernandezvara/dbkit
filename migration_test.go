package dbkit

import (
	"context"
	"testing"
)

func TestMigration_Basic(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Clean up tables from previous runs
	_, _ = db.NewDropTable().IfExists().TableExpr("users").Exec(ctx)
	_, err := db.NewDropTable().IfExists().TableExpr("_dbkit_migrations").Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to drop migrations table: %v", err)
	}

	migrations := []Migration{
		{
			ID:          "001_create_users",
			Description: "Create users table",
			SQL: `
				CREATE TABLE users (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					email VARCHAR(255) UNIQUE NOT NULL,
					name VARCHAR(255) NOT NULL,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
			`,
		},
		{
			ID:          "002_add_age_to_users",
			Description: "Add age column to users table",
			SQL:         `ALTER TABLE users ADD COLUMN age INTEGER;`,
		},
	}

	// Apply migrations
	result, err := db.Migrate(ctx, migrations)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Verify all migrations were applied
	if len(result.Applied) != 2 {
		t.Errorf("Expected 2 applied migrations, got %d", len(result.Applied))
	}

	if len(result.Skipped) != 0 {
		t.Errorf("Expected 0 skipped migrations, got %d", len(result.Skipped))
	}

	// Verify migration order
	if result.Applied[0].ID != "001_create_users" {
		t.Errorf("Expected first migration to be 001_create_users, got %s", result.Applied[0].ID)
	}

	if result.Applied[1].ID != "002_add_age_to_users" {
		t.Errorf("Expected second migration to be 002_add_age_to_users, got %s", result.Applied[1].ID)
	}

	// Verify tables exist
	var exists bool
	err = db.NewRaw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'users'
		);
	`).Scan(ctx, &exists)
	if err != nil {
		t.Fatalf("Failed to check if users table exists: %v", err)
	}

	if !exists {
		t.Error("Users table should exist")
	}

	// Verify column exists
	var columnExists bool
	err = db.NewRaw(`
		SELECT EXISTS (
			SELECT FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'age'
		);
	`).Scan(ctx, &columnExists)
	if err != nil {
		t.Fatalf("Failed to check if age column exists: %v", err)
	}

	if !columnExists {
		t.Error("Age column should exist")
	}
}

func TestMigration_SkipAlreadyApplied(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Clean up tables from previous runs
	_, _ = db.NewDropTable().IfExists().TableExpr("products").Exec(ctx)
	_, err := db.NewDropTable().IfExists().TableExpr("_dbkit_migrations").Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to drop migrations table: %v", err)
	}

	migrations := []Migration{
		{
			ID:          "001_create_products",
			Description: "Create products table",
			SQL: `
				CREATE TABLE products (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					name VARCHAR(255) NOT NULL,
					price DECIMAL(10,2) NOT NULL
				);
			`,
		},
	}

	// Apply migrations first time
	result1, err := db.Migrate(ctx, migrations)
	if err != nil {
		t.Fatalf("First migrate failed: %v", err)
	}

	if len(result1.Applied) != 1 {
		t.Errorf("Expected 1 applied migration, got %d", len(result1.Applied))
	}

	if len(result1.Skipped) != 0 {
		t.Errorf("Expected 0 skipped migrations, got %d", len(result1.Skipped))
	}

	// Apply migrations second time
	result2, err := db.Migrate(ctx, migrations)
	if err != nil {
		t.Fatalf("Second migrate failed: %v", err)
	}

	if len(result2.Applied) != 0 {
		t.Errorf("Expected 0 applied migrations, got %d", len(result2.Applied))
	}

	if len(result2.Skipped) != 1 {
		t.Errorf("Expected 1 skipped migration, got %d", len(result2.Skipped))
	}

	if result2.Skipped[0] != "001_create_products" {
		t.Errorf("Expected skipped migration to be 001_create_products, got %s", result2.Skipped[0])
	}
}

func TestMigration_ChecksumValidation(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Clean up tables from previous runs
	_, _ = db.NewDropTable().IfExists().TableExpr("orders").Exec(ctx)
	_, err := db.NewDropTable().IfExists().TableExpr("_dbkit_migrations").Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to drop migrations table: %v", err)
	}

	migrations := []Migration{
		{
			ID:          "001_create_orders",
			Description: "Create orders table",
			SQL: `
				CREATE TABLE orders (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL,
					total DECIMAL(10,2) NOT NULL,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
			`,
		},
	}

	// Apply migrations
	result, err := db.Migrate(ctx, migrations)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Errorf("Expected 1 applied migration, got %d", len(result.Applied))
	}

	// Try to apply same migration with different SQL (should fail)
	modifiedMigrations := []Migration{
		{
			ID:          "001_create_orders",
			Description: "Create orders table",
			SQL: `
				CREATE TABLE orders (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL,
					total DECIMAL(10,2) NOT NULL,
					status VARCHAR(50) NOT NULL,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
			`,
		},
	}

	_, err = db.Migrate(ctx, modifiedMigrations)
	if err == nil {
		t.Error("Expected error when applying migration with different checksum")
	}

	// Checksum mismatch should produce an error (not necessarily retryable)
	if err == nil {
		t.Error("Expected error for checksum mismatch")
	}
}

func TestMigration_MixedApplyAndSkip(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Clean up tables from previous runs
	_, _ = db.NewDropTable().IfExists().TableExpr("items").Exec(ctx)
	_, _ = db.NewDropTable().IfExists().TableExpr("categories").Exec(ctx)
	_, err := db.NewDropTable().IfExists().TableExpr("_dbkit_migrations").Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to drop migrations table: %v", err)
	}

	// Apply first migration
	firstMigration := []Migration{
		{
			ID:          "001_create_categories",
			Description: "Create categories table",
			SQL: `
				CREATE TABLE categories (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					name VARCHAR(255) NOT NULL
				);
			`,
		},
	}

	result1, err := db.Migrate(ctx, firstMigration)
	if err != nil {
		t.Fatalf("First migrate failed: %v", err)
	}

	if len(result1.Applied) != 1 {
		t.Errorf("Expected 1 applied migration, got %d", len(result1.Applied))
	}

	// Apply all migrations (first should be skipped, second applied)
	allMigrations := []Migration{
		{
			ID:          "001_create_categories",
			Description: "Create categories table",
			SQL: `
				CREATE TABLE categories (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					name VARCHAR(255) NOT NULL
				);
			`,
		},
		{
			ID:          "002_create_items",
			Description: "Create items table",
			SQL: `
				CREATE TABLE items (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					category_id UUID NOT NULL,
					name VARCHAR(255) NOT NULL,
					FOREIGN KEY (category_id) REFERENCES categories(id)
				);
			`,
		},
	}

	result2, err := db.Migrate(ctx, allMigrations)
	if err != nil {
		t.Fatalf("Second migrate failed: %v", err)
	}

	if len(result2.Applied) != 1 {
		t.Errorf("Expected 1 applied migration, got %d", len(result2.Applied))
	}

	if len(result2.Skipped) != 1 {
		t.Errorf("Expected 1 skipped migration, got %d", len(result2.Skipped))
	}

	if result2.Applied[0].ID != "002_create_items" {
		t.Errorf("Expected applied migration to be 002_create_items, got %s", result2.Applied[0].ID)
	}

	if result2.Skipped[0] != "001_create_categories" {
		t.Errorf("Expected skipped migration to be 001_create_categories, got %s", result2.Skipped[0])
	}
}

func TestMigration_ErrorHandling(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Clean up tables from previous runs
	_, _ = db.NewDropTable().IfExists().TableExpr("should_not_exist").Exec(ctx)
	_, _ = db.NewDropTable().IfExists().TableExpr("valid_table").Exec(ctx)
	_, err := db.NewDropTable().IfExists().TableExpr("_dbkit_migrations").Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to drop migrations table: %v", err)
	}

	migrations := []Migration{
		{
			ID:          "001_valid_migration",
			Description: "Valid migration",
			SQL: `
				CREATE TABLE valid_table (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					name VARCHAR(255) NOT NULL
				);
			`,
		},
		{
			ID:          "002_invalid_migration",
			Description: "Invalid migration with syntax error",
			SQL:         `INVALID SQL SYNTAX;`,
		},
		{
			ID:          "003_should_not_run",
			Description: "This should not run due to previous error",
			SQL: `
				CREATE TABLE should_not_exist (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid()
				);
			`,
		},
	}

	// Apply migrations - should fail on second migration
	_, err = db.Migrate(ctx, migrations)
	if err == nil {
		t.Error("Expected error due to invalid SQL")
	}

	// Verify first table was created but third was not
	var validTableExists bool
	err = db.NewRaw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'valid_table'
		);
	`).Scan(ctx, &validTableExists)
	if err != nil {
		t.Fatalf("Failed to check if valid_table exists: %v", err)
	}

	if !validTableExists {
		t.Error("valid_table should exist because first migration should be rolled back")
	}

	var shouldNotExistTableExists bool
	err = db.NewRaw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'should_not_exist'
		);
	`).Scan(ctx, &shouldNotExistTableExists)
	if err != nil {
		t.Fatalf("Failed to check if should_not_exist table exists: %v", err)
	}

	if shouldNotExistTableExists {
		t.Error("should_not_exist table should not exist due to previous error")
	}
}

func TestMigration_Status(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Clean up tables from previous runs
	_, _ = db.NewDropTable().IfExists().TableExpr("another_table").Exec(ctx)
	_, _ = db.NewDropTable().IfExists().TableExpr("test_table").Exec(ctx)
	_, err := db.NewDropTable().IfExists().TableExpr("_dbkit_migrations").Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to drop migrations table: %v", err)
	}

	migrations := []Migration{
		{
			ID:          "001_create_test_table",
			Description: "Create test table",
			SQL: `
				CREATE TABLE test_table (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					name VARCHAR(255) NOT NULL
				);
			`,
		},
		{
			ID:          "002_create_another_table",
			Description: "Create another table",
			SQL: `
				CREATE TABLE another_table (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					value INTEGER NOT NULL
				);
			`,
		},
	}

	// Check status before applying
	status, err := db.MigrationStatus(ctx, migrations)
	if err != nil {
		t.Fatalf("MigrationStatus failed: %v", err)
	}

	if len(status) != 2 {
		t.Errorf("Expected 2 status entries, got %d", len(status))
	}

	for _, entry := range status {
		if entry.Applied {
			t.Errorf("Migration %s should not be applied yet", entry.ID)
		}
	}

	// Apply first migration
	result, err := db.Migrate(ctx, migrations[:1])
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Errorf("Expected 1 applied migration, got %d", len(result.Applied))
	}

	// Check status after applying first migration
	status, err = db.MigrationStatus(ctx, migrations)
	if err != nil {
		t.Fatalf("MigrationStatus failed: %v", err)
	}

	if len(status) != 2 {
		t.Errorf("Expected 2 status entries, got %d", len(status))
	}

	var foundApplied, foundPending bool
	for _, entry := range status {
		if entry.ID == "001_create_test_table" {
			if !entry.Applied {
				t.Error("First migration should be applied")
			}
			foundApplied = true
		}
		if entry.ID == "002_create_another_table" {
			if entry.Applied {
				t.Error("Second migration should not be applied")
			}
			foundPending = true
		}
	}

	if !foundApplied {
		t.Error("Should find applied migration status")
	}

	if !foundPending {
		t.Error("Should find pending migration status")
	}
}

func TestMigration_GetAppliedMigrations(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Clean up migrations table
	_, err := db.NewDropTable().IfExists().TableExpr("_dbkit_migrations").Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to drop migrations table: %v", err)
	}

	migrations := []Migration{
		{
			ID:          "001_first_migration",
			Description: "First migration",
			SQL:         `SELECT 1;`,
		},
		{
			ID:          "002_second_migration",
			Description: "Second migration",
			SQL:         `SELECT 2;`,
		},
	}

	// Apply migrations
	result, err := db.Migrate(ctx, migrations)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if len(result.Applied) != 2 {
		t.Errorf("Expected 2 applied migrations, got %d", len(result.Applied))
	}

	// Get applied migrations
	applied, err := db.GetAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("GetAppliedMigrations failed: %v", err)
	}

	if len(applied) != 2 {
		t.Errorf("Expected 2 applied migrations, got %d", len(applied))
	}

	// Verify migration details
	migrationMap := make(map[string]AppliedMigration)
	for _, m := range applied {
		migrationMap[m.ID] = m
	}

	firstMigration := migrationMap["001_first_migration"]
	if firstMigration.Description != "First migration" {
		t.Errorf("Expected description 'First migration', got %s", firstMigration.Description)
	}

	if firstMigration.Checksum == "" {
		t.Error("Checksum should not be empty")
	}

	if firstMigration.AppliedAt.IsZero() {
		t.Error("AppliedAt should not be zero")
	}

	// Verify applied order
	if applied[0].ID != "001_first_migration" {
		t.Errorf("Expected first migration to be 001_first_migration, got %s", applied[0].ID)
	}

	if applied[1].ID != "002_second_migration" {
		t.Errorf("Expected second migration to be 002_second_migration, got %s", applied[1].ID)
	}
}

func TestMigration_EmptyMigrations(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Clean up migrations table
	_, err := db.NewDropTable().IfExists().TableExpr("_dbkit_migrations").Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to drop migrations table: %v", err)
	}

	// Apply empty migration list
	result, err := db.Migrate(ctx, []Migration{})
	if err != nil {
		t.Fatalf("Migrate with empty list failed: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Errorf("Expected 0 applied migrations, got %d", len(result.Applied))
	}

	if len(result.Skipped) != 0 {
		t.Errorf("Expected 0 skipped migrations, got %d", len(result.Skipped))
	}

	// Get applied migrations should be empty
	applied, err := db.GetAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("GetAppliedMigrations failed: %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("Expected 0 applied migrations, got %d", len(applied))
	}
}
