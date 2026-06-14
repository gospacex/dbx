package config_test

import (
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestTiDBConfig_DriverName(t *testing.T) {
	cfg := &config.TiDBConfig{}
	if cfg.DriverName() != "mysql" {
		t.Errorf("DriverName = %q, want mysql", cfg.DriverName())
	}
}

func TestTiDBConfig_DefaultPort(t *testing.T) {
	cfg := &config.TiDBConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Username: "root", Password: "pass", Database: "test",
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.Port != 4000 {
		t.Errorf("Port = %d, want 4000", cfg.Port)
	}
}

func TestTiDBConfig_DSN(t *testing.T) {
	cfg := &config.TiDBConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 4000, Username: "root", Password: "secret", Database: "test",
			},
		},
		Charset: "utf8mb4",
	}
	dsn := cfg.DSN()
	expected := "root:secret@tcp(localhost:4000)/test?charset=utf8mb4&parseTime=True&loc=Local"
	if dsn != expected {
		t.Errorf("DSN = %q, want %q", dsn, expected)
	}
}

func TestTiDBConfig_Validate_MissingRequired(t *testing.T) {
	cfg := &config.TiDBConfig{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestTiDBConfig_ImplementsDBConfig(t *testing.T) {
	var _ config.DBConfig = (*config.TiDBConfig)(nil)
}

func TestTiDBConfig_Validate_Error(t *testing.T) {
	cfg := &config.TiDBConfig{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestTiDBConfig_DSN_DefaultCharset(t *testing.T) {
	cfg := &config.TiDBConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 4000, Username: "root", Password: "secret", Database: "test",
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
	if cfg.Charset != "utf8mb4" {
		t.Errorf("Charset = %q, want utf8mb4", cfg.Charset)
	}
}
