package tracing

import (
	"context"
	"fmt"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// MQXSpanExporter dispatches spans to jaeger, kafka, or redis based on
// whichever sub-exporter is non-nil. Priority: kafka > redis > jaeger.
type MQXSpanExporter struct {
	jaeger sdktrace.SpanExporter
	kafka  *KafkaExporter
	redis  *RedisExporter
}

// NewMQXSpanExporter creates a dispatcher that routes spans to the first
// non-nil exporter in priority order: kafka, redis, jaeger.
func NewMQXSpanExporter(jaeger sdktrace.SpanExporter, kafka *KafkaExporter, redis *RedisExporter) *MQXSpanExporter {
	return &MQXSpanExporter{jaeger: jaeger, kafka: kafka, redis: redis}
}

// ExportSpans sends spans to the highest-priority configured exporter.
func (e *MQXSpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	switch {
	case e.kafka != nil:
		return e.kafka.ExportSpans(ctx, spans)
	case e.redis != nil:
		return e.redis.ExportSpans(ctx, spans)
	case e.jaeger != nil:
		return e.jaeger.ExportSpans(ctx, spans)
	default:
		return fmt.Errorf("mqxSpanExporter: no exporter configured")
	}
}

// Shutdown shuts down the underlying jaeger exporter if present.
// Kafka and Redis exporters delegate lifecycle management to mqx.
func (e *MQXSpanExporter) Shutdown(ctx context.Context) error {
	if e.jaeger != nil {
		return e.jaeger.Shutdown(ctx)
	}
	return nil
}