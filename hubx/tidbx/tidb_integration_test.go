//go:build integration

package tidbx

import (
	"context"
	"os"
	"testing"
)

func TestIntegration_BuildAndPing(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" { t.Skip() }
	dsn := os.Getenv("TIDB_DSN")
	if dsn == "" { t.Skip("set TIDB_DSN") }
	cli, err := New().Build("it", map[string]any{"config": map[string]any{"dsn": dsn}})
	if err != nil { t.Fatal(err) }
	defer cli.Close()
	if err := cli.HealthCheck(context.Background()); err != nil { t.Fatal(err) }
}
