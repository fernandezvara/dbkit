package dbkit

import (
	"testing"

	"github.com/uptrace/bun"
)

func TestHelpers_FindByID(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test record
	model := &TestModel{Name: "Find By ID Test", Email: "findbyid@example.com", Age: 25}
	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test FindByID
	found, err := FindByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.ID != model.ID {
		t.Errorf("Expected ID %s, got %s", model.ID, found.ID)
	}

	if found.Name != model.Name {
		t.Errorf("Expected name %s, got %s", model.Name, found.Name)
	}

	if found.Email != model.Email {
		t.Errorf("Expected email %s, got %s", model.Email, found.Email)
	}

	if found.Age != model.Age {
		t.Errorf("Expected age %d, got %d", model.Age, found.Age)
	}

	// Test FindByID with non-existent ID
	_, err = FindByID[TestModel](ctx, db, "00000000-0000-0000-0000-000000000000")
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got %v", err)
	}
}

func TestHelpers_FindOne(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test records
	models := []*TestModel{
		{Name: "Alice", Email: "alice@example.com", Age: 25},
		{Name: "Bob", Email: "bob@example.com", Age: 30},
		{Name: "Charlie", Email: "charlie@example.com", Age: 35},
	}

	err = CreateMany(ctx, db, models)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	// Test FindOne with filter
	found, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "bob@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if found.Name != "Bob" {
		t.Errorf("Expected name Bob, got %s", found.Name)
	}

	if found.Age != 30 {
		t.Errorf("Expected age 30, got %d", found.Age)
	}

	// Test FindOne with no results
	_, err = FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "nonexistent@example.com")
	})
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got %v", err)
	}

	// Test FindOne with multiple results (should return first)
	found, err = FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("age > ?", 25).Order("name")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if found.Name != "Bob" {
		t.Errorf("Expected name Bob (first in order), got %s", found.Name)
	}
}

func TestHelpers_FindAll(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test records
	models := []*TestModel{
		{Name: "FA_User1", Email: "fa_user1@example.com", Age: 55, Active: true},
		{Name: "FA_User2", Email: "fa_user2@example.com", Age: 60, Active: false},
		{Name: "FA_User3", Email: "fa_user3@example.com", Age: 65, Active: true},
		{Name: "FA_User4", Email: "fa_user4@example.com", Age: 70, Active: false},
	}

	err = CreateMany(ctx, db, models)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	// Test FindAll with filter
	activeUsers, err := FindAll[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("active = ?", true).Order("age")
	})
	if err != nil {
		t.Fatalf("FindAll with filter failed: %v", err)
	}

	if len(activeUsers) != 2 {
		t.Errorf("Expected 2 active users, got %d", len(activeUsers))
	}

	if len(activeUsers) >= 1 && activeUsers[0].Age != 55 {
		t.Errorf("Expected first active user to be age 55, got %d", activeUsers[0].Age)
	}

	if len(activeUsers) >= 2 && activeUsers[1].Age != 65 {
		t.Errorf("Expected second active user to be age 65, got %d", activeUsers[1].Age)
	}

	// Test FindAll with limit
	limited, err := FindAll[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Limit(2).Order("age")
	})
	if err != nil {
		t.Fatalf("FindAll with limit failed: %v", err)
	}

	if len(limited) != 2 {
		t.Errorf("Expected 2 limited records, got %d", len(limited))
	}

	if limited[0].Age != 55 {
		t.Errorf("Expected first limited record to be age 55, got %d", limited[0].Age)
	}
}

func TestHelpers_ExistsByID(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test record
	model := &TestModel{Name: "Exists Test", Email: "exists@example.com", Age: 28}
	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test ExistsByID with existing record
	exists, err := ExistsByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("ExistsByID failed: %v", err)
	}

	if !exists {
		t.Error("Expected true for existing record")
	}

	// Test ExistsByID with non-existent record
	exists, err = ExistsByID[TestModel](ctx, db, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("ExistsByID failed: %v", err)
	}

	if exists {
		t.Error("Expected false for non-existing record")
	}
}

func TestHelpers_Count(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test records
	models := []*TestModel{
		{Name: "HC_Count1", Email: "hc_count1@example.com", Age: 25, Active: true},
		{Name: "HC_Count2", Email: "hc_count2@example.com", Age: 30, Active: false},
		{Name: "HC_Count3", Email: "hc_count3@example.com", Age: 35, Active: true},
	}

	err = CreateMany(ctx, db, models)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	// Test Count all
	total, err := Count[TestModel](ctx, db, nil)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected total count 3, got %d", total)
	}

	// Test Count with filter
	activeCount, err := Count[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("active = ?", true)
	})
	if err != nil {
		t.Fatalf("Count with filter failed: %v", err)
	}

	if activeCount != 2 {
		t.Errorf("Expected active count 2, got %d", activeCount)
	}

	// Test Count with age filter
	ageCount, err := Count[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("age > ?", 30)
	})
	if err != nil {
		t.Fatalf("Count with age filter failed: %v", err)
	}

	if ageCount != 1 {
		t.Errorf("Expected age count 1, got %d", ageCount)
	}

	// Test Count with no matching records
	zeroCount, err := Count[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("age > ?", 100)
	})
	if err != nil {
		t.Fatalf("Count with no matches failed: %v", err)
	}

	if zeroCount != 0 {
		t.Errorf("Expected zero count, got %d", zeroCount)
	}
}

func TestHelpers_UpdateColumns(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test record
	model := &TestModel{Name: "Update Columns Test", Email: "update@example.com", Age: 25, Active: true}
	err = Create(ctx, db, model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test UpdateColumns
	model.Name = "Updated Name"
	model.Age = 30
	model.Active = false

	err = UpdateColumns(ctx, db, model, "name", "age")
	if err != nil {
		t.Fatalf("UpdateColumns failed: %v", err)
	}

	// Verify only specified columns were updated
	found, err := FindByID[TestModel](ctx, db, model.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.Name != "Updated Name" {
		t.Errorf("Expected name to be updated to 'Updated Name', got %s", found.Name)
	}

	if found.Age != 30 {
		t.Errorf("Expected age to be updated to 30, got %d", found.Age)
	}

	if !found.Active {
		t.Error("Expected Active to remain unchanged")
	}

	if found.Email != "update@example.com" {
		t.Errorf("Expected Email to remain unchanged, got %s", found.Email)
	}
}

func TestHelpers_UpdateWhere(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test records
	models := []*TestModel{
		{Name: "UpdateWhere1", Email: "updatewhere1@example.com", Age: 25},
		{Name: "UpdateWhere2", Email: "updatewhere2@example.com", Age: 30},
		{Name: "UpdateWhere3", Email: "updatewhere3@example.com", Age: 35},
		{Name: "UpdateWhere4", Email: "updatewhere4@example.com", Age: 40},
	}

	err = CreateMany(ctx, db, models)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	// Test UpdateWhere
	updateModel := &TestModel{Age: 99}
	rowsAffected, err := UpdateWhere[TestModel](ctx, db, updateModel, func(q *bun.UpdateQuery) *bun.UpdateQuery {
		return q.Where("age < ?", 35)
	})
	if err != nil {
		t.Fatalf("UpdateWhere failed: %v", err)
	}

	if rowsAffected != 2 {
		t.Errorf("Expected 2 rows affected, got %d", rowsAffected)
	}

	// Verify updates
	updated1, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "updatewhere1@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if updated1.Age != 99 {
		t.Errorf("Expected age to be updated to 99, got %d", updated1.Age)
	}

	updated2, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "updatewhere2@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if updated2.Age != 99 {
		t.Errorf("Expected age to be updated to 99, got %d", updated2.Age)
	}

	// Verify records not matching condition were not updated
	notUpdated, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "updatewhere3@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if notUpdated.Age != 35 {
		t.Errorf("Expected age to remain 35, got %d", notUpdated.Age)
	}
}

func TestHelpers_DeleteWhere(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Create test records
	models := []*TestModel{
		{Name: "DeleteWhere1", Email: "deletewhere1@example.com", Age: 25},
		{Name: "DeleteWhere2", Email: "deletewhere2@example.com", Age: 30},
		{Name: "DeleteWhere3", Email: "deletewhere3@example.com", Age: 35},
		{Name: "DeleteWhere4", Email: "deletewhere4@example.com", Age: 40},
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

	if rowsAffected != 2 {
		t.Errorf("Expected 2 rows affected, got %d", rowsAffected)
	}

	// Verify deletions
	_, err = FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "deletewhere3@example.com")
	})
	if !IsNotFound(err) {
		t.Error("Expected record with age 35 to be deleted")
	}

	_, err = FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "deletewhere4@example.com")
	})
	if !IsNotFound(err) {
		t.Error("Expected record with age 40 to be deleted")
	}

	// Verify records not matching condition still exist
	remaining1, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "deletewhere1@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if remaining1.Age != 25 {
		t.Errorf("Expected remaining record age 25, got %d", remaining1.Age)
	}

	remaining2, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "deletewhere2@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if remaining2.Age != 30 {
		t.Errorf("Expected remaining record age 30, got %d", remaining2.Age)
	}

	// Verify total count
	total, err := Count[TestModel](ctx, db, nil)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected final count 2, got %d", total)
	}
}

func TestHelpers_Upsert(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	var err error
	ctx := createTable(t, db)

	// Test Upsert (insert)
	model := &TestModel{Name: "Upsert Insert", Email: "upsert@example.com", Age: 25}
	err = Upsert(ctx, db, model, []string{"email"}, []string{"name", "age"})
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

	if inserted.Name != "Upsert Insert" {
		t.Errorf("Expected name 'Upsert Insert', got %s", inserted.Name)
	}

	if inserted.Age != 25 {
		t.Errorf("Expected age 25, got %d", inserted.Age)
	}

	originalID := inserted.ID

	// Test Upsert (update)
	model.Name = "Upsert Update"
	model.Age = 30
	err = Upsert(ctx, db, model, []string{"email"}, []string{"name", "age"})
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

	if updated.Name != "Upsert Update" {
		t.Errorf("Expected name 'Upsert Update', got %s", updated.Name)
	}

	if updated.Age != 30 {
		t.Errorf("Expected age 30, got %d", updated.Age)
	}

	// Verify ID didn't change
	if updated.ID != originalID {
		t.Error("ID should not change during upsert update")
	}

	// Test upsert with conflict (different record with same email)
	anotherModel := &TestModel{
		ID:    "00000000-0000-0000-0000-000000000000",
		Name:  "Another",
		Email: "upsert@example.com", // Same email
		Age:   99,
	}
	err = Upsert(ctx, db, anotherModel, []string{"email"}, []string{"name", "age"})
	if err != nil {
		t.Fatalf("Upsert with conflict failed: %v", err)
	}

	// Verify the original record was updated, not a new one created
	final, err := FindOne[TestModel](ctx, db, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("email = ?", "upsert@example.com")
	})
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if final.Name != "Another" {
		t.Errorf("Expected name 'Another' after conflict upsert, got %s", final.Name)
	}

	if final.Age != 99 {
		t.Errorf("Expected age 99 after conflict upsert, got %d", final.Age)
	}

	// Verify total count is still 1
	total, err := Count[TestModel](ctx, db, nil)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected count 1 after upsert conflict, got %d", total)
	}
}
