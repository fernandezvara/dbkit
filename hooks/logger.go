// Package hooks provides observability hooks for dbkit
package hooks

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// LoggerHook implements query logging
type LoggerHook struct {
	logger        *slog.Logger
	logAll        bool
	slowThreshold time.Duration
}

// NewLoggerHook creates a new logger hook
func NewLoggerHook(logger *slog.Logger, logAll bool, slowThreshold time.Duration) *LoggerHook {
	return &LoggerHook{
		logger:        logger,
		logAll:        logAll,
		slowThreshold: slowThreshold,
	}
}

// BeforeQuery is called before a query is executed
func (h *LoggerHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery is called after a query is executed
func (h *LoggerHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime)

	// Skip if not logging all and not slow
	if !h.logAll && (h.slowThreshold == 0 || duration < h.slowThreshold) {
		return
	}

	query := event.Query
	if len(query) > 500 {
		query = query[:500] + "..."
	}

	attrs := []slog.Attr{
		slog.Duration("duration", duration),
		slog.String("operation", OperationType(event.Query)),
	}

	if h.logAll {
		attrs = append(attrs, slog.String("query", query))
	}

	if event.Err != nil {
		attrs = append(attrs, slog.String("error", event.Err.Error()))
		h.logger.LogAttrs(ctx, slog.LevelError, "database query failed", attrs...)
	} else if h.slowThreshold > 0 && duration >= h.slowThreshold {
		attrs = append(attrs, slog.String("query", query))
		h.logger.LogAttrs(ctx, slog.LevelWarn, "slow database query", attrs...)
	} else if h.logAll {
		h.logger.LogAttrs(ctx, slog.LevelDebug, "database query", attrs...)
	}
}

// OperationType extracts the operation type from a query
func OperationType(query string) string {
	query = strings.TrimSpace(strings.ToUpper(query))
	switch {
	case strings.HasPrefix(query, "SELECT"):
		return "select"
	case strings.HasPrefix(query, "INSERT"):
		return "insert"
	case strings.HasPrefix(query, "UPDATE"):
		return "update"
	case strings.HasPrefix(query, "DELETE"):
		return "delete"
	case strings.HasPrefix(query, "CREATE"):
		return "create"
	case strings.HasPrefix(query, "DROP"):
		return "drop"
	case strings.HasPrefix(query, "ALTER"):
		return "alter"
	case strings.HasPrefix(query, "BEGIN"):
		return "begin"
	case strings.HasPrefix(query, "COMMIT"):
		return "commit"
	case strings.HasPrefix(query, "ROLLBACK"):
		return "rollback"
	case strings.HasPrefix(query, "SAVEPOINT"):
		return "savepoint"
	case strings.HasPrefix(query, "RELEASE"):
		return "release"
	default:
		return "other"
	}
}
