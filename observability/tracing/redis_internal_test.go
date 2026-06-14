package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type fakeRedisClient struct {
	calls   int
	xadd    func(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd
	lastArg *redis.XAddArgs
}

func (f *fakeRedisClient) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	f.calls++
	f.lastArg = args
	if f.xadd != nil {
		return f.xadd(ctx, args)
	}
	return redis.NewStringResult("1-0", nil)
}

func TestRedisExporter_ExportSpans_Success(t *testing.T) {
	c := &fakeRedisClient{}
	exp := &RedisExporter{client: c, stream: "stream1"}

	span := makeReadOnlySpan(t, "r1")
	if err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{span}); err != nil {
		t.Fatalf("ExportSpans() error: %v", err)
	}
	if c.calls != 1 {
		t.Errorf("XAdd calls = %d, want 1", c.calls)
	}
	if c.lastArg == nil {
		t.Fatal("lastArg should be set")
	}
	if c.lastArg.Stream != "stream1" {
		t.Errorf("stream = %q, want stream1", c.lastArg.Stream)
	}
	values, ok := c.lastArg.Values.(map[string]any)
	if !ok {
		t.Fatalf("Values is not map[string]any: %T", c.lastArg.Values)
	}
	if _, hasData := values["data"]; !hasData {
		t.Error("XAdd values missing 'data' key")
	}
}

func TestRedisExporter_ExportSpans_XAddError(t *testing.T) {
	wantErr := errors.New("xadd failed")
	c := &fakeRedisClient{
		xadd: func(_ context.Context, _ *redis.XAddArgs) *redis.StringCmd {
			return redis.NewStringResult("", wantErr)
		},
	}
	exp := &RedisExporter{client: c, stream: "stream1"}

	span := makeReadOnlySpan(t, "r2")
	err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{span})
	if err == nil {
		t.Fatal("expected error from ExportSpans")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error chain missing xadd error: %v", err)
	}
}

func TestRedisExporter_ExportSpans_EmptySpans(t *testing.T) {
	c := &fakeRedisClient{}
	exp := &RedisExporter{client: c, stream: "stream1"}
	if err := exp.ExportSpans(context.Background(), nil); err != nil {
		t.Errorf("ExportSpans(nil) error: %v", err)
	}
	if c.calls != 0 {
		t.Errorf("XAdd should not be called for empty spans, got %d", c.calls)
	}
}
