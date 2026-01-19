package dbkit

import (
	"context"
	"testing"
)

func TestBatchInsert_Empty(t *testing.T) {
	count, err := BatchInsert[TestModel](context.Background(), nil, nil, 100)
	if err != nil {
		t.Errorf("BatchInsert with empty slice should not error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 count, got %d", count)
	}
}

func TestBatchUpdate_Empty(t *testing.T) {
	count, err := BatchUpdate[TestModel](context.Background(), nil, nil, 100)
	if err != nil {
		t.Errorf("BatchUpdate with empty slice should not error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 count, got %d", count)
	}
}

func TestBatchDelete_Empty(t *testing.T) {
	count, err := BatchDelete[TestModel](context.Background(), nil, nil, 100)
	if err != nil {
		t.Errorf("BatchDelete with empty slice should not error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 count, got %d", count)
	}
}

func TestBatchUpsert_Empty(t *testing.T) {
	count, err := BatchUpsert[TestModel](context.Background(), nil, nil, nil, nil, 100)
	if err != nil {
		t.Errorf("BatchUpsert with empty slice should not error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 count, got %d", count)
	}
}

func TestBulkInsertReturning_Empty(t *testing.T) {
	var items []TestModel
	result, err := BulkInsertReturning[TestModel](context.Background(), nil, items)
	if err != nil {
		t.Errorf("BulkInsertReturning with empty slice should not error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestBatchSize_Default(t *testing.T) {
	if BatchSize != 100 {
		t.Errorf("Expected default BatchSize to be 100, got %d", BatchSize)
	}
}
