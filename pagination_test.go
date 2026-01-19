package dbkit

import (
	"testing"
)

func TestPaginate(t *testing.T) {
	// Test default values
	fn := Paginate(0, 0)
	if fn == nil {
		t.Error("Paginate should return a function")
	}

	// Test with valid values
	fn = Paginate(2, 10)
	if fn == nil {
		t.Error("Paginate should return a function")
	}

	// Test max page size
	fn = Paginate(1, 200)
	if fn == nil {
		t.Error("Paginate should return a function even with large page size")
	}
}

func TestEncodeCursor(t *testing.T) {
	cursor := EncodeCursor("test-id", "sort-value")
	if cursor == "" {
		t.Error("EncodeCursor should return a non-empty string")
	}

	// Decode and verify
	decoded, err := DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("DecodeCursor failed: %v", err)
	}

	if decoded.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", decoded.ID)
	}

	if decoded.SortValue != "sort-value" {
		t.Errorf("Expected SortValue 'sort-value', got %s", decoded.SortValue)
	}
}

func TestDecodeCursor_Empty(t *testing.T) {
	decoded, err := DecodeCursor("")
	if err != nil {
		t.Errorf("DecodeCursor should not error on empty string: %v", err)
	}
	if decoded != nil {
		t.Error("DecodeCursor should return nil for empty string")
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	_, err := DecodeCursor("invalid-cursor")
	if err == nil {
		t.Error("DecodeCursor should error on invalid cursor")
	}
}

func TestCursorPaginate(t *testing.T) {
	// Test with no cursor
	fn := CursorPaginate("id", "", "", 10, true)
	if fn == nil {
		t.Error("CursorPaginate should return a function")
	}

	// Test with cursor
	cursor := EncodeCursor("test-id", "")
	fn = CursorPaginate("id", "", cursor, 10, true)
	if fn == nil {
		t.Error("CursorPaginate should return a function")
	}

	// Test backward pagination
	fn = CursorPaginate("id", "", cursor, 10, false)
	if fn == nil {
		t.Error("CursorPaginate should return a function for backward pagination")
	}

	// Test with sort column
	cursor = EncodeCursor("test-id", "2024-01-01")
	fn = CursorPaginate("id", "created_at", cursor, 10, true)
	if fn == nil {
		t.Error("CursorPaginate should return a function with sort column")
	}
}

func TestCursorPaginateResult(t *testing.T) {
	type Item struct {
		ID   string
		Name string
	}

	items := []Item{
		{ID: "1", Name: "Item 1"},
		{ID: "2", Name: "Item 2"},
		{ID: "3", Name: "Item 3"},
	}

	cursorFn := func(i Item) string {
		return EncodeCursor(i.ID, "")
	}

	// Test with items less than limit
	result, pageInfo := CursorPaginateResult(items, 10, true, cursorFn)
	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}
	if pageInfo.HasNextPage {
		t.Error("Should not have next page")
	}

	// Test with items more than limit (has more)
	moreItems := []Item{
		{ID: "1", Name: "Item 1"},
		{ID: "2", Name: "Item 2"},
		{ID: "3", Name: "Item 3"},
		{ID: "4", Name: "Item 4"}, // Extra item
	}

	result, pageInfo = CursorPaginateResult(moreItems, 3, true, cursorFn)
	if len(result) != 3 {
		t.Errorf("Expected 3 items (trimmed), got %d", len(result))
	}
	if !pageInfo.HasNextPage {
		t.Error("Should have next page")
	}

	// Test backward pagination
	result, pageInfo = CursorPaginateResult(moreItems, 3, false, cursorFn)
	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}
	if !pageInfo.HasPreviousPage {
		t.Error("Should have previous page")
	}
}

func TestCursorPaginateResult_Empty(t *testing.T) {
	type Item struct {
		ID string
	}

	var items []Item
	cursorFn := func(i Item) string { return i.ID }

	result, pageInfo := CursorPaginateResult(items, 10, true, cursorFn)
	if len(result) != 0 {
		t.Error("Expected empty result")
	}
	if pageInfo.StartCursor != "" || pageInfo.EndCursor != "" {
		t.Error("Cursors should be empty for empty result")
	}
}

func TestKeysetPaginate(t *testing.T) {
	// Test with no last value
	fn := KeysetPaginate("id", nil, 10)
	if fn == nil {
		t.Error("KeysetPaginate should return a function")
	}

	// Test with last value
	fn = KeysetPaginate("id", "last-id", 10)
	if fn == nil {
		t.Error("KeysetPaginate should return a function")
	}

	// Test default limit
	fn = KeysetPaginate("id", nil, 0)
	if fn == nil {
		t.Error("KeysetPaginate should return a function with default limit")
	}
}

func TestOffsetPage_Fields(t *testing.T) {
	page := OffsetPage[string]{
		Items:      []string{"a", "b", "c"},
		Page:       1,
		PageSize:   10,
		TotalItems: 100,
		TotalPages: 10,
	}

	if len(page.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(page.Items))
	}

	if page.TotalPages != 10 {
		t.Errorf("Expected 10 total pages, got %d", page.TotalPages)
	}
}

func TestCursorPage_Fields(t *testing.T) {
	page := CursorPage[string]{
		Items: []string{"a", "b"},
		PageInfo: PageInfo{
			HasNextPage: true,
			EndCursor:   "cursor",
		},
	}

	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(page.Items))
	}

	if !page.PageInfo.HasNextPage {
		t.Error("Expected HasNextPage to be true")
	}
}
