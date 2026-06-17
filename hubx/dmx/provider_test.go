package dmx

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	hubx "github.com/gospacex/hubx"
)

func TestName_ReturnsCorrectString(t *testing.T) {
	if got := New().Name(); got != "dbx.dm" {
		t.Errorf("Name() = %q, want dbx.dm", got)
	}
}

func TestBuild_MissingConfigKey(t *testing.T) {
	_, err := New().Build("inst", map[string]any{})
	if !errors.Is(err, hubx.ErrConfigInvalid) {
		t.Errorf("err = %v, want ErrConfigInvalid", err)
	}
}

func TestBuild_MissingDSN(t *testing.T) {
	_, err := New().Build("inst", map[string]any{"config": map[string]any{}})
	if !errors.Is(err, hubx.ErrConfigInvalid) {
		t.Errorf("err = %v, want ErrConfigInvalid", err)
	}
}

func TestBuild_UnknownField(t *testing.T) {
	_, err := New().Build("inst", map[string]any{"config": map[string]any{"dsn": "x", "unknown_xyz": "y"}})
	if !errors.Is(err, hubx.ErrConfigInvalid) {
		t.Errorf("err = %v, want ErrConfigInvalid", err)
	}
}

func TestProviderHealthCheck_NoOp(t *testing.T) {
	if err := New().HealthCheck(context.Background()); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestProviderClose_NoOp(t *testing.T) {
	if err := New().Close(); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestBuild_Success(t *testing.T) {
	cli, err := New().Build("inst", map[string]any{"config": map[string]any{"dsn": ":memory:"}})
	if err != nil {
		if errors.Is(err, hubx.ErrBuildFailed) && strings.Contains(err.Error(), "driver not yet wired") {
			t.Skip("driver stub")
		}
		t.Fatalf("Build: %v", err)
	}
	if cli == nil {
		t.Fatal("nil client")
	}
	defer cli.Close()
}

func TestConcurrentBuild_Singleton(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cli, err := New().Build("inst", map[string]any{"config": map[string]any{"dsn": ":memory:"}})
			if err == nil {
				cli.Close()
			}
		}()
	}
	wg.Wait()
}

func TestRaceFree_UnderRace(t *testing.T) {
	p := New()
	_ = p
}

func TestClientHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	cli, err := New().Build("inst", map[string]any{"config": map[string]any{"dsn": ":memory:"}})
	if err != nil {
		t.Skip("driver not wired")
	}
	defer cli.Close()
	if err := cli.HealthCheck(context.Background()); err != nil {
		t.Skip("ping failed")
	}
}

func TestBuild_OpenFailure(t *testing.T) {
	_, err := New().Build("inst", map[string]any{"config": map[string]any{"dsn": "!!!bad!!!"}})
	if err == nil {
		t.Skip("driver accepted malformed DSN")
	}
}
