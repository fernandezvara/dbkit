package hooks

import (
	"context"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracingHook implements OpenTelemetry tracing
type TracingHook struct {
	tracer trace.Tracer
}

// NewTracingHook creates a new tracing hook
func NewTracingHook(tracer trace.Tracer) *TracingHook {
	return &TracingHook{tracer: tracer}
}

type spanCtxKey struct{}

// BeforeQuery is called before a query is executed
func (h *TracingHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	if h.tracer == nil {
		return ctx
	}

	op := OperationType(event.Query)

	ctx, span := h.tracer.Start(ctx, "db."+op,
		trace.WithSpanKind(trace.SpanKindClient),
	)

	return context.WithValue(ctx, spanCtxKey{}, span)
}

// AfterQuery is called after a query is executed
func (h *TracingHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	spanVal := ctx.Value(spanCtxKey{})
	if spanVal == nil {
		return
	}

	span, ok := spanVal.(trace.Span)
	if !ok {
		return
	}
	defer span.End()

	query := event.Query
	if len(query) > 500 {
		query = query[:500] + "..."
	}

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", query),
		attribute.String("db.operation", OperationType(event.Query)),
	)

	if event.Err != nil {
		span.RecordError(event.Err)
		span.SetStatus(codes.Error, event.Err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
}
