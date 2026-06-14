package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gospacex/dbx/config"
)

// --- Per-driver loader tests (YAML) ---

func TestLoader_LoadPostgres_YAML(t *testing.T) {
	content := []byte(`
postgres:
  host: localhost
  port: 5432
  username: postgres
  password: secret
  database: mydb
  ssl_mode: require
  time_zone: Asia/Shanghai
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "test_postgres.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, tc, err := config.LoadPostgres(path)
	if err != nil {
		t.Fatalf("LoadPostgres error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "postgres" {
		t.Errorf("DriverName = %q, want postgres", cfg.DriverName())
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want localhost", cfg.Host)
	}
	if tc != nil {
		t.Error("tracing should be nil")
	}
}

func TestLoader_LoadTiDB_YAML(t *testing.T) {
	content := []byte(`
tidb:
  host: localhost
  username: root
  password: secret
  database: testdb
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "test_tidb.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.LoadTiDB(path)
	if err != nil {
		t.Fatalf("LoadTiDB error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "mysql" {
		t.Errorf("DriverName = %q, want mysql (TiDB is wire-compatible with MySQL)", cfg.DriverName())
	}
	if cfg.Port != 4000 {
		t.Errorf("Port = %d, want 4000", cfg.Port)
	}
}

func TestLoader_LoadMariaDB_YAML(t *testing.T) {
	content := []byte(`
mariadb:
  host: localhost
  port: 3306
  username: root
  password: secret
  database: testdb
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "test_mariadb.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.LoadMariaDB(path)
	if err != nil {
		t.Fatalf("LoadMariaDB error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "mysql" {
		t.Errorf("DriverName = %q, want mysql", cfg.DriverName())
	}
}

func TestLoader_LoadGaussDB_YAML(t *testing.T) {
	content := []byte(`
gaussdb:
  host: localhost
  port: 5432
  username: dbuser
  password: secret
  database: mydb
  ssl_mode: require
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "test_gaussdb.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.LoadGaussDB(path)
	if err != nil {
		t.Fatalf("LoadGaussDB error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "postgres" {
		t.Errorf("DriverName = %q, want postgres", cfg.DriverName())
	}
}

func TestLoader_LoadMSSQL_YAML(t *testing.T) {
	content := []byte(`
mssql:
  host: localhost
  port: 1433
  username: sa
  password: secret
  database: master
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "test_mssql.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.LoadMSSQL(path)
	if err != nil {
		t.Fatalf("LoadMSSQL error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "mssql" {
		t.Errorf("DriverName = %q, want mssql", cfg.DriverName())
	}
}

func TestLoader_LoadOracle_YAML(t *testing.T) {
	content := []byte(`
oracle:
  host: localhost
  port: 1521
  username: system
  password: secret
  service_name: orcl
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "test_oracle.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.LoadOracle(path)
	if err != nil {
		t.Fatalf("LoadOracle error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "oracle" {
		t.Errorf("DriverName = %q, want oracle", cfg.DriverName())
	}
}

func TestLoader_LoadSQLite_YAML(t *testing.T) {
	content := []byte(`
sqlite:
  path: /tmp/test.db
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "test_sqlite.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.LoadSQLite(path)
	if err != nil {
		t.Fatalf("LoadSQLite error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "sqlite" {
		t.Errorf("DriverName = %q, want sqlite", cfg.DriverName())
	}
	if cfg.Path != "/tmp/test.db" {
		t.Errorf("Path = %q, want /tmp/test.db", cfg.Path)
	}
}

// --- Load() unified loader tests ---

func TestLoader_Load_Unified_Postgres(t *testing.T) {
	content := []byte(`{"postgres": {"host": "localhost", "port": 5432, "username": "postgres", "database": "test"}}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "postgres.json")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "postgres" {
		t.Errorf("DriverName = %q, want postgres", cfg.DriverName())
	}
}

func TestLoader_Load_Unified_PostgreSQL(t *testing.T) {
	content := []byte(`{"postgres": {"host": "localhost", "port": 5432, "username": "postgres", "database": "test"}}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "postgresql.json")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "postgres" {
		t.Errorf("DriverName = %q, want postgres", cfg.DriverName())
	}
}

func TestLoader_Load_Unified_Toml(t *testing.T) {
	content := []byte(`
[mysql]
host = "localhost"
port = 3306
username = "root"
database = "test"
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "mysql.toml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.DriverName() != "mysql" {
		t.Errorf("DriverName = %q, want mysql", cfg.DriverName())
	}
}

func TestLoader_Load_Unified_MissingSection(t *testing.T) {
	content := []byte(`host: localhost`)
	dir := t.TempDir()
	path := filepath.Join(dir, "mysql.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing mysql section")
	}
}

func TestLoader_Load_DispatchBySection_Postgres(t *testing.T) {
	content := []byte(`
postgres:
  host: localhost
  port: 5432
  username: postgres
  database: appdb
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "db.yaml") // generic filename
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil || cfg.DriverName() != "postgres" {
		t.Errorf("expected postgres, got %v", cfg)
	}
}

func TestLoader_Load_DispatchBySection_SQLite(t *testing.T) {
	content := []byte(`{"sqlite": {"path": "/tmp/x.db"}}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil || cfg.DriverName() != "sqlite" {
		t.Errorf("expected sqlite, got %v", cfg)
	}
}

func TestLoader_Load_DispatchBySection_NoSection(t *testing.T) {
	content := []byte(`host: localhost`)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for no driver section")
	}
}

func TestLoader_Load_DispatchBySection_InvalidConfig(t *testing.T) {
	content := []byte(`
mysql:
  port: 3306
  database: test
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid mysql config (no host/username)")
	}
}

func TestLoader_Load_DispatchBySection_InvalidTracing(t *testing.T) {
	content := []byte(`
postgres:
  host: localhost
  username: postgres
  database: test
tracing:
  enabled: true
  exporter: invalid_exporter
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestLoader_Load_UnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mysql.txt")
	if err := os.WriteFile(path, []byte("foo"), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

// --- loadFile tests ---

func TestLoader_loadFile_Json(t *testing.T) {
	content := []byte(`{"mysql": {"host": "localhost", "port": 3306, "username": "root", "database": "test"}}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.LoadMySQL(path)
	if err != nil {
		t.Fatalf("LoadMySQL error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
}

func TestLoader_loadFile_Toml(t *testing.T) {
	content := []byte(`
[mysql]
host = "localhost"
port = 3306
username = "root"
database = "test"
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := config.LoadMySQL(path)
	if err != nil {
		t.Fatalf("LoadMySQL error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
}

func TestLoader_loadFile_UnsupportedExt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xml")
	if err := os.WriteFile(path, []byte("<root/>"), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := config.LoadMySQL(path)
	if err == nil {
		t.Fatal("expected error for unsupported file extension")
	}
}

func TestLoader_loadFile_NonExistent(t *testing.T) {
	_, _, err := config.LoadMySQL("/tmp/nonexistent/test.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

// --- LoadCluster tracing validation ---

func TestLoader_LoadCluster_TracingInvalid(t *testing.T) {
	content := []byte(`
cluster:
  driver: mysql
  sources:
    - host: source1
      port: 3306
      username: root
      password: pass
      database: test
tracing:
  enabled: true
  exporter: invalid_exporter
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "cluster.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := config.LoadCluster(path)
	if err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

// --- DBConfig interface method tests ---

func TestBaseDBConfig_GetPool(t *testing.T) {
	b := config.BaseDBConfig{
		Pool: &config.PoolConfig{MaxOpenConns: 10},
	}
	p := b.GetPool()
	if p == nil || p.MaxOpenConns != 10 {
		t.Error("GetPool should return the pool config")
	}
}

func TestBaseDBConfig_GetTracing(t *testing.T) {
	b := config.BaseDBConfig{
		Tracing: &config.TracingConfig{Enabled: true},
	}
	tr := b.GetTracing()
	if tr == nil || !tr.Enabled {
		t.Error("GetTracing should return the tracing config")
	}
}

func TestBaseDBConfig_DSN(t *testing.T) {
	b := config.BaseDBConfig{
		CommonNetworkConfig: config.CommonNetworkConfig{
			Host: "localhost", Port: 3306, Username: "user", Password: "pass", Database: "db",
		},
	}
	dsn := b.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
}

func TestCommonNetworkConfig_DSN(t *testing.T) {
	c := config.CommonNetworkConfig{
		Host: "localhost", Port: 3306, Username: "user", Password: "pass", Database: "db",
	}
	dsn := c.DSN()
	expected := "user:pass@tcp(localhost:3306)/db"
	if dsn != expected {
		t.Errorf("DSN = %q, want %q", dsn, expected)
	}
}

// --- Missing-section + invalid-config + invalid-tracing coverage for per-driver loaders ---

func TestLoader_LoadPostgres_MissingSection(t *testing.T) {
	content := []byte(`mysql:
  host: h
  username: u
  database: d
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadPostgres(path); err == nil {
		t.Fatal("expected error for missing postgres section")
	}
}

func TestLoader_LoadPostgres_InvalidConfig(t *testing.T) {
	content := []byte(`postgres:
  host: h
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadPostgres(path); err == nil {
		t.Fatal("expected error for invalid postgres config (no username/db)")
	}
}

func TestLoader_LoadPostgres_InvalidTracing(t *testing.T) {
	content := []byte(`
postgres:
  host: h
  username: u
  database: d
tracing:
  enabled: true
  exporter: bogus
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadPostgres(path); err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestLoader_LoadTiDB_MissingSection(t *testing.T) {
	content := []byte(`mysql: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadTiDB(path); err == nil {
		t.Fatal("expected error for missing tidb section")
	}
}

func TestLoader_LoadTiDB_InvalidConfig(t *testing.T) {
	content := []byte(`tidb: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadTiDB(path); err == nil {
		t.Fatal("expected error for invalid tidb config (no host)")
	}
}

func TestLoader_LoadTiDB_InvalidTracing(t *testing.T) {
	content := []byte(`
tidb:
  host: h
tracing:
  enabled: true
  exporter: bogus
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadTiDB(path); err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestLoader_LoadMariaDB_MissingSection(t *testing.T) {
	content := []byte(`mysql: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadMariaDB(path); err == nil {
		t.Fatal("expected error for missing mariadb section")
	}
}

func TestLoader_LoadMariaDB_InvalidConfig(t *testing.T) {
	content := []byte(`mariadb: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadMariaDB(path); err == nil {
		t.Fatal("expected error for invalid mariadb config")
	}
}

func TestLoader_LoadMariaDB_InvalidTracing(t *testing.T) {
	content := []byte(`
mariadb:
  host: h
tracing:
  enabled: true
  exporter: bogus
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadMariaDB(path); err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestLoader_LoadGaussDB_MissingSection(t *testing.T) {
	content := []byte(`mysql: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadGaussDB(path); err == nil {
		t.Fatal("expected error for missing gaussdb section")
	}
}

func TestLoader_LoadGaussDB_InvalidConfig(t *testing.T) {
	content := []byte(`gaussdb: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadGaussDB(path); err == nil {
		t.Fatal("expected error for invalid gaussdb config")
	}
}

func TestLoader_LoadGaussDB_InvalidTracing(t *testing.T) {
	content := []byte(`
gaussdb:
  host: h
tracing:
  enabled: true
  exporter: bogus
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadGaussDB(path); err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestLoader_LoadMSSQL_MissingSection(t *testing.T) {
	content := []byte(`mysql: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadMSSQL(path); err == nil {
		t.Fatal("expected error for missing mssql section")
	}
}

func TestLoader_LoadMSSQL_InvalidConfig(t *testing.T) {
	content := []byte(`mssql: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadMSSQL(path); err == nil {
		t.Fatal("expected error for invalid mssql config")
	}
}

func TestLoader_LoadMSSQL_InvalidTracing(t *testing.T) {
	content := []byte(`
mssql:
  host: h
tracing:
  enabled: true
  exporter: bogus
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadMSSQL(path); err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestLoader_LoadOracle_MissingSection(t *testing.T) {
	content := []byte(`mysql: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadOracle(path); err == nil {
		t.Fatal("expected error for missing oracle section")
	}
}

func TestLoader_LoadOracle_InvalidConfig(t *testing.T) {
	content := []byte(`oracle: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadOracle(path); err == nil {
		t.Fatal("expected error for invalid oracle config")
	}
}

func TestLoader_LoadOracle_InvalidTracing(t *testing.T) {
	content := []byte(`
oracle:
  host: h
tracing:
  enabled: true
  exporter: bogus
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadOracle(path); err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestLoader_LoadSQLite_MissingSection(t *testing.T) {
	content := []byte(`mysql: {}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadSQLite(path); err == nil {
		t.Fatal("expected error for missing sqlite section")
	}
}

func TestLoader_LoadSQLite_InvalidTracing(t *testing.T) {
	content := []byte(`
sqlite:
  path: ":memory:"
tracing:
  enabled: true
  exporter: bogus
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadSQLite(path); err == nil {
		t.Fatal("expected error for invalid tracing exporter")
	}
}

func TestLoader_LoadCluster_MissingSection(t *testing.T) {
	content := []byte(`mysql: {host: h}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadCluster(path); err == nil {
		t.Fatal("expected error for missing cluster section")
	}
}

func TestLoader_LoadCluster_InvalidConfig(t *testing.T) {
	content := []byte(`cluster:
  driver: sqlite
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.LoadCluster(path); err == nil {
		t.Fatal("expected error for invalid cluster config (no sources)")
	}
}

// --- Load() dispatch coverage for all 8 drivers ---

func TestLoader_Load_Dispatch_TiDB(t *testing.T) {
	content := []byte(`
tidb:
  host: localhost
  username: root
  database: test
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "db.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil || cfg.DriverName() != "mysql" {
		t.Errorf("expected tidb->mysql, got %v", cfg)
	}
}

func TestLoader_Load_Dispatch_MariaDB(t *testing.T) {
	content := []byte(`
mariadb:
  host: localhost
  username: root
  database: test
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "db.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil || cfg.DriverName() != "mysql" {
		t.Errorf("expected mariadb->mysql, got %v", cfg)
	}
}

func TestLoader_Load_Dispatch_GaussDB(t *testing.T) {
	content := []byte(`
gaussdb:
  host: localhost
  username: postgres
  database: test
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "db.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil || cfg.DriverName() != "postgres" {
		t.Errorf("expected gaussdb->postgres, got %v", cfg)
	}
}

func TestLoader_Load_Dispatch_MSSQL(t *testing.T) {
	content := []byte(`
mssql:
  host: localhost
  username: sa
  database: test
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "db.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil || cfg.DriverName() != "mssql" {
		t.Errorf("expected mssql, got %v", cfg)
	}
}

func TestLoader_Load_Dispatch_Oracle(t *testing.T) {
	content := []byte(`
oracle:
  host: localhost
  username: system
  service_name: orcl
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "db.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil || cfg.DriverName() != "oracle" {
		t.Errorf("expected oracle, got %v", cfg)
	}
}

func TestLoader_Load_Dispatch_InvalidTracingFirst(t *testing.T) {
	// Tracing validate runs before driver dispatch in Load(); invalid tracing
	// must produce an error even if the driver section is well-formed.
	content := []byte(`
mysql:
  host: h
  username: u
  database: d
tracing:
  enabled: true
  exporter: bogus
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "db.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.Load(path); err == nil {
		t.Fatal("expected error for invalid tracing")
	}
}
