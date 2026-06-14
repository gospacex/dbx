//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// newTracerProvider wires the given exporter into a real, batched
// sdktrace.TracerProvider and registers shutdown on t.Cleanup.
//
// We need a real *TracerProvider (not just an exporter) so that
// TracerProvider.Shutdown forces a batch flush — this proves the
// exporter actually sent data to the downstream system, not just
// that it constructed cleanly.
func newTracerProvider(t *testing.T, exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	t.Helper()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
	)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	return tp
}
