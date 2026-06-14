package config

import "fmt"

const (
	ExporterJaeger      = "jaeger"
	ExporterKafka       = "kafka"
	ExporterRedisStream = "redis_stream"

	// Mode selects the mqx producer object topology.
	//   single  → mqx.POS  (single broker / single node)
	//   cluster → mqx.POC  (cluster mode; only meaningful for redis_stream —
	//              kafkax has no POC variant, the kafka client itself
	//              supports multiple brokers via the Addrs array)
	ModeSingle  = "single"
	ModeCluster = "cluster"
)

// TracingConfig holds OpenTelemetry tracing export configuration.
//
// Field names follow the mqx standard where overlap exists (e.g. `topic`,
// `stream`, `endpoint`, `sampler_type`). The translation from the user-facing
// mqx-style yaml to this internal struct happens in dbsql/tracer.go.
type TracingConfig struct {
	Enabled      bool    `yaml:"enabled" json:"enabled" toml:"enabled"`
	Service      string  `yaml:"service" json:"service" toml:"service"`
	Exporter     string  `yaml:"exporter" json:"exporter" toml:"exporter"`
	Endpoint     string  `yaml:"endpoint" json:"endpoint" toml:"endpoint"`
	Protocol     string  `yaml:"protocol" json:"protocol" toml:"protocol"`
	SamplerType  string  `yaml:"sampler_type" json:"sampler_type" toml:"sampler_type"`
	SamplerRatio float64 `yaml:"sampler_ratio" json:"sampler_ratio" toml:"sampler_ratio"`
	// Mode selects single (POS) vs cluster (POC) for the underlying mqx
	// producer. Only `redis_stream` actually branches on Mode — kafka's
	// client supports multiple brokers natively, so kafkax.POS is always
	// used regardless of Mode. The field is accepted for all exporters so
	// yaml schema is uniform.
	Mode string `yaml:"mode,omitempty" json:"mode,omitempty" toml:"mode,omitempty"`
	// Kafka-specific
	Topic                string `yaml:"topic" json:"topic" toml:"topic"`
	KafkaUsername        string `yaml:"kafka_username,omitempty" json:"kafka_username,omitempty" toml:"kafka_username,omitempty"`
	KafkaPassword        string `yaml:"kafka_password,omitempty" json:"-" toml:"kafka_password,omitempty"`
	KafkaSecurityProtocol string `yaml:"kafka_security_protocol,omitempty" json:"kafka_security_protocol,omitempty" toml:"kafka_security_protocol,omitempty"`
	KafkaSASLMechanism   string `yaml:"kafka_sasl_mechanism,omitempty" json:"kafka_sasl_mechanism,omitempty" toml:"kafka_sasl_mechanism,omitempty"`
	// Redis-specific
	Stream        string `yaml:"stream" json:"stream" toml:"stream"`
	RedisPassword string `yaml:"redis_password" json:"-" toml:"redis_password"`
}

// Validate sets defaults and validates TracingConfig.
func (tc *TracingConfig) Validate() error {
	if tc.Service == "" {
		tc.Service = "dbx"
	}
	if tc.SamplerType == "" {
		tc.SamplerType = "parentbased_always_on"
	}
	if tc.SamplerRatio <= 0 {
		tc.SamplerRatio = 1.0
	}
	// Mode defaults to single — see field doc on Mode for why this is
	// accepted for all exporters even though only redis_stream branches on it.
	if tc.Mode == "" {
		tc.Mode = ModeSingle
	}
	if tc.Mode != ModeSingle && tc.Mode != ModeCluster {
		return fmt.Errorf("tracing: invalid mode %q (want %q or %q)",
			tc.Mode, ModeSingle, ModeCluster)
	}

	switch tc.Exporter {
	case "":
		tc.Exporter = ExporterJaeger
		fallthrough
	case ExporterJaeger:
		if tc.Endpoint == "" {
			tc.Endpoint = "localhost:4318"
		}
		if tc.Protocol == "" {
			tc.Protocol = "http/protobuf"
		}
	case ExporterKafka:
		if tc.Endpoint == "" {
			tc.Endpoint = "localhost:9092"
		}
		if tc.Topic == "" {
			tc.Topic = "otel-traces"
		}
	case ExporterRedisStream:
		if tc.Endpoint == "" {
			tc.Endpoint = "localhost:6379"
		}
		if tc.Stream == "" {
			tc.Stream = "otel-traces"
		}
	default:
		return fmt.Errorf("tracing: unsupported exporter %q", tc.Exporter)
	}
	return nil
}
