package dbkit

import (
	"context"
	"database/sql"
	"time"
)

// HealthStatus represents the database health status
type HealthStatus struct {
	Healthy   bool          `json:"healthy"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
	PoolStats PoolStats     `json:"pool_stats"`
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxIdleTimeClosed  int64         `json:"max_idle_time_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}

// Health performs a health check with detailed status
func (db *DBKit) Health(ctx context.Context) HealthStatus {
	start := time.Now()

	err := db.Ping(ctx)
	latency := time.Since(start)

	stats := db.Stats()
	poolStats := PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}

	status := HealthStatus{
		Healthy:   err == nil,
		Latency:   latency,
		PoolStats: poolStats,
	}

	if err != nil {
		status.Error = err.Error()
	}

	return status
}

// IsHealthy returns true if the database is reachable
func (db *DBKit) IsHealthy(ctx context.Context) bool {
	return db.Ping(ctx) == nil
}

// PoolStatsFromSQL converts sql.DBStats to PoolStats
func PoolStatsFromSQL(stats sql.DBStats) PoolStats {
	return PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}
