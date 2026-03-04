package kf

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Publisher interface for Kafka
type Publisher interface {
	Publish(ctx context.Context, topic string, key string, value []byte) error
	Close() error
}

// KafkaPublisher implements Publisher
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher creates a new Kafka publisher
func NewKafkaPublisher(brokers []string) (*KafkaPublisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("at least one broker is required")
	}

	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			// Topic is set per-message in Publish(), so leave default empty
			Balancer:               &kafka.LeastBytes{},
			BatchTimeout:           10 * time.Millisecond,
			BatchSize:              100,
			Async:                  true,
			AllowAutoTopicCreation: true, // ← ADD THIS for dev
		},
	}, nil
}

// Publish sends a message to the specified Kafka topic
func (k *KafkaPublisher) Publish(ctx context.Context, topic string, key string, value []byte) error {
	return k.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	})
}

// Close closes the Kafka writer
func (k *KafkaPublisher) Close() error {
	if k.writer != nil {
		return k.writer.Close()
	}
	return nil
}