package config_test

import (
	"strings"
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestOracleConfig_DriverName(t *testing.T) {
	cfg := &config.OracleConfig{}
	if cfg.DriverName() != "oracle" {
		t.Errorf("DriverName = %q, want oracle", cfg.DriverName())
	}
}

func TestOracleConfig_Defaults(t *testing.T) {
	cfg := &config.OracleConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Username: "system", Password: "pass", Database: "XEPDB1",
			},
		},
		ServiceName: "XEPDB1",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.Port != 1521 {
		t.Errorf("Port = %d, want 1521", cfg.Port)
	}
}

func TestOracleConfig_DSNWithServiceName(t *testing.T) {
	cfg := &config.OracleConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 1521, Username: "system", Password: "pass", Database: "XEPDB1",
			},
		},
		ServiceName: "XEPDB1",
	}
	dsn := cfg.DSN()
	if !strings.HasPrefix(dsn, "oracle://") {
		t.Errorf("DSN should start with oracle://, got: %s", dsn)
	}
	if !strings.Contains(dsn, "XEPDB1") {
		t.Errorf("DSN should contain service name, got: %s", dsn)
	}
}

func TestOracleConfig_DSNWithSID(t *testing.T) {
	cfg := &config.OracleConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 1521, Username: "system", Password: "pass", Database: "ORCL",
			},
		},
		SID: "ORCL",
	}
	dsn := cfg.DSN()
	if !strings.Contains(dsn, "sid=ORCL") {
		t.Errorf("DSN should contain sid=ORCL, got: %s", dsn)
	}
}

func TestOracleConfig_ValidateMissingBothServiceNameAndSID(t *testing.T) {
	cfg := &config.OracleConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Username: "system", Password: "pass", Database: "ORCL",
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should error when both ServiceName and SID are empty")
	}
}

func TestOracleConfig_ValidateMissingHost(t *testing.T) {
	cfg := &config.OracleConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Username: "system",
			},
		},
		ServiceName: "XEPDB1",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should error when host is empty")
	}
}

func TestOracleConfig_Interface(t *testing.T) {
	cfg := &config.OracleConfig{}
	var _ config.DBConfig = cfg
}
