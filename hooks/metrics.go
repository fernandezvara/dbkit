package hooks

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/uptrace/bun"
)

// MetricsHook implements Prometheus metrics collection
type MetricsHook struct {
	queryDuration *prometheus.HistogramVec
	queryTotal    *prometheus.CounterVec
	queryErrors   *prometheus.CounterVec
}

// NewMetricsHook creates a new metrics hook and registers collectors
func NewMetricsHook(registry prometheus.Registerer) (*MetricsHook, error) {
	h := &MetricsHook{
		queryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dbkit_query_duration_seconds",
				Help:    "Duration of database queries in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"operation"},
		),
		queryTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dbkit_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation"},
		),
		queryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dbkit_query_errors_total",
				Help: "Total number of database query errors",
			},
			[]string{"operation"},
		),
	}

	// Register metrics
	collectors := []prometheus.Collector{h.queryDuration, h.queryTotal, h.queryErrors}
	for _, c := range collectors {
		if err := registry.Register(c); err != nil {
			// Check if already registered
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return nil, err
			}
		}
	}

	return h, nil
}

// BeforeQuery is called before a query is executed
func (h *MetricsHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery is called after a query is executed
func (h *MetricsHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime).Seconds()
	op := OperationType(event.Query)

	h.queryDuration.WithLabelValues(op).Observe(duration)
	h.queryTotal.WithLabelValues(op).Inc()

	if event.Err != nil {
		h.queryErrors.WithLabelValues(op).Inc()
	}
}
