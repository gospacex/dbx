package config_test

import (
	"strings"
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestGaussDBConfig_DriverName(t *testing.T) {
	cfg := &config.GaussDBConfig{}
	if cfg.DriverName() != "postgres" {
		t.Errorf("DriverName = %q, want postgres", cfg.DriverName())
	}
}

func TestGaussDBConfig_Defaults(t *testing.T) {
	cfg := &config.GaussDBConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Username: "gauss", Password: "pass", Database: "test",
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
}

func TestGaussDBConfig_DSN(t *testing.T) {
	cfg := &config.GaussDBConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 5432, Username: "gauss", Password: "pass", Database: "test",
			},
		},
		SSLMode: "disable",
	}
	dsn := cfg.DSN()
	if dsn == "" {
		t.Fatal("DSN should not be empty")
	}
	if strings.Contains(dsn, "TimeZone") {
		t.Error("GaussDB DSN should not contain TimeZone")
	}
}

func TestGaussDBConfig_DSNWithSchema(t *testing.T) {
	cfg := &config.GaussDBConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Host: "localhost", Port: 5432, Username: "gauss", Password: "pass", Database: "test",
			},
		},
		SSLMode: "disable", Schema: "myschema",
	}
	dsn := cfg.DSN()
	if !strings.Contains(dsn, "search_path=myschema") {
		t.Errorf("DSN should contain search_path, got: %s", dsn)
	}
}

func TestGaussDBConfig_ValidateMissingHost(t *testing.T) {
	cfg := &config.GaussDBConfig{
		BaseDBConfig: config.BaseDBConfig{
			CommonNetworkConfig: config.CommonNetworkConfig{
				Username: "gauss",
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should error when host is empty")
	}
}

func TestGaussDBConfig_Interface(t *testing.T) {
	cfg := &config.GaussDBConfig{}
	var _ config.DBConfig = cfg
}
