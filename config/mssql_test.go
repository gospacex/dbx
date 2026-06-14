package config_test

import (
	"strings"
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestMSSQLConfig_DriverName(t *testing.T) {
	cfg := &config.MSSQLConfig{}
	if cfg.DriverName() != "mssql" {
		t.Errorf("DriverName = %q, want mssql", cfg.DriverName())
	}
}

func TestMSSQLConfig_Defaults(t *testing.T) {
	cfg := &config.MSSQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Username: "sa", Password: "pass", Database: "testdb",
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.Port != 1433 {
		t.Errorf("Port = %d, want 1433", cfg.Port)
	}
}

func TestMSSQLConfig_DSN(t *testing.T) {
	cfg := &config.MSSQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 1433, Username: "sa", Password: "pass", Database: "testdb",
			},
		},
	}
	dsn := cfg.DSN()
	if !strings.HasPrefix(dsn, "sqlserver://") {
		t.Errorf("DSN should start with sqlserver://, got: %s", dsn)
	}
}

func TestMSSQLConfig_DSNWithInstance(t *testing.T) {
	cfg := &config.MSSQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Username: "sa", Password: "pass", Database: "testdb",
			},
		},
		Instance: "SQLEXPRESS",
	}
	dsn := cfg.DSN()
	if !strings.Contains(dsn, "instance=SQLEXPRESS") {
		t.Errorf("DSN should contain instance=SQLEXPRESS, got: %s", dsn)
	}
}

func TestMSSQLConfig_ValidateMissingHost(t *testing.T) {
	cfg := &config.MSSQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Username: "sa",
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should error when host is empty")
	}
}

func TestMSSQLConfig_Interface(t *testing.T) {
	cfg := &config.MSSQLConfig{}
	var _ config.DBConfig = cfg
}
