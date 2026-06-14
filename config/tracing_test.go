package config_test

import (
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestTracingConfig_Defaults_Kafka(t *testing.T) {
	tc := &config.TracingConfig{Exporter: "kafka"}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if tc.Endpoint != "localhost:9092" {
		t.Errorf("Endpoint = %q, want localhost:9092", tc.Endpoint)
	}
	if tc.Topic != "otel-traces" {
		t.Errorf("Topic = %q, want otel-traces", tc.Topic)
	}
}

func TestTracingConfig_Defaults_Redis(t *testing.T) {
	tc := &config.TracingConfig{Exporter: "redis_stream"}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if tc.Endpoint != "localhost:6379" {
		t.Errorf("Endpoint = %q, want localhost:6379", tc.Endpoint)
	}
	if tc.Stream != "otel-traces" {
		t.Errorf("Stream = %q, want otel-traces", tc.Stream)
	}
}

func TestTracingConfig_Defaults_Jaeger(t *testing.T) {
	tc := &config.TracingConfig{Exporter: "jaeger"}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if tc.Endpoint != "localhost:4318" {
		t.Errorf("Endpoint = %q, want localhost:4318", tc.Endpoint)
	}
}

func TestTracingConfig_DefaultExporter(t *testing.T) {
	tc := &config.TracingConfig{}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if tc.Exporter != "jaeger" {
		t.Errorf("Exporter = %q, want jaeger (default)", tc.Exporter)
	}
}

func TestTracingConfig_InvalidExporter(t *testing.T) {
	tc := &config.TracingConfig{Exporter: "invalid"}
	if err := tc.Validate(); err == nil {
		t.Fatal("expected error for invalid exporter")
	}
}

func TestTracingConfig_CustomValues(t *testing.T) {
	tc := &config.TracingConfig{
		Enabled:      true,
		Service:      "my-service",
		Exporter:     "kafka",
		Endpoint:     "broker1:9092,broker2:9092",
		Topic:        "my-traces",
		SamplerType:  "parentbased_traceidratio",
		SamplerRatio: 0.5,
	}
	if err := tc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if tc.Endpoint != "broker1:9092,broker2:9092" {
		t.Error("custom endpoint should be preserved")
	}
}
