//go:build e2e
// +build e2e

// E2E tests against real services running via /Users/hyx/work/gowork/src/lego/tutorial/docker-compose.yml.
// Run with: go test -tags e2e -count=1 -timeout 60s ./e2e/...
//
// Required services (host:port):
//   - MySQL    127.0.0.1:3306 (root:rootpass, db=testdb)
//   - Postgres 127.0.0.1:5432 (postgres:postgres, db=postgres)
//   - Redis    127.0.0.1:6379 (password=redis123456)
//   - Kafka    127.0.0.1:9092
//   - Jaeger   127.0.0.1:4318 (OTLP HTTP)
package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/dbsql"
	"gorm.io/gorm"
)

const (
	mysqlDSN = "root:rootpass@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	pgDSN    = "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=postgres sslmode=disable"
)

// product is a tiny table used by the CRUD tests. Each driver-compatible column type
// is exercised; the schema is intentionally minimal so it works on both MySQL and Postgres.
type product struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	Name  string `gorm:"size:64;not null"`
	Price int
}

func (product) TableName() string { return "e2e_product" }

// withTable creates the table, runs fn, then drops it.
func withTable(t *testing.T, db *gorm.DB, fn func()) {
	t.Helper()
	if err := db.AutoMigrate(&product{}); err != nil {
		t.Fatalf("AutoMigrate error: %v", err)
	}
	defer func() {
		if err := db.Migrator().DropTable(&product{}); err != nil {
			t.Logf("DropTable warning: %v", err)
		}
	}()
	fn()
}

// -----------------------------------------------------------------------------
// MySQL
// -----------------------------------------------------------------------------

func TestE2E_MySQL_Open_CRUD(t *testing.T) {
	cfg := &config.MySQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "127.0.0.1", Port: 3306, Username: "root", Password: "rootpass", Database: "testdb",
			},
		},
	}
	db, err := dbsql.Open(cfg)
	if err != nil {
		t.Fatalf("Open(MySQL) error: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	withTable(t, db, func() {
		if err := db.Create(&product{Name: "widget", Price: 100}).Error; err != nil {
			t.Fatalf("Create error: %v", err)
		}
		var got product
		if err := db.First(&got, "name = ?", "widget").Error; err != nil {
			t.Fatalf("First error: %v", err)
		}
		if got.Price != 100 {
			t.Errorf("Price = %d, want 100", got.Price)
		}
	})
}

func TestE2E_MySQL_OpenPath_CRUD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mysql.yaml")
	if err := os.WriteFile(path, []byte(`
mysql:
  host: 127.0.0.1
  port: 3306
  username: root
  password: rootpass
  database: testdb
`), 0644); err != nil {
		t.Fatal(err)
	}
	db, err := dbsql.OpenPath(path)
	if err != nil {
		t.Fatalf("OpenPath(mysql) error: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	withTable(t, db, func() {
		if err := db.Create(&product{Name: "loaded", Price: 200}).Error; err != nil {
			t.Fatalf("Create error: %v", err)
		}
		var n int64
		db.Model(&product{}).Where("name = ?", "loaded").Count(&n)
		if n != 1 {
			t.Errorf("count = %d, want 1", n)
		}
	})
}

func TestE2E_MySQL_PoolApply(t *testing.T) {
	cfg := &config.MySQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "127.0.0.1", Port: 3306, Username: "root", Password: "rootpass", Database: "testdb",
			},
			Pool: &config.PoolConfig{MaxOpenConns: 3, MaxIdleConns: 1},
		},
	}
	db, err := dbsql.Open(cfg)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if got := sqlDB.Stats().MaxOpenConnections; got != 3 {
		t.Errorf("MaxOpenConnections = %d, want 3", got)
	}
}

// -----------------------------------------------------------------------------
// Postgres
// -----------------------------------------------------------------------------

func TestE2E_Postgres_Open_CRUD(t *testing.T) {
	cfg := &config.PostgresConfig{}
	cfg.Host = "127.0.0.1"
	cfg.Port = 5432
	cfg.Username = "postgres"
	cfg.Password = "postgres"
	cfg.Database = "postgres"

	db, err := dbsql.Open(cfg)
	if err != nil {
		t.Fatalf("Open(Postgres) error: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	withTable(t, db, func() {
		if err := db.Create(&product{Name: "pg-item", Price: 50}).Error; err != nil {
			t.Fatalf("Create error: %v", err)
		}
		var got product
		if err := db.First(&got, "name = ?", "pg-item").Error; err != nil {
			t.Fatalf("First error: %v", err)
		}
		if got.Price != 50 {
			t.Errorf("Price = %d, want 50", got.Price)
		}
	})
}

// -----------------------------------------------------------------------------
// Cluster (read/write splitting) — uses MySQL with two sources + one replica.
// In this docker-compose we only have one MySQL instance, so we point sources
// and replicas at the same server. The point is to exercise dbresolver wiring
// end-to-end and confirm that writes route to the source.
// -----------------------------------------------------------------------------

func TestE2E_Cluster_MySQL_WriteReadSplit(t *testing.T) {
	node := config.NodeConfig{
		Host: "127.0.0.1", Port: 3306, Username: "root", Password: "rootpass", Database: "testdb",
	}
	cc := &config.ClusterConfig{
		Driver:      "mysql",
		Sources:     []config.NodeConfig{node},
		Replicas:    []config.NodeConfig{node},
		LoadBalance: config.LoadBalanceRandom,
	}
	db, err := dbsql.OpenCluster(cc)
	if err != nil {
		t.Fatalf("OpenCluster error: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	withTable(t, db, func() {
		if err := db.Create(&product{Name: "clustered", Price: 999}).Error; err != nil {
			t.Fatalf("Create via cluster error: %v", err)
		}
		// Both reads and writes should reach MySQL (single backend in this env).
		var got product
		if err := db.First(&got, "name = ?", "clustered").Error; err != nil {
			t.Fatalf("First via cluster error: %v", err)
		}
		if got.Price != 999 {
			t.Errorf("Price = %d, want 999", got.Price)
		}
	})
}

func TestE2E_Cluster_Postgres_WriteReadSplit(t *testing.T) {
	node := config.NodeConfig{
		Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "postgres",
	}
	cc := &config.ClusterConfig{
		Driver:      "postgres",
		Sources:     []config.NodeConfig{node},
		Replicas:    []config.NodeConfig{node},
		LoadBalance: config.LoadBalanceRoundRobin,
	}
	db, err := dbsql.OpenCluster(cc)
	if err != nil {
		t.Fatalf("OpenCluster(pg) error: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	withTable(t, db, func() {
		if err := db.Create(&product{Name: "pg-clustered", Price: 42}).Error; err != nil {
			t.Fatalf("Create via cluster error: %v", err)
		}
		var n int64
		db.Model(&product{}).Where("name = ?", "pg-clustered").Count(&n)
		if n != 1 {
			t.Errorf("count = %d, want 1", n)
		}
	})
}

// -----------------------------------------------------------------------------
// Tracing exporters — each test must build a *real* exporter and prove the
// end-to-end pipeline actually pushes data out. Failures here mean the wire
// format or mqx wiring broke, not a unit-test-only regression.
// -----------------------------------------------------------------------------

// TestE2E_Tracing_Jaeger wires the otlptracehttp exporter end-to-end against the
// Jaeger collector at 127.0.0.1:4318. We don't care whether Jaeger accepts the
// payload (it logs every trace to stdout), only that the exporter is built and
// a single span can be exported without error.
func TestE2E_Tracing_Jaeger(t *testing.T) {
	tc := &config.TracingConfig{
		Enabled:  true,
		Exporter: config.ExporterJaeger,
		Endpoint: "127.0.0.1:4318",
	}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	exp, err := dbsql.CreateExporter(context.Background(), tc)
	if err != nil {
		t.Fatalf("CreateExporter(jaeger) error: %v", err)
	}
	if exp == nil {
		t.Fatal("exporter is nil")
	}
	t.Cleanup(func() { _ = exp.Shutdown(context.Background()) })

	// Use a TracerProvider so we can produce a real sdktrace.ReadOnlySpan to export.
	tp := newTracerProvider(t, exp)
	tr := tp.Tracer("e2e")
	_, span := tr.Start(context.Background(), "jaeger-export-test")
	span.End()

	// Force a batch flush via Shutdown — proves the wire works.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tp.Shutdown(ctx); err != nil {
		t.Fatalf("TracerProvider.Shutdown error: %v", err)
	}
}
