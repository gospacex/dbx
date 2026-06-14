package main

import (
	"context"
	"fmt"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/dbsql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// mqxTraceConfig is the YAML-friendly trace configuration. Field names
// match the mqx standard so users can copy-paste trace blocks from any
// mqx-based service without rewriting the schema.
type mqxTraceConfig struct {
	Enabled     bool    `yaml:"enabled" json:"enabled"`
	ServiceName string  `yaml:"service_name" json:"service_name"`
	Exporter    string  `yaml:"exporter" json:"exporter"`
	Endpoint    string  `yaml:"endpoint" json:"endpoint"`
	Protocol    string  `yaml:"protocol" json:"protocol"`
	SamplerType string  `yaml:"sampler_type" json:"sampler_type"`
	SamplerRatio float64 `yaml:"sampler_ratio" json:"sampler_ratio"`
	// Mode is mqx standard: single | cluster. Drives redisx.POS vs
	// redisx.POC. Kafka ignores it (kafkax.POS handles multi-broker
	// via the Addrs array, no POC variant exists).
	Mode string `yaml:"mode,omitempty" json:"mode,omitempty"`

	// Kafka-specific
	Topic                string `yaml:"topic,omitempty" json:"topic,omitempty"`
	KafkaUsername        string `yaml:"kafka_username,omitempty" json:"kafka_username,omitempty"`
	KafkaPassword        string `yaml:"kafka_password,omitempty" json:"kafka_password,omitempty"`
	KafkaSecurityProtocol string `yaml:"kafka_security_protocol,omitempty" json:"kafka_security_protocol,omitempty"`
	KafkaSASLMechanism   string `yaml:"kafka_sasl_mechanism,omitempty" json:"kafka_sasl_mechanism,omitempty"`

	// Redis-specific
	Stream        string `yaml:"stream,omitempty" json:"stream,omitempty"`
	RedisPassword string `yaml:"redis_password,omitempty" json:"redis_password,omitempty"`

	// Optional OTLP headers
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// mqxTraceToDBX converts the mqx-style trace config into the internal
// config.TracingConfig understood by dbx. The mapping is intentionally
// explicit so the difference between the two schemas is documented in
// one place.
//
//	mqx yaml              dbx internal
//	-----------           --------------
//	service_name     →   Service
//	protocol: http   →   Protocol: "http/protobuf"
//	sampler_type:    →   SamplerType: "parentbased_<x>"
//	    always_on           parentbased_always_on
//	    always_off          parentbased_always_off
//	    traceidratio        parentbased_traceidratio
//	mode:            →   Mode (single|cluster, validated in dbx)
//	topic / stream / kafka_* / redis_*   →   verbatim
func mqxTraceToDBX(m *mqxTraceConfig) *config.TracingConfig {
	if m == nil {
		return nil
	}
	tc := &config.TracingConfig{
		Enabled:              m.Enabled,
		Service:              m.ServiceName,
		Exporter:             m.Exporter,
		Endpoint:             m.Endpoint,
		Protocol:             translateProtocol(m.Protocol),
		SamplerType:          translateSamplerType(m.SamplerType),
		SamplerRatio:         m.SamplerRatio,
		Mode:                 m.Mode,
		Topic:                m.Topic,
		KafkaUsername:        m.KafkaUsername,
		KafkaPassword:        m.KafkaPassword,
		KafkaSecurityProtocol: m.KafkaSecurityProtocol,
		KafkaSASLMechanism:   m.KafkaSASLMechanism,
		Stream:               m.Stream,
		RedisPassword:        m.RedisPassword,
	}
	return tc
}

// translateProtocol maps the mqx shorthand `http` to dbx's
// OTLP-over-HTTP/protobuf transport. Any other value passes through
// unchanged so advanced users can override.
func translateProtocol(p string) string {
	if p == "http" {
		return "http/protobuf"
	}
	return p
}

// translateSamplerType converts the mqx sampler shorthand into the
// parent-based form expected by dbx. OTel recommends parentbased
// variants so the local sampler respects the upstream sampling decision.
func translateSamplerType(s string) string {
	switch s {
	case "always_on":
		return "parentbased_always_on"
	case "always_off":
		return "parentbased_always_off"
	case "traceidratio":
		return "parentbased_traceidratio"
	}
	return s
}

// setupTracerProvider initializes an OTel TracerProvider for the given
// dbx trace config and registers it as the global provider. Returns
// a shutdown closure that callers MUST defer. When tracing is disabled
// the returned closure is a no-op.
//
// All exporter construction goes through dbsql.CreateExporter so the
// example never re-implements jaeger / kafka / redis wiring.
func setupTracerProvider(ctx context.Context, tc *config.TracingConfig) (func(context.Context) error, error) {
	if tc == nil || !tc.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	exp, err := dbsql.CreateExporter(ctx, tc)
	if err != nil {
		return nil, fmt.Errorf("tracing: create exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(tc.Service),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: build resource: %w", err)
	}

	sampler := parentBasedSampler(tc)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func(ctx context.Context) error {
		// Order is fixed: shutdown first to stop new spans, then
		// force flush to drain anything buffered by the batcher.
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("tracing: tp.Shutdown: %w", err)
		}
		if err := tp.ForceFlush(ctx); err != nil {
			return fmt.Errorf("tracing: tp.ForceFlush: %w", err)
		}
		return nil
	}, nil
}

// parentBasedSampler returns a Sampler that respects the upstream
// sampling decision. The ratio is forwarded to TraceIDRatioBased so
// ratio-driven configs still take effect.
func parentBasedSampler(tc *config.TracingConfig) sdktrace.Sampler {
	switch tc.SamplerType {
	case "parentbased_always_off", "always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_traceidratio", "traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(tc.SamplerRatio))
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}
