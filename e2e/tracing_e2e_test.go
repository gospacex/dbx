//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/dbsql"
)

// TestE2E_Tracing_Kafka spins up a real Kafka producer via kafkax.POS and
// routes one span through the MQXSpanExporter. If the broker is reachable the
// test passes — Kafka writes are fire-and-forget at the protocol level so we
// don't inspect a particular offset, we just confirm the call path works.
func TestE2E_Tracing_Kafka(t *testing.T) {
	tc := &config.TracingConfig{
		Enabled:  true,
		Exporter: config.ExporterKafka,
		Endpoint: "127.0.0.1:9092",
		Topic:    "otel-traces-e2e",
	}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	exp, err := dbsql.CreateExporter(context.Background(), tc)
	if err != nil {
		t.Fatalf("CreateExporter(kafka) error: %v", err)
	}
	if exp == nil {
		t.Fatal("exporter is nil")
	}

	tp := newTracerProvider(t, exp)
	tr := tp.Tracer("e2e")
	_, span := tr.Start(context.Background(), "kafka-export-test")
	span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tp.Shutdown(ctx); err != nil {
		t.Fatalf("TracerProvider.Shutdown error: %v", err)
	}
}

// TestE2E_Tracing_RedisStream spins up a real Redis client via redisx.POS
// and writes one span to a Redis Stream. If Redis is reachable the test
// passes — the stream entry is consumed by the test if you want to inspect
// it with `redis-cli XRANGE otel-traces-e2e - +`.
func TestE2E_Tracing_RedisStream(t *testing.T) {
	tc := &config.TracingConfig{
		Enabled:       true,
		Exporter:      config.ExporterRedisStream,
		Endpoint:      "127.0.0.1:6379",
		Stream:        "otel-traces-e2e",
		RedisPassword: "redis123456",
	}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	exp, err := dbsql.CreateExporter(context.Background(), tc)
	if err != nil {
		t.Fatalf("CreateExporter(redis) error: %v", err)
	}
	if exp == nil {
		t.Fatal("exporter is nil")
	}

	tp := newTracerProvider(t, exp)
	tr := tp.Tracer("e2e")
	_, span := tr.Start(context.Background(), "redis-stream-export-test")
	span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tp.Shutdown(ctx); err != nil {
		t.Fatalf("TracerProvider.Shutdown error: %v", err)
	}
}

// TestE2E_Tracing_Defaults ensures that an empty TracingConfig (no exporter
// set) defaults to jaeger, and that the ExporterValidator rejects garbage.
func TestE2E_Tracing_Defaults(t *testing.T) {
	tc := &config.TracingConfig{Enabled: true}
	if err := tc.Validate(); err != nil {
		t.Fatalf("default Validate error: %v", err)
	}
	if tc.Exporter != config.ExporterJaeger {
		t.Errorf("default Exporter = %q, want %q", tc.Exporter, config.ExporterJaeger)
	}

	bad := &config.TracingConfig{Enabled: true, Exporter: "prometheus"}
	if err := bad.Validate(); err == nil {
		t.Error("Validate(unknown exporter) should fail")
	}
}
