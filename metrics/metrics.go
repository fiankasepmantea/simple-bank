package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Outbox events metrics
	OutboxEventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "outbox_events_processed_total",
			Help: "Total number of outbox events processed",
		},
		[]string{"event_type", "status"}, // status: success, failed, dead_letter
	)

	OutboxProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "outbox_processing_duration_seconds",
			Help:    "Duration of outbox event processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"event_type"},
	)

	// Kafka publish metrics
	KafkaPublishAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_publish_attempts_total",
			Help: "Total number of Kafka publish attempts",
		},
		[]string{"topic", "status"}, // status: success, retry, failed
	)
)

// RegisterMetrics registers all metrics with the default registry
// (optional if using promauto)
func RegisterMetrics() {
	// promauto already registers automatically
}