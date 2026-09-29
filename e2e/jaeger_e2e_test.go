//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/dbsql"
	"gorm.io/gorm"
)

// jaegerTraces is the subset of the Jaeger HTTP query API response we assert on.
type jaegerTraces struct {
	Data []struct {
		Spans []struct {
			OperationName string `json:"operationName"`
			Tags          []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"tags"`
		} `json:"spans"`
	} `json:"data"`
}

// assertSQLStatementInChain is the shared body of the two Jaeger chain tests.
// It runs real CRUD through the handle returned by open, flushes the spans,
// then polls the Jaeger API until a span carries db.query.text containing the
// per-run table marker.
//
// Skips when Jaeger is unreachable so the suite stays green on machines
// without the container. DBX_E2E_JAEGER_OTLP / DBX_E2E_JAEGER_QUERY repoint
// it at a non-default port map.
func assertSQLStatementInChain(t *testing.T, service string, open func(t *testing.T, table string) *gorm.DB) {
	t.Helper()
	if !jaegerReachable(t) {
		t.Skipf("Jaeger not reachable at %s; start the container or set DBX_E2E_JAEGER_QUERY", jaegerQueryBase())
	}

	// GORM emits prepared SQL into span attributes (db.query.text carries
	// "WHERE name = ?", not the bound value), so the per-run unique marker
	// must live in the table name to be matchable in the statement text.
	table := fmt.Sprintf("chain_%d", time.Now().UnixNano())
	db := open(t, table)

	type chainItem struct {
		ID   uint `gorm:"primarykey"`
		Name string
	}
	if err := db.Table(table).AutoMigrate(&chainItem{}); err != nil {
		t.Fatalf("AutoMigrate error: %v", err)
	}
	if err := db.Table(table).Create(&chainItem{Name: "x"}).Error; err != nil {
		t.Fatalf("Create error: %v", err)
	}
	var items []chainItem
	if err := db.Table(table).Find(&items).Error; err != nil {
		t.Fatalf("Find error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("found %d items, want 1", len(items))
	}

	// Flush spans to Jaeger before querying the API.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := dbsql.ShutdownTracerProvider(ctx); err != nil {
		t.Fatalf("ShutdownTracerProvider error: %v", err)
	}

	// Jaeger indexing is asynchronous; poll until the table marker shows up.
	deadline := time.Now().Add(15 * time.Second)
	for {
		found, err := jaegerHasSQLTag(service, table)
		if err != nil {
			t.Fatalf("jaeger query error: %v", err)
		}
		if found {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("no span with db.query.text containing the marker appeared in Jaeger within 15s")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// TestE2E_Tracing_Jaeger_SQLStatementInChain proves review requirement:
// with the tracing switch on, executing a SQL operation through dbx puts
// the executed statement into the call chain. It runs real CRUD against
// SQLite, exports spans over OTLP/HTTP to a docker Jaeger all-in-one
// (jaeger-local: 4318 receive / 16686 query, both overridable), then queries
// the Jaeger API and asserts a span carries db.query.text containing our
// unique marker. This is the file-config entry point, dbsql.Open.
func TestE2E_Tracing_Jaeger_SQLStatementInChain(t *testing.T) {
	assertSQLStatementInChain(t, "dbx-e2e-jaeger", func(t *testing.T, _ string) *gorm.DB {
		t.Helper()
		cfg := &config.SQLiteConfig{
			Path: filepath.Join(t.TempDir(), "e2e.db"),
			Tracing: &config.TracingConfig{
				Enabled:  true,
				Service:  "dbx-e2e-jaeger",
				Exporter: config.ExporterJaeger,
				Endpoint: jaegerOTLPEndpoint(),
			},
		}
		db, err := dbsql.Open(cfg)
		if err != nil {
			t.Fatalf("dbsql.Open error: %v", err)
		}
		return db
	})
}

// TestE2E_Tracing_Jaeger_ConfigxSQLStatementInChain is the same proof for the
// configx entry point, dbsql.OpenFromConfigx.
//
// It exists because the file-config path was already covered and the configx
// path was not: OpenFromConfigx read enable_trace and installed the tracer
// callback without ever creating a TracerProvider, so a configx user got zero
// spans while the dbsql.Open test stayed green. A unit test with an in-process
// collector (TestOpenFromConfigx_TraceReachesCollector) pins the regression;
// this test keeps the evidence honest against a real collector.
func TestE2E_Tracing_Jaeger_ConfigxSQLStatementInChain(t *testing.T) {
	const service = "dbx-e2e-jaeger-configx"
	assertSQLStatementInChain(t, service, func(t *testing.T, _ string) *gorm.DB {
		t.Helper()
		cfg := readYAMLConfigFile(t, fmt.Sprintf(`
xmiddleware:
  sqlx:
    driver: gorm
    databases:
      traced:
        dsn: %q
        dialect: sqlite
        enable_trace: true
        tracing:
          enabled: true
          service: %s
          exporter: %s
          endpoint: %s
`, filepath.Join(t.TempDir(), "e2e.db"), service, config.ExporterJaeger, jaegerOTLPEndpoint()))
		db, err := dbsql.OpenFromConfigx(cfg, "traced", nil)
		if err != nil {
			t.Fatalf("dbsql.OpenFromConfigx error: %v", err)
		}
		return db
	})
}

func jaegerReachable(t *testing.T) bool {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(jaegerQueryBase() + "/api/services")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// jaegerHasSQLTag reports whether any span of the service carries a
// db.query.text tag whose value contains marker.
func jaegerHasSQLTag(service, marker string) (bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s/api/traces?service=%s&limit=100", jaegerQueryBase(), service)
	resp, err := client.Get(url)
	if err != nil {
		return false, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	var traces jaegerTraces
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		return false, fmt.Errorf("decode traces: %w", err)
	}
	for _, tr := range traces.Data {
		for _, sp := range tr.Spans {
			for _, tag := range sp.Tags {
				if tag.Key == "db.query.text" && strings.Contains(tag.Value, marker) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
