package config_test

import (
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestPostgresConfig_DriverName(t *testing.T) {
	cfg := &config.PostgresConfig{}
	if cfg.DriverName() != "postgres" {
		t.Errorf("DriverName = %q, want postgres", cfg.DriverName())
	}
}

func TestPostgresConfig_Defaults(t *testing.T) {
	cfg := &config.PostgresConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Username: "postgres", Password: "pass", Database: "test",
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %d, want 5432", cfg.Port)
	}
	if cfg.SSLMode != "disable" {
		t.Errorf("SSLMode = %q, want disable", cfg.SSLMode)
	}
	if cfg.TimeZone != "UTC" {
		t.Errorf("TimeZone = %q, want UTC", cfg.TimeZone)
	}
}

func TestPostgresConfig_DSN(t *testing.T) {
	cfg := &config.PostgresConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 5432, Username: "postgres", Password: "pass", Database: "test",
			},
		},
		SSLMode: "disable", TimeZone: "UTC",
	}
	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
}

func TestPostgresConfig_ValidateMissingHost(t *testing.T) {
	cfg := &config.PostgresConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Username: "postgres",
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should error when host is empty")
	}
}

func TestPostgresConfig_ValidateMissingUsername(t *testing.T) {
	cfg := &config.PostgresConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost",
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should error when username is empty")
	}
}

func TestPostgresConfig_DSNWithSchema(t *testing.T) {
	cfg := &config.PostgresConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 5432, Username: "postgres", Password: "pass", Database: "test",
			},
		},
		SSLMode: "disable", TimeZone: "UTC", Schema: "myschema",
	}
	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
}

func TestPostgresConfig_Interface(t *testing.T) {
	cfg := &config.PostgresConfig{}
	var _ config.DBConfig = cfg
}
