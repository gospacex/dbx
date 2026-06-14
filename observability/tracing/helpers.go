package tracing

import (
	"encoding/json"
	"fmt"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// spanData is a serializable snapshot of a ReadOnlySpan used for JSON export.
type spanData struct {
	Name         string            `json:"name"`
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Kind         string            `json:"kind"`
	StartTime    string            `json:"start_time"`
	EndTime      string            `json:"end_time"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Status       string            `json:"status,omitempty"`
	StatusDesc   string            `json:"status_description,omitempty"`
}

// marshalSpanJSON serializes a ReadOnlySpan to JSON bytes.
func marshalSpanJSON(span sdktrace.ReadOnlySpan) ([]byte, error) {
	attrs := make(map[string]string, span.DroppedAttributes())
	for _, kv := range span.Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}

	data := spanData{
		Name:         span.Name(),
		TraceID:      span.SpanContext().TraceID().String(),
		SpanID:       span.SpanContext().SpanID().String(),
		ParentSpanID: span.Parent().SpanID().String(),
		Kind:         span.SpanKind().String(),
		StartTime:    span.StartTime().Format("2006-01-02T15:04:05.999999999Z"),
		EndTime:      span.EndTime().Format("2006-01-02T15:04:05.999999999Z"),
		Attributes:   attrs,
		Status:       span.Status().Code.String(),
		StatusDesc:   span.Status().Description,
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal span: %w", err)
	}
	return b, nil
}
