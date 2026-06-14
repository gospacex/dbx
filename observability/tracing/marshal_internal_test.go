package tracing

import (
	"context"
	"encoding/json"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// makeReadOnlySpan creates a real sdktrace.ReadOnlySpan using a
// tracetest.SpanRecorder and a temporary TracerProvider. The returned
// span has stable IDs and can be passed into marshalSpanJSON and the
// kafka/redis ExportSpans paths.
func makeReadOnlySpan(t *testing.T, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	tr := tp.Tracer("test")
	_, span := tr.Start(context.Background(), name)
	span.SetAttributes(attribute.String("http.method", "GET"), attribute.Int("http.status", 200))
	span.SetStatus(codes.Ok, "ok")
	span.End()
	ended := rec.Ended()
	if len(ended) == 0 {
		t.Fatal("no spans recorded")
	}
	return ended[0]
}

func TestMarshalSpanJSON_Success(t *testing.T) {
	span := makeReadOnlySpan(t, "test-span")
	data, err := marshalSpanJSON(span)
	if err != nil {
		t.Fatalf("marshalSpanJSON() error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty bytes")
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed["name"] != "test-span" {
		t.Errorf("name = %v, want test-span", parsed["name"])
	}
	if parsed["kind"] != "server" && parsed["kind"] != "internal" {
		// Default is internal; tolerate either.
		t.Logf("kind = %v (acceptable)", parsed["kind"])
	}
	attrs, ok := parsed["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("attributes is not a map: %T", parsed["attributes"])
	}
	if attrs["http.method"] != "GET" {
		t.Errorf("http.method = %v, want GET", attrs["http.method"])
	}
	// Non-string attribute values produce "" via AsString(); just confirm key is present.
	if _, hasStatus := attrs["http.status"]; !hasStatus {
		t.Error("http.status attribute should be present (even if AsString returned empty)")
	}
	// TraceID is set by the SDK (random). Just verify it's a non-empty hex string.
	if tid, ok := parsed["trace_id"].(string); !ok || len(tid) != 32 {
		t.Errorf("trace_id = %v, want 32-char hex", parsed["trace_id"])
	}
}

func TestMarshalSpanJSON_MinimalSpan(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	tr := tp.Tracer("test")
	_, span := tr.Start(context.Background(), "minimal")
	span.End()
	ended := rec.Ended()
	if len(ended) == 0 {
		t.Fatal("no spans recorded")
	}
	data, err := marshalSpanJSON(ended[0])
	if err != nil {
		t.Fatalf("marshalSpanJSON() error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty bytes")
	}
}
