package tracing

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// kafkaProducer is the minimal contract KafkaExporter needs from a producer.
// Using an interface keeps tests mockable while *kafka.Producer remains the
// production implementation.
type kafkaProducer interface {
	Produce(msg *kafka.Message, dr chan kafka.Event) error
	Flush(timeoutMs int) int
}

// KafkaExporter exports spans via Kafka using a confluent producer.
type KafkaExporter struct {
	producer kafkaProducer
	topic    string
}

// NewKafkaExporter creates a KafkaExporter that publishes serialized spans
// to the given Kafka topic using the provided producer. A nil producer is
// stored as a nil interface so Shutdown can detect it without a typed-nil
// panic.
func NewKafkaExporter(producer *kafka.Producer, topic string) *KafkaExporter {
	var p kafkaProducer
	if producer != nil {
		p = producer
	}
	return &KafkaExporter{producer: p, topic: topic}
}

// ExportSpans marshals each span to JSON and produces it to Kafka.
func (e *KafkaExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		data, err := marshalSpanJSON(span)
		if err != nil {
			return fmt.Errorf("kafka: marshal span: %w", err)
		}
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &e.topic},
			Value:          data,
		}
		if err := e.producer.Produce(msg, nil); err != nil {
			return fmt.Errorf("kafka: produce trace: %w", err)
		}
	}
	return nil
}

// flushTimeoutMs bounds how long Shutdown waits for the producer queue
// to drain before giving up.
const flushTimeoutMs = 10 * 1000

// Shutdown flushes the producer's queued messages so spans buffered by
// librdkafka are not lost when the process exits. Closing the producer
// itself remains mqx's responsibility.
func (e *KafkaExporter) Shutdown(ctx context.Context) error {
	if e.producer == nil {
		return nil
	}
	if remaining := e.producer.Flush(flushTimeoutMs); remaining != 0 {
		return fmt.Errorf("kafka: flush: %d message(s) still in queue after %dms", remaining, flushTimeoutMs)
	}
	return nil
}
