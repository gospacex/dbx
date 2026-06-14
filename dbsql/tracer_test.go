package dbsql_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/dbsql"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestCreateExporter_Jaeger(t *testing.T) {
	tc := &config.TracingConfig{Exporter: "jaeger", Endpoint: "localhost:4318"}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	exp, err := dbsql.CreateExporter(context.Background(), tc)
	if err != nil {
		t.Fatalf("CreateExporter(jaeger) error: %v", err)
	}
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}

func TestCreateExporter_Invalid(t *testing.T) {
	tc := &config.TracingConfig{Exporter: "unsupported"}
	_, err := dbsql.CreateExporter(context.Background(), tc)
	if err == nil {
		t.Fatal("expected error for unsupported exporter")
	}
}

func TestCreateExporter_Kafka(t *testing.T) {
	tc := &config.TracingConfig{Exporter: "kafka"}
	exp, err := dbsql.CreateExporter(context.Background(), tc)
	if err != nil {
		t.Fatalf("CreateExporter(kafka) error: %v", err)
	}
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}

func TestCreateExporter_RedisStream(t *testing.T) {
	tc := &config.TracingConfig{Exporter: "redis_stream", Endpoint: "localhost:6379"}
	_, err := dbsql.CreateExporter(context.Background(), tc)
	// Either an error from PING (no real redis) is expected; we just want to
	// exercise the code path.
	_ = err
}

func TestCreateExporter_Default(t *testing.T) {
	// Empty exporter should resolve to jaeger (default in Validate), then succeed.
	tc := &config.TracingConfig{}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	exp, err := dbsql.CreateExporter(context.Background(), tc)
	if err != nil {
		t.Fatalf("CreateExporter(default) error: %v", err)
	}
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}

func TestCreateExporter_Kafka_WithAuth(t *testing.T) {
	tc := &config.TracingConfig{
		Exporter:              "kafka",
		Endpoint:              "broker1:9092",
		Topic:                 "traces",
		KafkaUsername:         "user",
		KafkaPassword:         "pass",
		KafkaSecurityProtocol: "SASL_SSL",
		KafkaSASLMechanism:    "PLAIN",
	}
	exp, err := dbsql.CreateExporter(context.Background(), tc)
	if err != nil {
		t.Fatalf("CreateExporter(kafka+auth) error: %v", err)
	}
	if exp == nil {
		t.Fatal("exporter should not be nil")
	}
}

func TestExtractTracingAndApply_Enabled_RedisStream(t *testing.T) {
	tc := &config.TracingConfig{Enabled: true, Exporter: "redis_stream", Endpoint: "localhost:6379"}
	_ = dbsql.ExtractTracingAndApply(context.Background(), tc)
	// Result is environment-dependent; just ensure the call doesn't panic.
}

func TestKafkaConfigFromTracing(t *testing.T) {
	tc := &config.TracingConfig{
		Endpoint: "broker1:9092,broker2:9092",
		Topic:    "my-traces",
	}
	brokers, topic := dbsql.KafkaConfigFromTracing(tc)
	if len(brokers) != 2 {
		t.Fatalf("expected 2 brokers, got %d", len(brokers))
	}
	if brokers[0] != "broker1:9092" {
		t.Errorf("broker[0] = %q, want broker1:9092", brokers[0])
	}
	if topic != "my-traces" {
		t.Errorf("topic = %q, want my-traces", topic)
	}
}

func TestRedisConfigFromTracing(t *testing.T) {
	tc := &config.TracingConfig{
		Endpoint:      "localhost:6379",
		RedisPassword: "secret",
	}
	addr, password := dbsql.RedisConfigFromTracing(tc)
	if addr != "localhost:6379" {
		t.Errorf("addr = %q, want localhost:6379", addr)
	}
	if password != "secret" {
		t.Errorf("password = %q, want secret", password)
	}
}

// TestCreateExporter_Jaeger_PlainHTTP is the regression test for the
// "CRUD SQL not reaching Jaeger" bug. The otlptracehttp client defaults
// to TLS; if the exporter is built without WithInsecure(), the
// BatchSpanProcessor will fail the TLS handshake against a plain HTTP
// Jaeger receiver (the default for jaeger-all-in-one on :4318) and
// silently drop every span — including the GORM CRUD spans that the
// orm/instrumentation package is producing.
//
// We start an in-process httptest server, point the exporter at it
// using a bare host:port endpoint (the exact path the example yaml
// takes: `endpoint: localhost:4318` → WithEndpoint + WithInsecure),
// fire one real span through ExportSpans, and assert the test server
// received the POST. Before the fix this test times out or returns
// the TLS handshake error from ExportSpans; after the fix it passes
// in milliseconds.
func TestCreateExporter_Jaeger_PlainHTTP(t *testing.T) {
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("parse test server url: %v", err)
	}
	// Bare host:port form (no scheme) — the same form the example's
	// `endpoint: localhost:4318` line takes through
	// newJaegerHTTPExporter.
	hostPort := u.Host

	tc := &config.TracingConfig{Exporter: "jaeger", Endpoint: hostPort}
	exp, err := dbsql.CreateExporter(context.Background(), tc)
	if err != nil {
		t.Fatalf("CreateExporter(jaeger) error: %v", err)
	}
	defer func() { _ = exp.Shutdown(context.Background()) }()

	// Build one real ReadOnlySpan via tracetest — this is the same
	// type the BatchSpanProcessor hands to ExportSpans, so the
	// exercise is end-to-end.
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	defer func() { _ = tp.Shutdown(context.Background()) }()
	_, span := tp.Tracer("test").Start(context.Background(), "regression")
	span.End()
	spans := rec.Ended()
	if len(spans) == 0 {
		t.Fatal("recorder produced no spans")
	}

	if err := exp.ExportSpans(context.Background(), spans); err != nil {
		t.Fatalf("ExportSpans returned an error — likely TLS handshake against plain HTTP receiver: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got == 0 {
		t.Fatal("OTLP receiver got no POST; the jaeger exporter is not actually reaching the endpoint")
	}
}

// TestCreateExporter_Jaeger_HTTPSchemeURL covers the explicit-scheme
// path: when the user writes `endpoint: http://collector:4318` we
// route through WithEndpointURL. This is the path production users
// would take when they need to switch to TLS or use a non-default
// path, so it must also exercise plain HTTP correctly.
func TestCreateExporter_Jaeger_HTTPSchemeURL(t *testing.T) {
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	tc := &config.TracingConfig{Exporter: "jaeger", Endpoint: ts.URL}
	exp, err := dbsql.CreateExporter(context.Background(), tc)
	if err != nil {
		t.Fatalf("CreateExporter(jaeger) error: %v", err)
	}
	defer func() { _ = exp.Shutdown(context.Background()) }()

	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	defer func() { _ = tp.Shutdown(context.Background()) }()
	_, span := tp.Tracer("test").Start(context.Background(), "regression-http-url")
	span.End()

	if err := exp.ExportSpans(context.Background(), rec.Ended()); err != nil {
		t.Fatalf("ExportSpans returned an error: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got == 0 {
		t.Fatal("OTLP receiver got no POST via http:// URL form")
	}
}
