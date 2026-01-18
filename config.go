// Package dbkit provides a consistent database layer for Go applications.
// It wraps Bun ORM with connection pooling, migrations, transactions,
// generic CRUD helpers, rich error handling, and configurable observability.
package dbkit

import (
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// Config holds database configuration
type Config struct {
	// Connection
	URL string // PostgreSQL connection string (required)

	// Pool settings
	MaxOpenConns    int           // Max open connections (default: 25)
	MaxIdleConns    int           // Max idle connections (default: 5)
	ConnMaxLifetime time.Duration // Max connection lifetime (default: 5m)
	ConnMaxIdleTime time.Duration // Max idle time (default: 1m)

	// Timeouts
	DialTimeout  time.Duration // Connection dial timeout (default: 5s)
	ReadTimeout  time.Duration // Read timeout (default: 30s)
	WriteTimeout time.Duration // Write timeout (default: 30s)

	// Observability (all optional)
	Logger          *slog.Logger          // Structured logger
	LogQueries      bool                  // Log all queries
	LogSlowQueries  time.Duration         // Log queries slower than this (0 = disabled)
	MetricsRegistry prometheus.Registerer // Prometheus registry for metrics
	Tracer          trace.Tracer          // OpenTelemetry tracer
}

// DefaultConfig returns sensible defaults
func DefaultConfig(url string) Config {
	return Config{
		URL:             url,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
	}
}

// applyDefaults fills in zero values with defaults
func (c *Config) applyDefaults() {
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 25
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 5
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 5 * time.Minute
	}
	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = 1 * time.Minute
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = 5 * time.Second
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = 30 * time.Second
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = 30 * time.Second
	}
}

// WithLogger enables query logging
func (c Config) WithLogger(logger *slog.Logger) Config {
	c.Logger = logger
	c.LogQueries = true
	return c
}

// WithSlowQueryLog logs queries slower than the threshold
func (c Config) WithSlowQueryLog(threshold time.Duration) Config {
	c.LogSlowQueries = threshold
	return c
}

// WithMetrics enables Prometheus metrics
func (c Config) WithMetrics(registry prometheus.Registerer) Config {
	c.MetricsRegistry = registry
	return c
}

// WithTracing enables OpenTelemetry tracing
func (c Config) WithTracing(tracer trace.Tracer) Config {
	c.Tracer = tracer
	return c
}
