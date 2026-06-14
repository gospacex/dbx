package tracing_test

import (
	"testing"

	"github.com/gospacex/dbx/observability/tracing"
)

func TestKafkaExporter_ExportSpans(t *testing.T) {
	exp := tracing.NewKafkaExporter(nil, "test-topic")
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}

func TestRedisExporter_ExportSpans(t *testing.T) {
	exp := tracing.NewRedisExporter(nil, "test-stream")
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}

func TestMQXSpanExporter_Defaults(t *testing.T) {
	exp := tracing.NewMQXSpanExporter(nil, nil, nil)
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}
