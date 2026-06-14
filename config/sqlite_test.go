package config_test

import (
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestSQLiteConfig_DriverName(t *testing.T) {
	cfg := &config.SQLiteConfig{}
	if cfg.DriverName() != "sqlite" {
		t.Errorf("DriverName = %q, want sqlite", cfg.DriverName())
	}
}

func TestSQLiteConfig_DSN(t *testing.T) {
	cfg := &config.SQLiteConfig{Path: "/data/test.db"}
	if dsn := cfg.DSN(); dsn != "/data/test.db" {
		t.Errorf("DSN = %q, want /data/test.db", dsn)
	}
}

func TestSQLiteConfig_Validate(t *testing.T) {
	cfg := &config.SQLiteConfig{Path: "/data/test.db"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
}

func TestSQLiteConfig_ValidateMissingPath(t *testing.T) {
	cfg := &config.SQLiteConfig{}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should error when path is empty")
	}
}

func TestSQLiteConfig_GetPool(t *testing.T) {
	cfg := &config.SQLiteConfig{}
	if cfg.GetPool() != nil {
		t.Error("GetPool() should return nil by default")
	}
}

func TestSQLiteConfig_GetTracing(t *testing.T) {
	cfg := &config.SQLiteConfig{}
	if cfg.GetTracing() != nil {
		t.Error("GetTracing() should return nil by default")
	}
}

func TestSQLiteConfig_Interface(t *testing.T) {
	cfg := &config.SQLiteConfig{}
	var _ config.DBConfig = cfg
}
