package dbsql

import (
	"context"
	"fmt"
	"strings"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/observability/tracing"
	"github.com/gospacex/mqx"
	"github.com/gospacex/mqx/kafkax"
	"github.com/gospacex/mqx/redisx"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// CreateExporter creates a span exporter based on TracingConfig.
// For Kafka and Redis exporters, the connection is obtained from mqx-managed pools
// (kafkax.POS / redisx.POS or redisx.POC), so lifecycle is delegated to mqx.
//
// Mode drives the producer topology:
//   - kafka: kafkax.POS always — kafka clients support multi-broker via
//     the Addrs array, so single POS covers both single and cluster brokers.
//   - redis_stream: redisx.POS (single → *redis.Client) or redisx.POC
//     (cluster → *redis.ClusterClient). The two return types differ, hence
//     the branch.
func CreateExporter(ctx context.Context, tc *config.TracingConfig) (sdktrace.SpanExporter, error) {
	switch tc.Exporter {
	case config.ExporterJaeger:
		exp, err := newJaegerHTTPExporter(ctx, tc.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("dbsql: jaeger exporter: %w", err)
		}
		return tracing.NewMQXSpanExporter(exp, nil, nil), nil
	case config.ExporterKafka:
		producer, err := kafkax.POS(buildKafkaConfig(tc))
		if err != nil {
			return nil, fmt.Errorf("dbsql: kafkax.POS: %w", err)
		}
		ka := tracing.NewKafkaExporter(producer, tc.Topic)
		return tracing.NewMQXSpanExporter(nil, ka, nil), nil
	case config.ExporterRedisStream:
		var adder tracing.RedisXAdder
		switch tc.Mode {
		case config.ModeCluster:
			c, err := redisx.POC(buildRedisConfig(tc))
			if err != nil {
				return nil, fmt.Errorf("dbsql: redisx.POC: %w", err)
			}
			adder = c
		default:
			c, err := redisx.POS(buildRedisConfig(tc))
			if err != nil {
				return nil, fmt.Errorf("dbsql: redisx.POS: %w", err)
			}
			adder = c
		}
		re := tracing.NewRedisExporter(adder, tc.Stream)
		return tracing.NewMQXSpanExporter(nil, nil, re), nil
	default:
		return nil, fmt.Errorf("dbsql: unsupported exporter %q", tc.Exporter)
	}
}

// buildKafkaConfig maps TracingConfig to an mqx.Config consumable by kafkax.POS.
func buildKafkaConfig(tc *config.TracingConfig) mqx.Config {
	brokers := strings.Split(tc.Endpoint, ",")
	return mqx.Config{
		Driver: "kafka",
		Addrs:  brokers,
		Producer: mqx.ProducerConfig{
			Topic: tc.Topic,
		},
		Auth: mqx.AuthConfig{
			Username: tc.KafkaUsername,
			Password: tc.KafkaPassword,
		},
		Kafka: &mqx.KafkaConfig{
			SecurityProtocol: tc.KafkaSecurityProtocol,
			SASLMechanism:    tc.KafkaSASLMechanism,
		},
	}
}

// buildRedisConfig maps TracingConfig to an mqx.Config consumable by
// redisx.POS (single) or redisx.POC (cluster). Endpoints may be
// comma-separated so the same field can hold one node (single) or many
// cluster nodes — POS picks Addrs[0], POC uses them all.
func buildRedisConfig(tc *config.TracingConfig) mqx.Config {
	addrs := strings.Split(tc.Endpoint, ",")
	return mqx.Config{
		Driver: "redis",
		Mode:   tc.Mode,
		Addrs:  addrs,
		Auth: mqx.AuthConfig{
			Password: tc.RedisPassword,
		},
	}
}

// KafkaConfigFromTracing maps TracingConfig to Kafka connection settings.
// Exported for testing; called internally by buildKafkaConfig.
func KafkaConfigFromTracing(tc *config.TracingConfig) (brokers []string, defaultTopic string) {
	return strings.Split(tc.Endpoint, ","), tc.Topic
}

// RedisConfigFromTracing maps TracingConfig to Redis connection settings.
// Exported for testing; called internally by buildRedisConfig.
func RedisConfigFromTracing(tc *config.TracingConfig) (addr, password string) {
	return tc.Endpoint, tc.RedisPassword
}

// ExtractTracingAndApply is a no-op kept for backward compatibility.
//
// dbx does NOT own the OTel global state: the caller is expected to
// set up a TracerProvider themselves (via the OTel SDK, or by calling
// dbsql.CreateExporter and wiring it into a TracerProvider). Calling
// CreateExporter here would leak a second kafka/redis connection
// when the example (or any other caller) has already done so.
//
// The TracingConfig on DBConfig / ClusterConfig is still validated by
// Validate() at config-load time, so misconfigurations still surface
// before any queries run.
func ExtractTracingAndApply(ctx context.Context, tc *config.TracingConfig) error {
	_ = ctx
	_ = tc
	return nil
}

// newJaegerHTTPExporter builds an otlptracehttp exporter for the jaeger
// case. The otlptracehttp client defaults to TLS, so a bare `host:port`
// endpoint would otherwise be sent over HTTPS — which is wrong for the
// common local Jaeger all-in-one setup that listens on plain HTTP at
// :4318. We split on scheme so both forms are accepted:
//
//	"localhost:4318"        → WithEndpoint + WithInsecure   (plain HTTP)
//	"http://localhost:4318" → WithEndpointURL               (plain HTTP, explicit)
//	"https://otel.example"  → WithEndpointURL               (TLS, production)
//
// Without this, GORM spans are created in the SDK but the BatchSpanProcessor
// drops them at the TLS handshake — explaining the "CRUD SQL not in Jaeger"
// report when running against a default jaeger-all-in-one container.
func newJaegerHTTPExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	switch {
	case strings.HasPrefix(endpoint, "http://"), strings.HasPrefix(endpoint, "https://"):
		return otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	default:
		return otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(),
		)
	}
}
