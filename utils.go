package dbkit

import "strings"

// joinColumns joins column names with commas for SQL queries.
func joinColumns(cols []string) string {
	return strings.Join(cols, ", ")
}
