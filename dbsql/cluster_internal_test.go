package dbsql

import (
	"testing"

	"github.com/gospacex/dbx/config"
	"gorm.io/plugin/dbresolver"
)

// TestSourceConfig_DialectorSource exercises the sourceConfig wrapper that
// adapts a (driver, NodeConfig) pair to the orm.DialectorSource interface.
// Pool/Tracing/Validate are intentionally not part of sourceConfig — those
// settings live on the parent ClusterConfig and are applied by OpenCluster.
func TestSourceConfig_DialectorSource(t *testing.T) {
	src := &sourceConfig{
		driver: "mysql",
		node: config.NodeConfig{
			Host:     "h",
			Port:     3306,
			Username: "u",
			Password: "p",
			Database: "d",
		},
	}
	if src.DriverName() != "mysql" {
		t.Errorf("DriverName = %q, want mysql", src.DriverName())
	}
	if src.DSN() == "" {
		t.Error("DSN() should be non-empty")
	}
}

// TestSourceConfig_DSN_Sqlite covers the sqlite branch of NodeConfig.DSN,
// which is the only branch used by cluster tests on a real in-memory db.
func TestSourceConfig_DSN_Sqlite(t *testing.T) {
	src := &sourceConfig{
		driver: "sqlite",
		node:   config.NodeConfig{Path: ":memory:"},
	}
	if got := src.DSN(); got != ":memory:" {
		t.Errorf("DSN = %q, want :memory:", got)
	}
}

// TestPolicyFromString covers all three branches of the load-balance policy
// mapping. The unknown branch is otherwise unreachable via OpenCluster
// because ClusterConfig.Validate rejects unknown load_balance values.
func TestPolicyFromString(t *testing.T) {
	tests := []struct {
		in   string
		name string
	}{
		{"random", "RandomPolicy"},
		{"round_robin", "RoundRobinPolicy"},
		{"", "RoundRobinPolicy"},
		{"unknown", "RoundRobinPolicy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policyFromString(tt.in)
			if got == nil {
				t.Error("policyFromString returned nil")
			}
		})
	}
	// Sanity: known values should yield the matching concrete type.
	if _, ok := policyFromString("random").(dbresolver.RandomPolicy); !ok {
		t.Error("random should return dbresolver.RandomPolicy")
	}
	rr := policyFromString("round_robin")
	if rr == nil {
		t.Error("round_robin policy should not be nil")
	}
}
