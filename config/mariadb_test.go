package config_test

import (
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestMariaDBConfig_DriverName(t *testing.T) {
	cfg := &config.MariaDBConfig{}
	if cfg.DriverName() != "mysql" {
		t.Errorf("DriverName = %q, want mysql", cfg.DriverName())
	}
}

func TestMariaDBConfig_DefaultPort(t *testing.T) {
	cfg := &config.MariaDBConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Username: "root", Password: "pass", Database: "test",
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.Port != 3306 {
		t.Errorf("Port = %d, want 3306", cfg.Port)
	}
}

func TestMariaDBConfig_DSN(t *testing.T) {
	cfg := &config.MariaDBConfig{
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

func TestMariaDBConfig_Validate_MissingRequired(t *testing.T) {
	cfg := &config.MariaDBConfig{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestMariaDBConfig_ImplementsDBConfig(t *testing.T) {
	var _ config.DBConfig = (*config.MariaDBConfig)(nil)
}
