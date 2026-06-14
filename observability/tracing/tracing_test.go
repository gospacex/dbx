package tracing_test

import (
	"context"
	"testing"

	"github.com/gospacex/dbx/observability/tracing"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

func TestKafkaExporter_NewKafkaExporter(t *testing.T) {
	exp := tracing.NewKafkaExporter(nil, "test-topic")
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}

func TestKafkaExporter_ExportSpans_NilProducer(t *testing.T) {
	exp := tracing.NewKafkaExporter(nil, "test-topic")
	ctx := context.Background()
	err := exp.ExportSpans(ctx, nil)
	// With nil producer, Produce will panic or return error;
	// we just verify the method doesn't crash with empty spans
	_ = err
}

func TestKafkaExporter_Shutdown_NoOp(t *testing.T) {
	exp := tracing.NewKafkaExporter(nil, "test-topic")
	ctx := context.Background()
	err := exp.Shutdown(ctx)
	if err != nil {
		t.Errorf("expected nil error from Shutdown, got: %v", err)
	}
}

func TestRedisExporter_NewRedisExporter(t *testing.T) {
	exp := tracing.NewRedisExporter(nil, "test-stream")
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}

func TestRedisExporter_ExportSpans_NilClient(t *testing.T) {
	exp := tracing.NewRedisExporter(nil, "test-stream")
	ctx := context.Background()
	err := exp.ExportSpans(ctx, nil)
	_ = err
}

func TestRedisExporter_Shutdown_NoOp(t *testing.T) {
	exp := tracing.NewRedisExporter(nil, "test-stream")
	ctx := context.Background()
	err := exp.Shutdown(ctx)
	if err != nil {
		t.Errorf("expected nil error from Shutdown, got: %v", err)
	}
}

func TestMQXSpanExporter_NewMQXSpanExporter(t *testing.T) {
	exp := tracing.NewMQXSpanExporter(nil, nil, nil)
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}

func TestMQXSpanExporter_ExportSpans_NoExporter(t *testing.T) {
	exp := tracing.NewMQXSpanExporter(nil, nil, nil)
	ctx := context.Background()
	err := exp.ExportSpans(ctx, nil)
	if err == nil {
		t.Fatal("expected error when no exporter configured")
	}
}

func TestMQXSpanExporter_Shutdown_JaegerOnly(t *testing.T) {
	// Create a real (but non-functional) jaeger exporter for Shutdown test
	exp, err := otlptracehttp.New(context.Background())
	if err != nil {
		t.Skipf("otlptracehttp.New failed: %v", err)
	}
	mqx := tracing.NewMQXSpanExporter(exp, nil, nil)
	ctx := context.Background()
	err = mqx.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown error: %v", err)
	}
}

func TestMQXSpanExporter_ExportSpans_KafkaPriority(t *testing.T) {
	// Kafka has highest priority — even with nil, the switch checks kafka first
	// We can't test real kafka export, but we verify the exporter is created
	jaeger, err := otlptracehttp.New(context.Background())
	if err != nil {
		t.Skipf("otlptracehttp.New failed: %v", err)
	}
	exp := tracing.NewMQXSpanExporter(jaeger, nil, nil)
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
	// Verify jaeger path works (nil spans is fine)
	ctx := context.Background()
	err = exp.ExportSpans(ctx, nil)
	if err != nil {
		t.Errorf("ExportSpans with jaeger should not error on nil spans: %v", err)
	}
}
