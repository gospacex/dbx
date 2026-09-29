package dbsql

import (
	"testing"

	"github.com/gospacex/dbx/config"
)

func TestBuildKafkaConfig(t *testing.T) {
	tc := &config.TracingConfig{
		Endpoint:              "broker1:9092,broker2:9092",
		Topic:                 "my-traces",
		KafkaUsername:         "user",
		KafkaPassword:         "pass",
		KafkaSecurityProtocol: "SASL_PLAINTEXT",
		KafkaSASLMechanism:    "PLAIN",
	}
	mc := buildKafkaConfig(tc)
	if mc.Driver != "kafka" {
		t.Errorf("Driver = %q, want kafka", mc.Driver)
	}
	if len(mc.Addrs) != 2 || mc.Addrs[0] != "broker1:9092" || mc.Addrs[1] != "broker2:9092" {
		t.Errorf("Addrs = %v, want [broker1:9092 broker2:9092]", mc.Addrs)
	}
	if mc.Producer.Topic != "my-traces" {
		t.Errorf("Topic = %q, want my-traces", mc.Producer.Topic)
	}
	if mc.Auth.Username != "user" || mc.Auth.Password != "pass" {
		t.Errorf("Auth = %+v, want user/pass", mc.Auth)
	}
	if mc.Kafka == nil || mc.Kafka.SecurityProtocol != "SASL_PLAINTEXT" || mc.Kafka.SASLMechanism != "PLAIN" {
		t.Errorf("Kafka = %+v, want SASL_PLAINTEXT/PLAIN", mc.Kafka)
	}
}

func TestBuildRedisConfig(t *testing.T) {
	tc := &config.TracingConfig{
		Endpoint:      "node1:6379,node2:6379",
		Mode:          config.ModeCluster,
		RedisPassword: "secret",
	}
	mc := buildRedisConfig(tc)
	if mc.Driver != "redis" {
		t.Errorf("Driver = %q, want redis", mc.Driver)
	}
	if mc.Mode != config.ModeCluster {
		t.Errorf("Mode = %q, want %q", mc.Mode, config.ModeCluster)
	}
	if len(mc.Addrs) != 2 || mc.Addrs[0] != "node1:6379" {
		t.Errorf("Addrs = %v, want [node1:6379 node2:6379]", mc.Addrs)
	}
	if mc.Auth.Password != "secret" {
		t.Errorf("Password = %q, want secret", mc.Auth.Password)
	}
}
