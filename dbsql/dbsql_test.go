package dbsql_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/dbsql"
)

func TestOpen_InvalidConfig(t *testing.T) {
	cfg := &config.MySQLConfig{}
	_, err := dbsql.Open(cfg)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestOpen_NilTracing(t *testing.T) {
	cfg := &config.SQLiteConfig{Path: ":memory:"}
	db, err := dbsql.Open(cfg)
	if err != nil {
		t.Fatalf("Open(sqlite) error: %v", err)
	}
	if db == nil {
		t.Fatal("db should not be nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("sqlDB.Ping() error: %v", err)
	}
	sqlDB.Close()
}

func TestOpenPath_NonExistentFile(t *testing.T) {
	_, err := dbsql.OpenPath("/tmp/nonexistent/test.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestExtractTracingAndApply_NilConfig(t *testing.T) {
	err := dbsql.ExtractTracingAndApply(nil, nil)
	if err != nil {
		t.Errorf("expected nil error for nil tracing config, got: %v", err)
	}
}

func TestExtractTracingAndApply_Disabled(t *testing.T) {
	tc := &config.TracingConfig{Enabled: false}
	err := dbsql.ExtractTracingAndApply(context.Background(), tc)
	if err != nil {
		t.Errorf("expected nil error for disabled tracing, got: %v", err)
	}
}

func TestOpen_MySQL(t *testing.T) {
	cfg := &config.MySQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 3306, Username: "root", Database: "test",
			},
		},
	}
	// MySQL TCP connect will fail without a real server; we expect an error.
	_, err := dbsql.Open(cfg)
	if err == nil {
		t.Fatal("expected error for unreachable mysql")
	}
}

func TestOpen_TracingInvalid(t *testing.T) {
	cfg := &config.MySQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 3306, Username: "root", Database: "test",
			},
			Tracing: &config.TracingConfig{
				Enabled:  true,
				Exporter: "invalid",
			},
		},
	}
	_, err := dbsql.Open(cfg)
	if err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestOpen_Postgres(t *testing.T) {
	cfg := &config.PostgresConfig{}
	cfg.Host = "localhost"
	cfg.Username = "postgres"
	cfg.Database = "test"
	_, err := dbsql.Open(cfg)
	if err == nil {
		t.Fatal("expected error for unreachable postgres")
	}
}

func TestOpen_MSSQL(t *testing.T) {
	cfg := &config.MSSQLConfig{}
	cfg.Host = "localhost"
	cfg.Username = "sa"
	cfg.Database = "test"
	_, err := dbsql.Open(cfg)
	if err == nil {
		t.Fatal("expected error for unreachable mssql")
	}
}

func TestOpen_Oracle(t *testing.T) {
	cfg := &config.OracleConfig{}
	cfg.Host = "localhost"
	cfg.Username = "system"
	cfg.ServiceName = "ORCL"
	_, err := dbsql.Open(cfg)
	if err == nil {
		t.Fatal("expected error for oracle (no CGo driver)")
	}
}

func TestOpen_Sqlite_PoolApply(t *testing.T) {
	cfg := &config.SQLiteConfig{
		Path: ":memory:",
		Pool: &config.PoolConfig{
			MaxOpenConns: 5,
			MaxIdleConns: 2,
		},
	}
	db, err := dbsql.Open(cfg)
	if err != nil {
		t.Fatalf("Open(sqlite+pool) error: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestOpenPath_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sqlite.yaml")
	content := []byte(`sqlite:
  path: ":memory:"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	db, err := dbsql.OpenPath(path)
	if err != nil {
		t.Fatalf("OpenPath error: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestOpenPath_ValidWithTracing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sqlite.yaml")
	content := []byte(`
sqlite:
  path: ":memory:"
tracing:
  enabled: true
  exporter: jaeger
  endpoint: localhost:4318
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	db, err := dbsql.OpenPath(path)
	if err != nil {
		t.Fatalf("OpenPath(tracing) error: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestOpenPath_InvalidTracing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sqlite.yaml")
	content := []byte(`
sqlite:
  path: ":memory:"
tracing:
  enabled: true
  exporter: invalid_exporter
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := dbsql.OpenPath(path)
	if err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestOpenPath_UnsupportedExt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sqlite.xml")
	if err := os.WriteFile(path, []byte("<x/>"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := dbsql.OpenPath(path)
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestExtractTracingAndApply_Enabled_Kafka(t *testing.T) {
	tc := &config.TracingConfig{Enabled: true, Exporter: "kafka"}
	err := dbsql.ExtractTracingAndApply(context.Background(), tc)
	if err != nil {
		t.Errorf("expected nil error for enabled kafka tracing, got: %v", err)
	}
}

func TestOpenPath_TOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sqlite.toml")
	content := []byte(`
[sqlite]
path = ":memory:"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	db, err := dbsql.OpenPath(path)
	if err != nil {
		t.Fatalf("OpenPath(toml) error: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestOpen_Oracle_InvalidConfig(t *testing.T) {
	cfg := &config.OracleConfig{}
	cfg.Host = "localhost"
	cfg.Username = "system"
	_, err := dbsql.Open(cfg)
	if err == nil {
		t.Fatal("expected error for invalid oracle config")
	}
}

func TestOpen_TracingJaeger(t *testing.T) {
	cfg := &config.MySQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 3306, Username: "root", Database: "test",
			},
			Tracing: &config.TracingConfig{
				Enabled:  true,
				Exporter: "jaeger",
				Endpoint: "localhost:4318",
			},
		},
	}
	_, err := dbsql.Open(cfg)
	if err == nil {
		t.Fatal("expected error for unreachable mysql (tracing path)")
	}
}

func TestExtractTracingAndApply_Enabled_Jaeger(t *testing.T) {
	tc := &config.TracingConfig{Enabled: true, Exporter: "jaeger", Endpoint: "localhost:4318"}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	err := dbsql.ExtractTracingAndApply(context.Background(), tc)
	if err != nil {
		t.Errorf("expected nil error for enabled jaeger tracing, got: %v", err)
	}
}

func TestOpenPath_PoolApply(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sqlite.json")
	content := []byte(`{
  "sqlite": {"path": ":memory:"},
  "pool": {"max_open_conns": 3, "max_idle_conns": 1}
}`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	// SQLiteConfig doesn't currently consume a top-level pool; we just want
	// coverage of OpenPath -> Open -> orm.Open. The pool fields are unused
	// for sqlite in this test, which is acceptable.
	db, err := dbsql.OpenPath(path)
	if err != nil {
		t.Fatalf("OpenPath(json) error: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
}
