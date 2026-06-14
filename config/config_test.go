package config_test

import (
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestCommonNetworkConfig_Fields(t *testing.T) {
	c := config.CommonNetworkConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "secret",
		Database: "testdb",
	}
	if c.Host != "localhost" || c.Port != 3306 {
		t.Error("CommonNetworkConfig fields should be accessible")
	}
}

func TestBaseDBConfig_Embedding(t *testing.T) {
	b := config.BaseDBConfig{}
	if b.Pool != nil {
		t.Error("Pool should be nil by default")
	}
	if b.Tracing != nil {
		t.Error("Tracing should be nil by default")
	}
}
