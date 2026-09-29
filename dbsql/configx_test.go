package dbsql_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/dbx/dbsql"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/proto"

	collectortracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

const sqlxContractYAML = `
xmiddleware:
  sqlx:
    driver: gorm
    databases:
      default:
        dsn: ":memory:"
        dialect: sqlite
        max_open_conns: 20
        max_idle_conns: 5
        conn_max_lifetime: 5m
        conn_max_idle_time: 3m
        slow_threshold: 200ms
        enable_log: true
        log_level: info
        enable_trace: false
        health_check:
          enabled: true
          interval: 30s
          timeout: 5s
      replica:
        dsn: ":memory:"
`

func readTestConfig(t *testing.T) configx.Config {
	t.Helper()
	return readYAMLConfig(t, sqlxContractYAML)
}

// readYAMLConfig writes body to a temp config.yaml and returns a configx
// handle for it. Callers own the handle; it is closed on test cleanup.
func readYAMLConfig(t *testing.T, body string) configx.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := configx.ReadConfig(configx.ConfigOptions{ConfigFile: path})
	if err != nil {
		t.Fatalf("configx.ReadConfig error: %v", err)
	}
	t.Cleanup(func() { _ = cfg.Close() })
	return cfg
}

// sqlxTraceYAML is the requirement-3 fixture: a traced instance whose
// tracing sub-section points at the in-process collector (%s = host:port).
// The tracing sub-section, not just enable_trace, is what configx carries
// exporter/endpoint/sampler details in.
const sqlxTraceYAML = `
xmiddleware:
  sqlx:
    driver: gorm
    databases:
      traced:
        dsn: "file:dbx_configx_trace?mode=memory&cache=shared"
        dialect: sqlite
        max_idle_conns: 1
        enable_trace: true
        tracing:
          enabled: true
          service: dbx-configx-test
          exporter: jaeger
          endpoint: "%s"
`

// traceRow is the minimal entity the tracing test migrates and mutates, so
// the orm instrumentation sees one statement of each instrumented kind.
type traceRow struct {
	ID   uint `gorm:"primaryKey;autoIncrement"`
	Name string
}

func (traceRow) TableName() string { return "trace_rows" }

// otlpCollector is an in-process OTLP/HTTP trace receiver. It decodes every
// exported span so a test can assert on span names and attributes without a
// real Jaeger or collector process.
type otlpCollector struct {
	// hostPort is the bare host:port the tracing config points at, the
	// form newJaegerHTTPExporter takes (WithEndpoint + WithInsecure) —
	// the same shape as "localhost:4318" in the contract.
	hostPort string

	mu       sync.Mutex
	received []*tracepb.ResourceSpans
}

func newOTLPCollector(t *testing.T) *otlpCollector {
	t.Helper()
	c := &otlpCollector{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		req := &collectortracepb.ExportTraceServiceRequest{}
		if err := proto.Unmarshal(body, req); err == nil {
			c.mu.Lock()
			c.received = append(c.received, req.ResourceSpans...)
			c.mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse collector url: %v", err)
	}
	c.hostPort = u.Host
	return c
}

func (c *otlpCollector) spans() []*tracepb.ResourceSpans {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.received
}

func (c *otlpCollector) spanNames() []string {
	var names []string
	for _, rs := range c.spans() {
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				names = append(names, s.Name)
			}
		}
	}
	return names
}

func (c *otlpCollector) hasSpanName(want string) bool {
	for _, n := range c.spanNames() {
		if n == want {
			return true
		}
	}
	return false
}

// sqlTexts returns the db.query.text / db.statement attributes, which is
// where the executed SQL is expected to land.
func (c *otlpCollector) sqlTexts() []string {
	var out []string
	for _, rs := range c.spans() {
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				for _, attr := range s.Attributes {
					if strings.HasPrefix(attr.GetKey(), "db.query") || attr.GetKey() == "db.statement" {
						out = append(out, attr.GetValue().GetStringValue())
					}
				}
			}
		}
	}
	return out
}

func (c *otlpCollector) hasSQLText(want string) bool {
	for _, s := range c.sqlTexts() {
		if strings.Contains(s, want) {
			return true
		}
	}
	return false
}

func TestReadConfigFromConfigx_AlignedWithContract(t *testing.T) {
	cfg := readTestConfig(t)
	sc := dbsql.ReadConfigFromConfigx(cfg, "")
	if sc.Driver != "gorm" {
		t.Errorf("Driver = %q, want gorm", sc.Driver)
	}
	if sc.Dsn != ":memory:" || sc.Dialect != "sqlite" {
		t.Errorf("Dsn/Dialect = %q/%q", sc.Dsn, sc.Dialect)
	}
	if sc.MaxOpenConns != 20 || sc.MaxIdleConns != 5 {
		t.Errorf("pool = %d/%d, want 20/5", sc.MaxOpenConns, sc.MaxIdleConns)
	}
	if sc.ConnMaxLifetime != 5*time.Minute || sc.ConnMaxIdleTime != 3*time.Minute {
		t.Errorf("durations = %v/%v", sc.ConnMaxLifetime, sc.ConnMaxIdleTime)
	}
	if sc.SlowThreshold != 200*time.Millisecond {
		t.Errorf("SlowThreshold = %v", sc.SlowThreshold)
	}
	if !sc.EnableLog || sc.LogLevel != "info" {
		t.Errorf("log = %v/%q", sc.EnableLog, sc.LogLevel)
	}
	if sc.EnableTrace {
		t.Error("EnableTrace should be false")
	}
	if !sc.HealthCheck.Enabled || sc.HealthCheck.Interval != 30*time.Second || sc.HealthCheck.Timeout != 5*time.Second {
		t.Errorf("HealthCheck = %+v", sc.HealthCheck)
	}
}

func TestOpenFromConfigx_SQLite(t *testing.T) {
	cfg := readTestConfig(t)
	db, err := dbsql.OpenFromConfigx(cfg, "default", nil)
	if err != nil {
		t.Fatalf("OpenFromConfigx error: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping error: %v", err)
	}
	if got := sqlDB.Stats().MaxOpenConnections; got != 20 {
		t.Errorf("MaxOpenConnections = %d, want 20", got)
	}
}

func TestOpenFromConfigx_DialectDetection(t *testing.T) {
	cfg := readTestConfig(t)
	// "replica" has no explicit dialect; ":memory:" should detect as sqlite.
	db, err := dbsql.OpenFromConfigx(cfg, "replica", nil)
	if err != nil {
		t.Fatalf("OpenFromConfigx(replica) error: %v", err)
	}
	if db.Dialector.Name() != "sqlite" {
		t.Errorf("dialect = %q, want sqlite", db.Dialector.Name())
	}
}

func TestOpenFromConfigx_MissingDSN(t *testing.T) {
	cfg := readTestConfig(t)
	if _, err := dbsql.OpenFromConfigx(cfg, "nonexistent", nil); err == nil {
		t.Fatal("expected error for missing dsn")
	}
}

// TestOpenFromConfigx_TraceReachesCollector is the requirement-3 regression
// test. It asserts the whole configx path end to end: enable_trace + the
// tracing sub-section must (a) provision a real TracerProvider — not the
// noop global — and (b) make the SQL statements issued through the returned
// handle arrive at an OTLP receiver with db.query.text attached.
//
// The failure this guards against is silent: without a provisioned provider
// every span goes to the noop tracer, CRUD still succeeds, and the only
// symptom is an empty trace in the collector. e2e/jaeger_e2e_test.go covers
// the same behaviour through the file-config path (dbsql.Open), so the
// configx entry point needs its own proof.
//
// The DSN uses a shared-cache in-memory database and the fixture pins
// max_idle_conns: 1. Both are required: a bare ":memory:" gives every pooled
// connection a private, empty database, and a shared-cache memory database is
// discarded as soon as its last connection closes — with the contract's
// default max_idle_conns of 0 that happens after every single statement.
func TestOpenFromConfigx_TraceReachesCollector(t *testing.T) {
	ctx := context.Background()
	// ExtractTracingAndApply is first-init-wins process-wide; clear any
	// provider a previously-run test left behind so the assertions below
	// observe this test's own configuration.
	if err := dbsql.ShutdownTracerProvider(ctx); err != nil {
		t.Fatalf("ShutdownTracerProvider(reset) error: %v", err)
	}
	// Cleanup, not Defer: a t.Fatalf below would otherwise leave the
	// batch processor holding spans for a collector that is already closed.
	t.Cleanup(func() { _ = dbsql.ShutdownTracerProvider(ctx) })

	rec := newOTLPCollector(t)
	cfg := readYAMLConfig(t, fmt.Sprintf(sqlxTraceYAML, rec.hostPort))
	db, err := dbsql.OpenFromConfigx(cfg, "traced", nil)
	if err != nil {
		t.Fatalf("OpenFromConfigx error: %v", err)
	}
	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Fatal("no SDK TracerProvider was registered: enable_trace left the global provider as noop")
	}

	// CRUD through the handle — one statement of each kind the orm
	// instrumentation labels db.create / db.query / db.update / db.delete.
	if err := db.AutoMigrate(&traceRow{}); err != nil {
		t.Fatalf("AutoMigrate error: %v", err)
	}
	row := &traceRow{Name: "configx"}
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("Create error: %v", err)
	}
	var got traceRow
	if err := db.First(&got, row.ID).Error; err != nil {
		t.Fatalf("First error: %v", err)
	}
	if err := db.Model(&traceRow{}).Where("id = ?", got.ID).Update("name", "updated").Error; err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if err := db.Delete(&traceRow{}, got.ID).Error; err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	// Shutdown flushes the BatchSpanProcessor; without it the spans are
	// still queued and the collector legitimately has nothing yet.
	if err := dbsql.ShutdownTracerProvider(ctx); err != nil {
		t.Fatalf("ShutdownTracerProvider(flush) error: %v", err)
	}

	spans := rec.spans()
	if len(spans) == 0 {
		t.Fatal("collector received no spans: SQL never entered the call chain")
	}
	for _, want := range []string{"db.create", "db.query", "db.update", "db.delete"} {
		if !rec.hasSpanName(want) {
			t.Errorf("missing span %q; got %v", want, rec.spanNames())
		}
	}
	if !rec.hasSQLText("INSERT INTO") || !rec.hasSQLText("UPDATE") {
		t.Errorf("db.query.text not carried into the chain; got statements %v", rec.sqlTexts())
	}
}

// TestOpenFromConfigx_TraceOff_RegistersNoProvider pins the other half of
// the switch: with no enable_trace in the config, OpenFromConfigx must leave
// the global provider exactly as it found it, so a host application's own
// TracerProvider is not displaced.
func TestOpenFromConfigx_TraceOff_RegistersNoProvider(t *testing.T) {
	ctx := context.Background()
	if err := dbsql.ShutdownTracerProvider(ctx); err != nil {
		t.Fatalf("ShutdownTracerProvider(reset) error: %v", err)
	}
	t.Cleanup(func() { _ = dbsql.ShutdownTracerProvider(ctx) })

	// Identity, not type: otel keeps the last SetTracerProvider value
	// forever, so a type assertion would still see the SDK provider a
	// previously-run test installed.
	before := otel.GetTracerProvider()

	cfg := readTestConfig(t)
	if _, err := dbsql.OpenFromConfigx(cfg, "replica", nil); err != nil {
		t.Fatalf("OpenFromConfigx error: %v", err)
	}
	if after := otel.GetTracerProvider(); after != before {
		t.Errorf("global TracerProvider changed although enable_trace is absent: %T -> %T", before, after)
	}
}

// TestOpenFromConfigx_UnknownDSNScheme asserts an unrecognised URL-scheme
// DSN fails loudly. It previously fell through to the sqlite branch and
// surfaced as a driver-level "unable to open database file", hiding the
// real problem (a missing or misspelled dialect).
func TestOpenFromConfigx_UnknownDSNScheme(t *testing.T) {
	cfg := readYAMLConfig(t, `
xmiddleware:
  sqlx:
    databases:
      odd:
        dsn: "clickhouse://user:secret@clickhouse:9000/db"
`)
	_, err := dbsql.OpenFromConfigx(cfg, "odd", nil)
	if err == nil {
		t.Fatal("expected an error for an unrecognized dsn scheme")
	}
	if !strings.Contains(err.Error(), "clickhouse://") {
		t.Errorf("error should name the offending scheme, got: %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("error leaked the dsn password: %v", err)
	}
}

// TestOpenFromConfigx_PostgresURLDSN covers the DSN shape the switch above
// used to misroute: URL-style postgres must detect as postgres, not sqlite.
func TestOpenFromConfigx_PostgresURLDSN(t *testing.T) {
	cfg := readYAMLConfig(t, `
xmiddleware:
  sqlx:
    databases:
      pg:
        dsn: "postgres://user:pass@127.0.0.1:1/db?sslmode=disable"
`)
	_, err := dbsql.OpenFromConfigx(cfg, "pg", nil)
	if err == nil {
		t.Fatal("expected a connection error, proving the postgres dialect was selected")
	}
	if strings.Contains(err.Error(), "unable to open database file") {
		t.Errorf("postgres URL dsn was misrouted to sqlite: %v", err)
	}
}

// TestReadConfigFromConfigx_RealContractFile reads the actual
// configx/configs/config.simple.yaml shipped with configx and asserts dbx
// parses every xmiddleware.sqlx.databases.default field the contract
// declares. This is the two-sided alignment proof: if configx renames or
// re-types a contract key, this test breaks.
func TestReadConfigFromConfigx_RealContractFile(t *testing.T) {
	cfg, err := configx.ReadConfig(configx.ConfigOptions{
		ConfigFile: "../../configx/configs/config.simple.yaml",
	})
	if err != nil {
		t.Fatalf("configx.ReadConfig error: %v", err)
	}
	t.Cleanup(func() { _ = cfg.Close() })

	sc := dbsql.ReadConfigFromConfigx(cfg, "default")
	if sc.Driver != "gorm" {
		t.Errorf("Driver = %q, want gorm", sc.Driver)
	}
	if sc.Dsn == "" {
		t.Error("Dsn should be read from contract file")
	}
	if sc.MaxOpenConns != 100 || sc.MaxIdleConns != 25 {
		t.Errorf("pool = %d/%d, want 100/25 per config.simple.yaml", sc.MaxOpenConns, sc.MaxIdleConns)
	}
	if sc.ConnMaxLifetime != 5*time.Minute || sc.ConnMaxIdleTime != 3*time.Minute {
		t.Errorf("durations = %v/%v, want 5m/3m", sc.ConnMaxLifetime, sc.ConnMaxIdleTime)
	}
	if sc.SlowThreshold != 200*time.Millisecond {
		t.Errorf("SlowThreshold = %v, want 200ms", sc.SlowThreshold)
	}
	if !sc.EnableLog || sc.LogLevel != "debug" {
		t.Errorf("log = %v/%q, want true/debug", sc.EnableLog, sc.LogLevel)
	}
	if sc.EnableTrace {
		t.Error("EnableTrace = true, want false per config.simple.yaml")
	}
}
