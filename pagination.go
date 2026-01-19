package dbkit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/uptrace/bun"
)

// PageInfo contains pagination metadata.
type PageInfo struct {
	HasNextPage     bool   `json:"has_next_page"`
	HasPreviousPage bool   `json:"has_previous_page"`
	StartCursor     string `json:"start_cursor,omitempty"`
	EndCursor       string `json:"end_cursor,omitempty"`
	TotalCount      int    `json:"total_count,omitempty"`
}

// OffsetPage represents an offset-based paginated result.
type OffsetPage[T any] struct {
	Items      []T      `json:"items"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	TotalItems int      `json:"total_items"`
	TotalPages int      `json:"total_pages"`
	PageInfo   PageInfo `json:"page_info"`
}

// CursorPage represents a cursor-based paginated result.
type CursorPage[T any] struct {
	Items    []T      `json:"items"`
	PageInfo PageInfo `json:"page_info"`
}

// PaginationOptions configures pagination behavior.
type PaginationOptions struct {
	// Page number (1-indexed) for offset pagination
	Page int
	// PageSize is the number of items per page
	PageSize int
	// After cursor for forward cursor pagination
	After string
	// Before cursor for backward cursor pagination
	Before string
	// First N items (cursor pagination)
	First int
	// Last N items (cursor pagination)
	Last int
	// IncludeTotalCount includes total count in response (can be expensive)
	IncludeTotalCount bool
}

// DefaultPageSize is the default number of items per page.
const DefaultPageSize = 20

// MaxPageSize is the maximum allowed page size.
const MaxPageSize = 100

// Paginate applies offset-based pagination to a query.
// Returns a query modifier that can be used with Apply().
//
// Usage:
//
//	var users []User
//	db.NewSelect().Model(&users).Apply(dbkit.Paginate(2, 10)).Scan(ctx)
func Paginate(page, pageSize int) func(*bun.SelectQuery) *bun.SelectQuery {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	offset := (page - 1) * pageSize

	return func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Limit(pageSize).Offset(offset)
	}
}

// PaginateWithCount executes an offset-paginated query and returns results with metadata.
//
// Usage:
//
//	page, err := dbkit.PaginateWithCount[User](ctx, db, 1, 10, func(q *bun.SelectQuery) *bun.SelectQuery {
//	    return q.Where("active = ?", true).Order("created_at DESC")
//	})
func PaginateWithCount[T any](ctx context.Context, db bun.IDB, page, pageSize int, queryFn func(*bun.SelectQuery) *bun.SelectQuery) (*OffsetPage[T], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	offset := (page - 1) * pageSize

	// Get total count
	var model T
	countQuery := db.NewSelect().Model(&model)
	if queryFn != nil {
		countQuery = queryFn(countQuery)
	}
	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		return nil, wrapError(err, "PaginateWithCount.Count")
	}

	// Get items
	var items []T
	itemsQuery := db.NewSelect().Model(&items).Limit(pageSize).Offset(offset)
	if queryFn != nil {
		itemsQuery = queryFn(itemsQuery)
	}
	err = itemsQuery.Scan(ctx)
	if err != nil {
		return nil, wrapError(err, "PaginateWithCount.Scan")
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return &OffsetPage[T]{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalCount,
		TotalPages: totalPages,
		PageInfo: PageInfo{
			HasNextPage:     page < totalPages,
			HasPreviousPage: page > 1,
			TotalCount:      totalCount,
		},
	}, nil
}

// Cursor represents a pagination cursor.
type Cursor struct {
	ID        string `json:"id"`
	SortValue string `json:"sv,omitempty"`
}

// EncodeCursor encodes a cursor to a base64 string.
func EncodeCursor(id string, sortValue string) string {
	c := Cursor{ID: id, SortValue: sortValue}
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor decodes a base64 cursor string.
func DecodeCursor(cursor string) (*Cursor, error) {
	if cursor == "" {
		return nil, nil
	}

	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	return &c, nil
}

// CursorPaginate applies cursor-based pagination to a query.
// The idColumn should be the primary key or a unique column.
// The sortColumn is optional and used for sorting before the ID.
//
// Usage:
//
//	var users []User
//	db.NewSelect().Model(&users).
//	    Apply(dbkit.CursorPaginate("id", "", afterCursor, 10, true)).
//	    Scan(ctx)
func CursorPaginate(idColumn, sortColumn, cursor string, limit int, forward bool) func(*bun.SelectQuery) *bun.SelectQuery {
	if limit < 1 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	return func(q *bun.SelectQuery) *bun.SelectQuery {
		c, err := DecodeCursor(cursor)
		if err != nil || c == nil {
			// No cursor, just apply limit and order
			if sortColumn != "" {
				if forward {
					q = q.Order(sortColumn + " ASC")
				} else {
					q = q.Order(sortColumn + " DESC")
				}
			}
			if forward {
				q = q.Order(idColumn + " ASC")
			} else {
				q = q.Order(idColumn + " DESC")
			}
			return q.Limit(limit + 1) // Fetch one extra to check hasMore
		}

		// Apply cursor filter
		if sortColumn != "" && c.SortValue != "" {
			if forward {
				q = q.Where("("+sortColumn+" > ?) OR ("+sortColumn+" = ? AND "+idColumn+" > ?)",
					c.SortValue, c.SortValue, c.ID)
				q = q.Order(sortColumn + " ASC")
			} else {
				q = q.Where("("+sortColumn+" < ?) OR ("+sortColumn+" = ? AND "+idColumn+" < ?)",
					c.SortValue, c.SortValue, c.ID)
				q = q.Order(sortColumn + " DESC")
			}
		} else {
			if forward {
				q = q.Where(idColumn+" > ?", c.ID)
			} else {
				q = q.Where(idColumn+" < ?", c.ID)
			}
		}

		if forward {
			q = q.Order(idColumn + " ASC")
		} else {
			q = q.Order(idColumn + " DESC")
		}

		return q.Limit(limit + 1)
	}
}

// CursorPaginateResult processes cursor pagination results and builds page info.
// Pass the items fetched with limit+1, and it will trim and determine hasMore.
//
// Usage:
//
//	items, pageInfo := dbkit.CursorPaginateResult(users, 10, true, func(u User) string {
//	    return dbkit.EncodeCursor(u.ID, "")
//	})
func CursorPaginateResult[T any](items []T, limit int, forward bool, cursorFn func(T) string) ([]T, PageInfo) {
	if len(items) == 0 {
		return items, PageInfo{}
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	// Reverse if backward pagination
	if !forward {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}

	pageInfo := PageInfo{}
	if len(items) > 0 {
		pageInfo.StartCursor = cursorFn(items[0])
		pageInfo.EndCursor = cursorFn(items[len(items)-1])
	}

	if forward {
		pageInfo.HasNextPage = hasMore
		// HasPreviousPage is true if we had a cursor (not first page)
	} else {
		pageInfo.HasPreviousPage = hasMore
	}

	return items, pageInfo
}

// KeysetPaginate applies keyset pagination (also known as seek method).
// This is more efficient than offset for large datasets.
//
// Usage:
//
//	var users []User
//	db.NewSelect().Model(&users).
//	    Apply(dbkit.KeysetPaginate("id", lastID, 10)).
//	    Order("id ASC").
//	    Scan(ctx)
func KeysetPaginate(column string, lastValue interface{}, limit int) func(*bun.SelectQuery) *bun.SelectQuery {
	if limit < 1 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	return func(q *bun.SelectQuery) *bun.SelectQuery {
		if lastValue != nil && lastValue != "" {
			q = q.Where(column+" > ?", lastValue)
		}
		return q.Limit(limit)
	}
}
