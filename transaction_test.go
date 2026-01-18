package dbkit

import (
	"context"
	"errors"
	"testing"

	"github.com/uptrace/bun"
)

func TestTransaction_Commit(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test successful transaction
	model := &TestModel{Name: "Transaction Test", Email: "tx@example.com", Age: 25}

	err = db.Transaction(ctx, func(tx *Tx) error {
		return Create(ctx, tx, model)
	})
	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify record was committed
	found, err := FindByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.Name != "Transaction Test" {
		t.Errorf("Expected name Transaction Test, got %s", found.Name)
	}
}

func TestTransaction_Rollback(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test transaction rollback
	model := &TestModel{Name: "Rollback Test", Email: "rollback@example.com", Age: 25}

	err = db.Transaction(ctx, func(tx *Tx) error {
		if err := Create(ctx, tx, model); err != nil {
			return err
		}
		return errors.New("intentional error to trigger rollback")
	})
	if err == nil {
		t.Fatal("Expected error from transaction")
	}

	if err.Error() != "intentional error to trigger rollback" {
		t.Errorf("Expected intentional error, got %v", err)
	}

	// Verify record was not committed
	_, err = FindByID[TestModel](ctx, db, model.ID)
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got %v", err)
	}
}

func TestTransaction_ManualCommit(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test manual transaction control
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	model := &TestModel{Name: "Manual Commit", Email: "manual@example.com", Age: 30}
	err = Create(ctx, tx, model)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Create failed: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify record was committed
	found, err := FindByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.Name != "Manual Commit" {
		t.Errorf("Expected name Manual Commit, got %s", found.Name)
	}
}

func TestTransaction_ManualRollback(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test manual rollback
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	model := &TestModel{Name: "Manual Rollback", Email: "manualrollback@example.com", Age: 35}
	err = Create(ctx, tx, model)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Create failed: %v", err)
	}

	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify record was not committed
	_, err = FindByID[TestModel](ctx, db, model.ID)
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got %v", err)
	}
}

func TestTransaction_Nested_Commit(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test nested transaction (savepoint)
	err = db.Transaction(ctx, func(tx *Tx) error {
		// Outer transaction
		outerModel := &TestModel{Name: "Outer", Email: "outer@example.com", Age: 40}
		if err := Create(ctx, tx, outerModel); err != nil {
			return err
		}

		// Nested transaction
		err := tx.Transaction(ctx, func(tx2 *Tx) error {
			innerModel := &TestModel{Name: "Inner", Email: "inner@example.com", Age: 45}
			return Create(ctx, tx2, innerModel)
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Nested transaction failed: %v", err)
	}

	// Verify both records were committed
	outerFound, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "outer@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed for outer: %v", err)
	}

	if outerFound.Name != "Outer" {
		t.Errorf("Expected name Outer, got %s", outerFound.Name)
	}

	innerFound, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "inner@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed for inner: %v", err)
	}

	if innerFound.Name != "Inner" {
		t.Errorf("Expected name Inner, got %s", innerFound.Name)
	}
}

func TestTransaction_Nested_Rollback(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test nested transaction rollback
	err = db.Transaction(ctx, func(tx *Tx) error {
		// Outer transaction
		outerModel := &TestModel{Name: "Outer Nested", Email: "outernested@example.com", Age: 50}
		if err := Create(ctx, tx, outerModel); err != nil {
			return err
		}

		// Nested transaction that fails
		err := tx.Transaction(ctx, func(tx2 *Tx) error {
			innerModel := &TestModel{Name: "Inner Nested", Email: "innernested@example.com", Age: 55}
			if err := Create(ctx, tx2, innerModel); err != nil {
				return err
			}
			return errors.New("nested transaction error")
		})
		if err != nil {
			// Nested transaction should rollback, but outer should continue
			// This is expected behavior for savepoints
		}

		// Create another record in outer transaction
		anotherModel := &TestModel{Name: "Another Outer", Email: "another@example.com", Age: 60}
		if err := Create(ctx, tx, anotherModel); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Outer transaction failed: %v", err)
	}

	// Verify outer records were committed
	outerFound, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "outernested@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed for outer: %v", err)
	}

	if outerFound.Name != "Outer Nested" {
		t.Errorf("Expected name Outer Nested, got %s", outerFound.Name)
	}

	anotherFound, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "another@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed for another: %v", err)
	}

	if anotherFound.Name != "Another Outer" {
		t.Errorf("Expected name Another Outer, got %s", anotherFound.Name)
	}

	// Verify inner record was not committed
	_, err = FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "innernested@example.com")
	})
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error for inner record, got %v", err)
	}
}

func TestTransaction_ReadOnly(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table and add test data
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	testModel := &TestModel{Name: "Read Only Test", Email: "readonly@example.com", Age: 65}
	err = Create(ctx, db, testModel)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test read-only transaction
	err = db.ReadOnlyTransaction(ctx, func(tx *Tx) error {
		// Should be able to read
		found, err := FindByID[TestModel](ctx, tx, testModel.ID)
		if err != nil {
			return err
		}

		if found.Name != "Read Only Test" {
			t.Errorf("Expected name Read Only Test, got %s", found.Name)
		}

		// Should not be able to write (this will cause an error)
		newModel := &TestModel{Name: "Should Not Work", Email: "shouldnotwork@example.com", Age: 70}
		return Create(ctx, tx, newModel)
	})
	if err == nil {
		t.Error("Expected error when writing in read-only transaction")
	}
}

func TestTransaction_MultipleOperations(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test multiple operations in a transaction
	models := []*TestModel{
		{Name: "Multi 1", Email: "multi1@example.com", Age: 25},
		{Name: "Multi 2", Email: "multi2@example.com", Age: 30},
		{Name: "Multi 3", Email: "multi3@example.com", Age: 35},
	}

	err = db.Transaction(ctx, func(tx *Tx) error {
		// Create multiple records
		for _, model := range models {
			if err := Create(ctx, tx, model); err != nil {
				return err
			}
		}

		// Update one record
		models[0].Age = 26
		if err := Update(ctx, tx, models[0]); err != nil {
			return err
		}

		// Delete one record
		if err := Delete(ctx, tx, models[2]); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Multi-operation transaction failed: %v", err)
	}

	// Verify results
	// Check updated record
	updated, err := FindByID[TestModel](ctx, db, models[0].ID)
	if err != nil {
		t.Fatalf("FindByID failed for updated record: %v", err)
	}

	if updated.Age != 26 {
		t.Errorf("Expected age 26, got %d", updated.Age)
	}

	// Check remaining record
	remaining, err := FindByID[TestModel](ctx, db, models[1].ID)
	if err != nil {
		t.Fatalf("FindByID failed for remaining record: %v", err)
	}

	if remaining.Name != "Multi 2" {
		t.Errorf("Expected name Multi 2, got %s", remaining.Name)
	}

	// Check deleted record
	_, err = FindByID[TestModel](ctx, db, models[2].ID)
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error for deleted record, got %v", err)
	}

	// Verify total count
	count, err := Count[TestModel](ctx, db, nil)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestTransaction_RollbackTo(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test rollback to savepoint
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	defer tx.Rollback()

	// Create first record
	model1 := &TestModel{Name: "Savepoint 1", Email: "savepoint1@example.com", Age: 40}
	err = Create(ctx, tx, model1)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create savepoint
	err = tx.Savepoint(ctx, "test_savepoint")
	if err != nil {
		t.Fatalf("Savepoint failed: %v", err)
	}

	// Create second record
	model2 := &TestModel{Name: "Savepoint 2", Email: "savepoint2@example.com", Age: 45}
	err = Create(ctx, tx, model2)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Rollback to savepoint
	err = tx.RollbackTo(ctx, "test_savepoint")
	if err != nil {
		t.Fatalf("RollbackTo failed: %v", err)
	}

	// Create third record
	model3 := &TestModel{Name: "Savepoint 3", Email: "savepoint3@example.com", Age: 50}
	err = Create(ctx, tx, model3)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify results
	// First record should exist
	_, err = FindByID[TestModel](ctx, db, model1.ID)
	if err != nil {
		t.Errorf("Expected first record to exist, got %v", err)
	}

	// Second record should not exist (rolled back)
	_, err = FindByID[TestModel](ctx, db, model2.ID)
	if !IsNotFound(err) {
		t.Errorf("Expected second record to not exist, got %v", err)
	}

	// Third record should exist
	_, err = FindByID[TestModel](ctx, db, model3.ID)
	if err != nil {
		t.Errorf("Expected third record to exist, got %v", err)
	}
}

func TestTransaction_DBKitAccess(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create table
	_, err := db.NewCreateTable().Model((*TestModel)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test accessing parent DB from transaction
	err = db.Transaction(ctx, func(tx *Tx) error {
		parentDB := tx.DBKit()
		if parentDB == nil {
			return errors.New("parent DB should not be nil")
		}

		// Use parent DB to create a record (this should work)
		model := &TestModel{Name: "Parent DB Test", Email: "parentdb@example.com", Age: 55}
		return Create(ctx, parentDB, model)
	})
	if err != nil {
		t.Fatalf("Transaction with parent DB access failed: %v", err)
	}

	// Verify record was created
	found, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "parentdb@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if found.Name != "Parent DB Test" {
		t.Errorf("Expected name Parent DB Test, got %s", found.Name)
	}
}
