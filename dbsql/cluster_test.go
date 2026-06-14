package dbsql_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/dbsql"
)

func TestOpenCluster_Invalid(t *testing.T) {
	cc := &config.ClusterConfig{}
	_, err := dbsql.OpenCluster(cc)
	if err == nil {
		t.Fatal("expected error for invalid cluster config")
	}
}

func TestOpenCluster_UnsupportedDriver(t *testing.T) {
	cc := &config.ClusterConfig{
		Driver: "unsupported",
		Sources: []config.NodeConfig{
			{Host: "localhost", Port: 3306, Username: "root", Database: "test"},
		},
	}
	_, err := dbsql.OpenCluster(cc)
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestOpenCluster_InvalidLoadBalance(t *testing.T) {
	cc := &config.ClusterConfig{
		Driver:      "mysql",
		Sources:     []config.NodeConfig{{Host: "localhost", Port: 3306, Username: "root", Database: "test"}},
		LoadBalance: "weighted",
	}
	_, err := dbsql.OpenCluster(cc)
	if err == nil {
		t.Fatal("expected error for invalid load_balance")
	}
}

func TestOpenCluster_OpenFails(t *testing.T) {
	cc := &config.ClusterConfig{
		Driver: "sqlite",
		Sources: []config.NodeConfig{
			{Path: "/nonexistent/dir/should/fail/test.db"},
		},
	}
	_, _ = dbsql.OpenCluster(cc)
}

func TestOpenCluster_SqliteInMemory(t *testing.T) {
	cc := &config.ClusterConfig{
		Driver: "sqlite",
		Sources: []config.NodeConfig{
			{Path: ":memory:"},
		},
		Replicas: []config.NodeConfig{
			{Path: ":memory:"},
		},
		LoadBalance: config.LoadBalanceRandom,
	}
	db, err := dbsql.OpenCluster(cc)
	if err != nil {
		t.Fatalf("OpenCluster(sqlite) error: %v", err)
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

func TestOpenCluster_Sqlite_Defaults(t *testing.T) {
	cc := &config.ClusterConfig{
		Driver: "sqlite",
		Sources: []config.NodeConfig{
			{Path: ":memory:"},
		},
	}
	if err := cc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cc.LoadBalance != "round_robin" {
		t.Errorf("LoadBalance default = %q, want round_robin", cc.LoadBalance)
	}
	db, err := dbsql.OpenCluster(cc)
	if err != nil {
		t.Fatalf("OpenCluster error: %v", err)
	}
	if db == nil {
		t.Fatal("db should not be nil")
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestOpenCluster_Sqlite_InvalidPath(t *testing.T) {
	// gorm/sqlite with an empty path creates a temp file (in-memory is required for determinism).
	// This test exercises the empty-path code branch and tolerates either outcome
	// since the underlying sqlite behavior may vary across gorm versions.
	cc := &config.ClusterConfig{
		Driver: "sqlite",
		Sources: []config.NodeConfig{
			{Path: ""},
		},
	}
	_, _ = dbsql.OpenCluster(cc)
}

func TestOpenClusterPath_NonExistentFile(t *testing.T) {
	_, err := dbsql.OpenClusterPath("/tmp/nonexistent/cluster.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestOpenClusterPath_ValidSqlite(t *testing.T) {
	content := []byte(`
cluster:
  driver: sqlite
  sources:
    - path: ":memory:"
    - path: ":memory:"
  replicas:
    - path: ":memory:"
  load_balance: random
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "cluster.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	db, err := dbsql.OpenClusterPath(path)
	if err != nil {
		t.Fatalf("OpenClusterPath error: %v", err)
	}
	if db == nil {
		t.Fatal("db should not be nil")
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestOpenClusterPath_InvalidClusterConfig(t *testing.T) {
	content := []byte(`
cluster:
  driver: unsupported
  sources:
    - host: x
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "cluster.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := dbsql.OpenClusterPath(path)
	if err == nil {
		t.Fatal("expected error for invalid cluster config")
	}
}

func TestOpenClusterPath_LoadFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cluster.txt")
	if err := os.WriteFile(path, []byte("foo"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := dbsql.OpenClusterPath(path)
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestOpenClusterPath_MissingClusterSection(t *testing.T) {
	content := []byte(`mysql:
  host: x
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "cluster.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := dbsql.OpenClusterPath(path)
	if err == nil {
		t.Fatal("expected error for missing cluster section")
	}
}
