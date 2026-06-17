//go:build integration

package postgresx

import (
	"context"
	"os"
	"testing"
)

func TestIntegration_BuildAndPing(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" { t.Skip() }
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" { t.Skip("set POSTGRES_DSN") }
	cli, err := New().Build("it", map[string]any{"config": map[string]any{"dsn": dsn}})
	if err != nil { t.Fatal(err) }
	defer cli.Close()
	if err := cli.HealthCheck(context.Background()); err != nil { t.Fatal(err) }
}
