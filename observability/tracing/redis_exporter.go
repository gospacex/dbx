package tracing

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// RedisXAdder is the minimal contract RedisExporter needs from a client for
// streaming serialized spans. Using an interface keeps tests mockable while
// *redis.Client and *redis.ClusterClient remain the production
// implementations (POS and POC respectively). Exported so dbsql/tracer.go
// can declare a local variable of this type when branching on Mode.
type RedisXAdder interface {
	XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd
}

// RedisExporter exports spans via Redis Stream using a go-redis client.
type RedisExporter struct {
	client RedisXAdder
	stream string
}

// NewRedisExporter creates a RedisExporter that XADDs serialized spans
// to the given Redis stream using the provided client.
//
// The client is accepted as the RedisXAdder interface so both
// *redis.Client (single node, produced by redisx.POS) and
// *redis.ClusterClient (cluster, produced by redisx.POC) work without
// a separate type — the XAdd method signature is identical on both.
func NewRedisExporter(client RedisXAdder, stream string) *RedisExporter {
	return &RedisExporter{client: client, stream: stream}
}

// ExportSpans marshals each span to JSON and adds it to a Redis stream.
func (e *RedisExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		data, err := marshalSpanJSON(span)
		if err != nil {
			return fmt.Errorf("redis: marshal span: %w", err)
		}
		if err := e.client.XAdd(ctx, &redis.XAddArgs{
			Stream: e.stream,
			Values: map[string]any{"data": data},
		}).Err(); err != nil {
			return fmt.Errorf("redis: xadd trace: %w", err)
		}
	}
	return nil
}

// Shutdown is a no-op; mqx manages the client lifecycle.
func (e *RedisExporter) Shutdown(ctx context.Context) error {
	return nil
}