package dbsql_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gospacex/configx"
	"github.com/gospacex/dbx/dbsql"
)

// writeEntryConfig writes body to a temp config.yaml and returns its path.
func writeEntryConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// entryYAML mirrors the xmiddleware.sqlx contract with a sqlite default
// instance, so the triad can be exercised without a database server.
const entryYAML = `
xmiddleware:
  sqlx:
    driver: gorm
    databases:
      default:
        dsn: ":memory:"
        dialect: sqlite
        max_open_conns: 4
        max_idle_conns: 1
        enable_log: false
        enable_trace: false
`

// TestReadAndCreate_OpensDefaultInstance proves the entry point performs the
// whole job — read config, pick the default instance, open a working
// connection — rather than merely returning a config handle.
func TestReadAndCreate_OpensDefaultInstance(t *testing.T) {
	db, err := dbsql.ReadAndCreate(configx.ConfigOptions{ConfigFile: writeEntryConfig(t, entryYAML)})
	if err != nil {
		t.Fatalf("ReadAndCreate: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	// A real round-trip: the connection must actually work, not just exist.
	if err := db.Exec("CREATE TABLE entry_probe (id INTEGER PRIMARY KEY, name TEXT)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec("INSERT INTO entry_probe (name) VALUES (?)", "hubx").Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	var name string
	if err := db.Raw("SELECT name FROM entry_probe LIMIT 1").Scan(&name).Error; err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "hubx" {
		t.Errorf("round-trip returned %q, want %q", name, "hubx")
	}

	// max_open_conns is part of the same contract and must reach database/sql.
	if got := sqlDB.Stats().MaxOpenConnections; got != 4 {
		t.Errorf("MaxOpenConnections = %d, want 4", got)
	}
}

// TestReadAndCreate_TargetsDefaultInstance pins the instance name: a config
// that only defines a differently named database must fail while naming
// "default", proving the entry point does not silently pick a sibling.
func TestReadAndCreate_TargetsDefaultInstance(t *testing.T) {
	const onlyReplica = `
xmiddleware:
  sqlx:
    driver: gorm
    databases:
      replica:
        dsn: ":memory:"
        dialect: sqlite
`
	_, err := dbsql.ReadAndCreate(configx.ConfigOptions{ConfigFile: writeEntryConfig(t, onlyReplica)})
	if err == nil {
		t.Fatal("ReadAndCreate must fail when xmiddleware.sqlx.databases.default is absent")
	}
	if !strings.Contains(err.Error(), "default") {
		t.Errorf("error %q should name the default instance", err.Error())
	}
}

// TestReadAndCreate_MissingDSNFails keeps the contract's required key honest.
func TestReadAndCreate_MissingDSNFails(t *testing.T) {
	const noDSN = `
xmiddleware:
  sqlx:
    driver: gorm
    databases:
      default:
        dialect: sqlite
`
	_, err := dbsql.ReadAndCreate(configx.ConfigOptions{ConfigFile: writeEntryConfig(t, noDSN)})
	if err == nil {
		t.Fatal("ReadAndCreate must fail without a dsn")
	}
	if !strings.Contains(err.Error(), "dsn is required") {
		t.Errorf("error %q should say the dsn is required", err.Error())
	}
}

// TestReadAndCreate_NoConfigFileFails: no config source at all must error out
// instead of returning a half-built handle.
func TestReadAndCreate_NoConfigFileFails(t *testing.T) {
	_, err := dbsql.ReadAndCreate(configx.ConfigOptions{
		ConfigFile: filepath.Join(t.TempDir(), "absent.yaml"),
	})
	if err == nil {
		t.Fatal("ReadAndCreate without a config file must fail")
	}
}

func TestReadConfig_DelegatesWithoutPanic(t *testing.T) {
	if err := dbsql.ReadConfig(configx.ConfigOptions{ConfigFile: writeEntryConfig(t, entryYAML)}); err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
}

func TestMustReadAndCreate_ReturnsDB(t *testing.T) {
	db := dbsql.MustReadAndCreate(configx.ConfigOptions{ConfigFile: writeEntryConfig(t, entryYAML)})
	if db == nil {
		t.Fatal("MustReadAndCreate returned nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	_ = sqlDB.Close()
}
