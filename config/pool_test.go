package config_test

import (
	"testing"
	"time"

	"github.com/gospacex/dbx/config"
)

func TestPoolConfig_Defaults(t *testing.T) {
	p := &config.PoolConfig{}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if p.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns = %d, want 50", p.MaxOpenConns)
	}
	if p.MaxIdleConns != 10 {
		t.Errorf("MaxIdleConns = %d, want 10", p.MaxIdleConns)
	}
	if p.ConnMaxLifetime != 1800 {
		t.Errorf("ConnMaxLifetime = %d, want 1800", p.ConnMaxLifetime)
	}
	if p.ConnMaxIdleTime != 600 {
		t.Errorf("ConnMaxIdleTime = %d, want 600", p.ConnMaxIdleTime)
	}
}

func TestPoolConfig_IdleCap(t *testing.T) {
	p := &config.PoolConfig{MaxOpenConns: 5, MaxIdleConns: 20}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if p.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns = %d, want 5 (capped by MaxOpenConns)", p.MaxIdleConns)
	}
}

func TestPoolConfig_CustomValues(t *testing.T) {
	p := &config.PoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    25,
		ConnMaxLifetime: 3600,
		ConnMaxIdleTime: 1200,
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if p.MaxOpenConns != 100 || p.ConnMaxLifetime != 3600 {
		t.Error("custom values should be preserved")
	}
}

func TestPoolConfig_DurationConversion(t *testing.T) {
	p := &config.PoolConfig{ConnMaxLifetime: 30, ConnMaxIdleTime: 15}
	p.Validate()
	if p.ConnMaxLifetimeDuration() != 30*time.Second {
		t.Errorf("ConnMaxLifetimeDuration = %v, want 30s", p.ConnMaxLifetimeDuration())
	}
	if p.ConnMaxIdleTimeDuration() != 15*time.Second {
		t.Errorf("ConnMaxIdleTimeDuration = %v, want 15s", p.ConnMaxIdleTimeDuration())
	}
}
