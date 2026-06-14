package tracing

import (
	"context"
	"errors"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type fakeJaegerExporter struct {
	shutdownCalls int
	exportCalls   int
	exportErr     error
	shutdownErr   error
}

func (f *fakeJaegerExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error {
	f.exportCalls++
	return f.exportErr
}

func (f *fakeJaegerExporter) Shutdown(_ context.Context) error {
	f.shutdownCalls++
	return f.shutdownErr
}

var _ sdktrace.SpanExporter = (*fakeJaegerExporter)(nil)

func TestMQXSpanExporter_ExportSpans_AllNil(t *testing.T) {
	exp := NewMQXSpanExporter(nil, nil, nil)
	err := exp.ExportSpans(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when all exporters nil")
	}
	if !errors.Is(err, err) { // sanity: error path was hit
		t.Fatal("error not produced")
	}
}

func TestMQXSpanExporter_ExportSpans_KafkaPriority(t *testing.T) {
	kp := &fakeKafkaProducer{}
	jaeger := &fakeJaegerExporter{}
	mqx := NewMQXSpanExporter(jaeger, &KafkaExporter{producer: kp, topic: "t"}, nil)
	if err := mqx.ExportSpans(context.Background(), nil); err != nil {
		t.Fatalf("ExportSpans(kafka path) error: %v", err)
	}
	if kp.calls != 0 {
		t.Errorf("no spans = no Produce calls, got %d", kp.calls)
	}
	if jaeger.exportCalls != 0 {
		t.Errorf("jaeger should not be called when kafka is set, got %d", jaeger.exportCalls)
	}
}

func TestMQXSpanExporter_ExportSpans_RedisPriority(t *testing.T) {
	rc := &fakeRedisClient{}
	jaeger := &fakeJaegerExporter{}
	mqx := NewMQXSpanExporter(jaeger, nil, &RedisExporter{client: rc, stream: "s"})
	if err := mqx.ExportSpans(context.Background(), nil); err != nil {
		t.Fatalf("ExportSpans(redis path) error: %v", err)
	}
	if rc.calls != 0 {
		t.Errorf("no spans = no XAdd calls, got %d", rc.calls)
	}
	if jaeger.exportCalls != 0 {
		t.Errorf("jaeger should not be called when redis is set, got %d", jaeger.exportCalls)
	}
}

func TestMQXSpanExporter_ExportSpans_JaegerOnly(t *testing.T) {
	jaeger := &fakeJaegerExporter{}
	mqx := NewMQXSpanExporter(jaeger, nil, nil)
	if err := mqx.ExportSpans(context.Background(), nil); err != nil {
		t.Fatalf("ExportSpans(jaeger path) error: %v", err)
	}
	if jaeger.exportCalls != 1 {
		t.Errorf("jaeger.ExportSpans calls = %d, want 1", jaeger.exportCalls)
	}
}

func TestMQXSpanExporter_ExportSpans_JaegerError(t *testing.T) {
	want := errors.New("jaeger down")
	jaeger := &fakeJaegerExporter{exportErr: want}
	mqx := NewMQXSpanExporter(jaeger, nil, nil)
	err := mqx.ExportSpans(context.Background(), nil)
	if !errors.Is(err, want) {
		t.Errorf("expected wrapped jaeger error, got: %v", err)
	}
}

func TestMQXSpanExporter_Shutdown_JaegerPresent(t *testing.T) {
	jaeger := &fakeJaegerExporter{}
	mqx := NewMQXSpanExporter(jaeger, nil, nil)
	if err := mqx.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown error: %v", err)
	}
	if jaeger.shutdownCalls != 1 {
		t.Errorf("jaeger.Shutdown calls = %d, want 1", jaeger.shutdownCalls)
	}
}

func TestMQXSpanExporter_Shutdown_JaegerAbsent(t *testing.T) {
	mqx := NewMQXSpanExporter(nil, nil, nil)
	if err := mqx.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown with no jaeger should be nil, got: %v", err)
	}
}
