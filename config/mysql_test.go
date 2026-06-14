package config_test

import (
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestMySQLConfig_DriverName(t *testing.T) {
	cfg := &config.MySQLConfig{}
	if cfg.DriverName() != "mysql" {
		t.Errorf("DriverName = %q, want mysql", cfg.DriverName())
	}
}

func TestMySQLConfig_Validate_Defaults(t *testing.T) {
	cfg := &config.MySQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Username: "root", Password: "secret", Database: "test",
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.Port != 3306 {
		t.Errorf("Port = %d, want 3306", cfg.Port)
	}
	if cfg.Charset != "utf8mb4" {
		t.Errorf("Charset = %q, want utf8mb4", cfg.Charset)
	}
}

func TestMySQLConfig_DSN(t *testing.T) {
	cfg := &config.MySQLConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 3306, Username: "root", Password: "secret", Database: "test",
			},
		},
		Charset: "utf8mb4",
	}
	dsn := cfg.DSN()
	expected := "root:secret@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
	if dsn != expected {
		t.Errorf("DSN = %q, want %q", dsn, expected)
	}
}

func TestMySQLConfig_Validate_MissingRequired(t *testing.T) {
	cfg := &config.MySQLConfig{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestMySQLConfig_ImplementsDBConfig(t *testing.T) {
	var _ config.DBConfig = (*config.MySQLConfig)(nil)
}
