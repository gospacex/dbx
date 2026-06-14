package config_test

import (
	"strings"
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestClusterConfig_Validate_UnsupportedDriver(t *testing.T) {
	cc := &config.ClusterConfig{
		Driver: "unsupported",
		Sources: []config.NodeConfig{
			{Host: "localhost", Port: 3306, Username: "root", Database: "test"},
		},
	}
	err := cc.Validate()
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestClusterConfig_Validate_NoSources(t *testing.T) {
	cc := &config.ClusterConfig{Driver: "mysql"}
	err := cc.Validate()
	if err == nil {
		t.Fatal("expected error for empty sources")
	}
}

func TestClusterConfig_Validate_InvalidLoadBalance(t *testing.T) {
	cc := &config.ClusterConfig{
		Driver:      "mysql",
		Sources:     []config.NodeConfig{{Host: "localhost", Port: 3306, Username: "root", Database: "test"}},
		LoadBalance: "invalid",
	}
	err := cc.Validate()
	if err == nil {
		t.Fatal("expected error for invalid load_balance")
	}
}

func TestClusterConfig_Defaults(t *testing.T) {
	cc := &config.ClusterConfig{
		Driver: "mysql",
		Sources: []config.NodeConfig{
			{Host: "source1", Port: 3306, Username: "root", Password: "pass", Database: "test"},
		},
		Replicas: []config.NodeConfig{
			{Host: "replica1", Port: 3306, Username: "root", Password: "pass", Database: "test"},
		},
	}
	if err := cc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cc.LoadBalance != "round_robin" {
		t.Errorf("LoadBalance = %q, want round_robin", cc.LoadBalance)
	}
	if cc.Pool == nil {
		t.Error("Pool should be initialized after Validate")
	}
}

func TestNodeConfig_DSN(t *testing.T) {
	n := config.NodeConfig{Host: "localhost", Port: 3306, Username: "root", Password: "pass", Database: "test"}
	dsn := n.DSN("mysql")
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
}

func TestNodeConfig_DSN_Postgres(t *testing.T) {
	n := config.NodeConfig{Host: "localhost", Port: 5432, Username: "u", Password: "p", Database: "d", SSLMode: "require", Schema: "public"}
	dsn := n.DSN("postgres")
	want := "host=localhost port=5432 user=u password=p dbname=d sslmode=require search_path=public"
	if dsn != want {
		t.Errorf("postgres DSN = %q, want %q", dsn, want)
	}
}

func TestNodeConfig_DSN_GaussDB(t *testing.T) {
	n := config.NodeConfig{Host: "localhost", Port: 5432, Username: "u", Password: "p", Database: "d"}
	dsn := n.DSN("gaussdb")
	if dsn == "" {
		t.Error("gaussdb DSN should not be empty")
	}
	if !strings.Contains(dsn, "host=localhost") {
		t.Errorf("gaussdb DSN missing host: %q", dsn)
	}
}

func TestNodeConfig_DSN_MSSQL_WithInstance(t *testing.T) {
	n := config.NodeConfig{Host: "localhost", Port: 1433, Username: "sa", Password: "p", Database: "d", Instance: "SQLEXPRESS"}
	dsn := n.DSN("mssql")
	if !strings.Contains(dsn, "instance=SQLEXPRESS") {
		t.Errorf("mssql DSN missing instance: %q", dsn)
	}
}

func TestNodeConfig_DSN_MSSQL_NoInstance(t *testing.T) {
	n := config.NodeConfig{Host: "localhost", Port: 1433, Username: "sa", Password: "p", Database: "d"}
	dsn := n.DSN("mssql")
	if !strings.Contains(dsn, "sqlserver://") {
		t.Errorf("mssql DSN missing scheme: %q", dsn)
	}
}

func TestNodeConfig_DSN_Oracle_ServiceName(t *testing.T) {
	n := config.NodeConfig{Host: "localhost", Port: 1521, Username: "u", Password: "p", ServiceName: "ORCL"}
	dsn := n.DSN("oracle")
	if !strings.Contains(dsn, "ORCL") {
		t.Errorf("oracle DSN missing service_name: %q", dsn)
	}
}

func TestNodeConfig_DSN_Oracle_SID(t *testing.T) {
	n := config.NodeConfig{Host: "localhost", Port: 1521, Username: "u", Password: "p", Database: "d", SID: "ORCLSID"}
	dsn := n.DSN("oracle")
	if !strings.Contains(dsn, "sid=ORCLSID") {
		t.Errorf("oracle DSN missing sid: %q", dsn)
	}
}

func TestNodeConfig_DSN_SQLite(t *testing.T) {
	n := config.NodeConfig{Path: "/tmp/test.db"}
	dsn := n.DSN("sqlite")
	if dsn != "/tmp/test.db" {
		t.Errorf("sqlite DSN = %q, want /tmp/test.db", dsn)
	}
}

func TestNodeConfig_DSN_Unknown(t *testing.T) {
	n := config.NodeConfig{Host: "localhost", Port: 3306, Username: "u", Password: "p", Database: "d"}
	dsn := n.DSN("unknown")
	if dsn == "" {
		t.Error("unknown driver DSN should fall back to non-empty")
	}
}

func TestNodeConfig_DSN_TiDB(t *testing.T) {
	n := config.NodeConfig{Host: "localhost", Port: 4000, Username: "root", Password: "p", Database: "d"}
	dsn := n.DSN("tidb")
	if !strings.Contains(dsn, "charset=utf8mb4") {
		t.Errorf("tidb DSN missing charset: %q", dsn)
	}
}
