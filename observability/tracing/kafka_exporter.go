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
}

// KafkaExporter exports spans via Kafka using a confluent producer.
type KafkaExporter struct {
	producer kafkaProducer
	topic    string
}

// NewKafkaExporter creates a KafkaExporter that publishes serialized spans
// to the given Kafka topic using the provided producer.
func NewKafkaExporter(producer *kafka.Producer, topic string) *KafkaExporter {
	return &KafkaExporter{producer: producer, topic: topic}
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

// Shutdown is a no-op; mqx manages the producer lifecycle.
func (e *KafkaExporter) Shutdown(ctx context.Context) error {
	return nil
}