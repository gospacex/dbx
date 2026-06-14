package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type fakeKafkaProducer struct {
	calls    int
	produce  func(msg *kafka.Message) error
	gotTopic string
	gotValue []byte
}

func (f *fakeKafkaProducer) Produce(msg *kafka.Message, _ chan kafka.Event) error {
	f.calls++
	if f.produce != nil {
		err := f.produce(msg)
		if err == nil {
			f.gotTopic = *msg.TopicPartition.Topic
			f.gotValue = msg.Value
		}
		return err
	}
	f.gotTopic = *msg.TopicPartition.Topic
	f.gotValue = msg.Value
	return nil
}

func TestKafkaExporter_ExportSpans_Success(t *testing.T) {
	prod := &fakeKafkaProducer{}
	exp := &KafkaExporter{producer: prod, topic: "traces"}

	span := makeReadOnlySpan(t, "k1")
	if err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{span}); err != nil {
		t.Fatalf("ExportSpans() error: %v", err)
	}
	if prod.calls != 1 {
		t.Errorf("Produce calls = %d, want 1", prod.calls)
	}
	if prod.gotTopic != "traces" {
		t.Errorf("topic = %q, want traces", prod.gotTopic)
	}
	if len(prod.gotValue) == 0 {
		t.Error("value should be non-empty")
	}
}

func TestKafkaExporter_ExportSpans_ProduceError(t *testing.T) {
	wantErr := errors.New("produce failed")
	prod := &fakeKafkaProducer{produce: func(_ *kafka.Message) error { return wantErr }}
	exp := &KafkaExporter{producer: prod, topic: "traces"}

	span := makeReadOnlySpan(t, "k2")
	err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{span})
	if err == nil {
		t.Fatal("expected error from ExportSpans")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error chain missing produce error: %v", err)
	}
}

func TestKafkaExporter_ExportSpans_EmptySpans(t *testing.T) {
	prod := &fakeKafkaProducer{}
	exp := &KafkaExporter{producer: prod, topic: "traces"}
	if err := exp.ExportSpans(context.Background(), nil); err != nil {
		t.Errorf("ExportSpans(nil) error: %v", err)
	}
	if prod.calls != 0 {
		t.Errorf("Produce should not be called for empty spans, got %d", prod.calls)
	}
}
