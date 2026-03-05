// consumer/consumer.go - FINAL PRODUCTION VERSION
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
	_ "github.com/lib/pq" // Register postgres driver
)

// ============================================================================
// PROMETHEUS METRICS (global, registered once via init)
// ============================================================================

var (
	metricsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "consumer_transactions_processed_total",
			Help: "Total number of transactions processed by status",
		},
		[]string{"status"}, // success, failed
	)
	metricsProcessingTime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "consumer_processing_duration_seconds",
			Help:    "Time spent processing a single transaction",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	prometheus.MustRegister(metricsProcessed, metricsProcessingTime)
}

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// TransactionEvent represents the payload from Kafka
type TransactionEvent struct {
	Amount      int64  `json:"amount"`
	ToAccount   int64  `json:"to_account"`
	FromAccount int64  `json:"from_account"`
	TransferID  int64  `json:"transfer_id"`
	Currency    string `json:"currency,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

// ConsumerConfig holds dependencies for Consumer
type ConsumerConfig struct {
	Brokers          []string
	Topic            string
	GroupID          string
	DatabaseDSN      string // Optional: for DLQ to PostgreSQL
	EmailEnabled     bool
	AnalyticsEnabled bool
	PushEnabled      bool
}

// Consumer handles Kafka message consumption
type Consumer struct {
	reader    *kafka.Reader
	db        *sql.DB // Optional: for DLQ
	config    ConsumerConfig
	metrics   *ConsumerMetrics
	mu        sync.RWMutex
	isRunning bool
	logger    *slog.Logger
}

// ConsumerMetrics tracks basic stats (thread-safe)
type ConsumerMetrics struct {
	ProcessedCount int64
	FailedCount    int64
	LastProcessed  time.Time
	mu             sync.RWMutex
}

// DeadLetterEntry represents an event that failed processing
type DeadLetterEntry struct {
	ID              int64            `json:"id,omitempty"`
	OriginalEvent   TransactionEvent `json:"original_event"`
	Error           string           `json:"error"`
	FailedAt        time.Time        `json:"failed_at"`
	ConsumerGroup   string           `json:"consumer_group"`
	Topic           string           `json:"topic"`
	RetryCount      int              `json:"retry_count"`
	MaxRetries      int              `json:"max_retries"`
	NextRetryAt     time.Time        `json:"next_retry_at"`
	ProcessingStage string           `json:"processing_stage"`
	Resolved        bool             `json:"resolved"`
	ResolvedAt      *time.Time       `json:"resolved_at,omitempty"`
}

// ============================================================================
// CONSTRUCTOR
// ============================================================================

// NewConsumer creates a new Kafka consumer instance with dependencies
func NewConsumer(cfg ConsumerConfig) (*Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   cfg.Brokers,
		Topic:     cfg.Topic,
		GroupID:   cfg.GroupID,
		MinBytes:  10e3,
		MaxBytes:  10e6,
		MaxWait:   1 * time.Second,
	})

	c := &Consumer{
		reader:    reader,
		config:    cfg,
		isRunning: false,
		metrics:   &ConsumerMetrics{},
		logger:    slog.Default(),
	}

	// Optional: Initialize database connection for DLQ
	if cfg.DatabaseDSN != "" {
		db, err := sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}
		c.db = db
		c.logger.Info("Database connected for Dead Letter Queue", "dsn", cfg.DatabaseDSN)
	}

	return c, nil
}

// ============================================================================
// MAIN LOOP
// ============================================================================

// Start begins consuming messages from Kafka
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	c.isRunning = true
	c.mu.Unlock()

	c.logger.Info("🚀 Consumer started",
		"topic", c.config.Topic,
		"group", c.config.GroupID,
		"brokers", c.config.Brokers)

	defer func() {
		c.mu.Lock()
		c.isRunning = false
		c.mu.Unlock()
		c.logger.Info("🛑 Consumer stopped", "topic", c.config.Topic)
		if c.db != nil {
			_ = c.db.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			c.logger.Warn("Context cancelled, stopping consumer", "error", ctx.Err())
			return ctx.Err()
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return err
				}
				c.logger.Warn("Fetch error", "error", err)
				time.Sleep(1 * time.Second) // Backoff on transient errors
				continue
			}

			var event TransactionEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				c.logger.Error("Failed to unmarshal event",
					"offset", msg.Offset,
					"error", err,
					"raw_payload", string(msg.Value))
				c.incrementFailed()

				// Commit to avoid poison pill (malformed message)
				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					c.logger.Error("Failed to commit failed message", "error", err)
				}
				continue
			}

			// Process event with comprehensive error handling
			if err := c.processEvent(ctx, event); err != nil {
				c.logger.Error("Failed to process transfer",
					"transfer_id", event.TransferID,
					"error", err)
				c.handleProcessingError(ctx, event, err)
				c.incrementFailed()
			} else {
				c.logger.Info("✅ Processed transfer",
					"transfer_id", event.TransferID,
					"amount", event.Amount,
					"from_account", event.FromAccount,
					"to_account", event.ToAccount)
				c.incrementProcessed()
			}

			// Commit offset after processing (at-least-once semantics)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("Failed to commit message",
					"offset", msg.Offset,
					"error", err)
			}
		}
	}
}

// ============================================================================
// BUSINESS LOGIC
// ============================================================================

// processEvent handles the complete business logic for a transaction event
func (c *Consumer) processEvent(ctx context.Context, event TransactionEvent) error {
	// ✅ Input validation (defensive programming)
	if event.TransferID <= 0 {
		return fmt.Errorf("invalid transfer_id: %d", event.TransferID)
	}
	if event.Amount < 0 {
		return fmt.Errorf("invalid amount: %d", event.Amount)
	}

	// ✅ Track processing time with Prometheus
	start := time.Now()
	defer func() {
		metricsProcessingTime.Observe(time.Since(start).Seconds())
	}()

	c.logger.Debug("Processing event", "event", event)

	// 📧 1. Send email notification (if enabled, non-critical)
	if c.config.EmailEnabled {
		if err := c.sendEmailNotification(ctx, event); err != nil {
			c.logger.Warn("Email notification failed", "transfer_id", event.TransferID, "error", err)
			// Don't return error - email is non-critical
		}
	}

	// 📊 2. Update analytics (if enabled, non-critical)
	if c.config.AnalyticsEnabled {
		if err := c.updateAnalytics(ctx, event); err != nil {
			c.logger.Warn("Analytics update failed", "transfer_id", event.TransferID, "error", err)
			// Don't return error - analytics is non-critical
		}
	}

	// 🔔 3. Send push notification (if enabled, non-critical)
	if c.config.PushEnabled {
		if err := c.sendPushNotification(ctx, event); err != nil {
			c.logger.Warn("Push notification failed", "transfer_id", event.TransferID, "error", err)
			// Don't return error - push is non-critical
		}
	}

	// 🔔 4. Core notification log (always executed)
	c.logger.Info("🔔 [NOTIFICATION] Transfer alert",
		"from_account", event.FromAccount,
		"to_account", event.ToAccount,
		"amount", event.Amount,
		"currency", event.Currency,
		"transfer_id", event.TransferID)

	// ✅ Record success metric
	metricsProcessed.WithLabelValues("success").Inc()
	return nil
}

// ============================================================================
// NOTIFICATION STUBS (implement with your providers)
// ============================================================================

// sendEmailNotification sends email to account owners (stub)
func (c *Consumer) sendEmailNotification(_ context.Context, event TransactionEvent) error {
	c.logger.Info("📧 [EMAIL] Would send notification",
		"transfer_id", event.TransferID,
		"from_account", event.FromAccount,
		"to_account", event.ToAccount,
		"amount", event.Amount)
	// TODO: Implement with SendGrid, AWS SES, Mailgun, etc.
	return nil
}

// updateAnalytics updates analytics dashboard (stub)
func (c *Consumer) updateAnalytics(_ context.Context, event TransactionEvent) error {
	c.logger.Info("📊 [ANALYTICS] Would update dashboard",
		"transfer_id", event.TransferID,
		"amount", event.Amount,
		"currency", event.Currency,
		"from_account", event.FromAccount,
		"to_account", event.ToAccount)
	// TODO: Implement with your analytics service
	return nil
}

// sendPushNotification sends mobile push notification (stub)
func (c *Consumer) sendPushNotification(_ context.Context, event TransactionEvent) error {
	c.logger.Info("🔔 [PUSH] Would send mobile notification",
		"transfer_id", event.TransferID,
		"accounts", []int64{event.FromAccount, event.ToAccount})
	// TODO: Implement with FCM, APNs, OneSignal, etc.
	return nil
}

// ============================================================================
// ERROR HANDLING & DEAD LETTER QUEUE
// ============================================================================

// handleProcessingError handles failed event processing by sending to Dead Letter Queue
func (c *Consumer) handleProcessingError(ctx context.Context, event TransactionEvent, err error) {
	dlqEntry := DeadLetterEntry{
		OriginalEvent:   event,
		Error:           err.Error(),
		FailedAt:        time.Now().UTC(),
		ConsumerGroup:   c.config.GroupID,
		Topic:           c.config.Topic,
		RetryCount:      0,
		MaxRetries:      3,
		NextRetryAt:     time.Now().Add(5 * time.Minute),
		ProcessingStage: "consumer_process_event",
	}

	c.logger.Warn("🗑️ Sending event to Dead Letter Queue",
		"transfer_id", event.TransferID,
		"error", err)

	// Try to persist to database DLQ (if DB connected)
	if c.db != nil {
		if dlqErr := c.saveToDeadLetterQueue(ctx, dlqEntry); dlqErr != nil {
			c.logger.Error("❌ Failed to save to database DLQ", "error", dlqErr)
		} else {
			c.logger.Info("✅ Saved to database Dead Letter Queue", "transfer_id", event.TransferID)
		}
	}

	// ✅ Record failed metric (regardless of DLQ save success)
	metricsProcessed.WithLabelValues("failed").Inc()
}

// saveToDeadLetterQueue persists failed event to PostgreSQL dead_letter_events table
// ✅ Uses context timeout to prevent hanging queries
func (c *Consumer) saveToDeadLetterQueue(ctx context.Context, entry DeadLetterEntry) error {
	if c.db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// ✅ Add timeout for DB operation (production best practice)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	eventJSON, err := json.Marshal(entry.OriginalEvent)
	if err != nil {
		return fmt.Errorf("marshal original_event: %w", err)
	}

	// Raw SQL query - parameterized to prevent SQL injection
	query := `
		INSERT INTO dead_letter_events (
			aggregate_type,
			aggregate_id,
			event_type,
			payload,
			error_message,
			failed_at,
			consumer_group,
			topic,
			retry_count,
			max_retries,
			next_retry_at,
			processing_stage,
			resolved
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	_, err = c.db.ExecContext(ctxWithTimeout, query,
		"transfer",                        // aggregate_type
		entry.OriginalEvent.TransferID,    // aggregate_id
		"transfer_created",                // event_type
		eventJSON,                         // payload (JSONB)
		entry.Error,                       // error_message
		entry.FailedAt,                    // failed_at
		entry.ConsumerGroup,               // consumer_group
		entry.Topic,                       // topic
		entry.RetryCount,                  // retry_count
		entry.MaxRetries,                  // max_retries
		entry.NextRetryAt,                 // next_retry_at
		entry.ProcessingStage,             // processing_stage
		false,                             // resolved
	)

	if err != nil {
		return fmt.Errorf("insert dead_letter_events: %w", err)
	}

	return nil
}

// publishToKafkaDLQ publishes failed event to separate Kafka DLQ topic (stub/alternative)
func (c *Consumer) publishToKafkaDLQ(_ context.Context, entry DeadLetterEntry) error {
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal DLQ entry: %w", err)
	}

	c.logger.Info("🗑️ [KAFKA-DLQ] Would publish",
		"topic", "transaction-events.dlq",
		"transfer_id", entry.OriginalEvent.TransferID,
		"payload_size", len(entryJSON))
	// TODO: Implement with kafka-go Writer for DLQ topic
	return nil
}

// ============================================================================
// METRICS & STATE (thread-safe)
// ============================================================================

// incrementProcessed safely increments processed counter
func (c *Consumer) incrementProcessed() {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()
	c.metrics.ProcessedCount++
	c.metrics.LastProcessed = time.Now()
}

// incrementFailed safely increments failed counter
func (c *Consumer) incrementFailed() {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()
	c.metrics.FailedCount++
}

// GetMetrics returns current consumer metrics (thread-safe)
func (c *Consumer) GetMetrics() ConsumerMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()
	return ConsumerMetrics{
		ProcessedCount: c.metrics.ProcessedCount,
		FailedCount:    c.metrics.FailedCount,
		LastProcessed:  c.metrics.LastProcessed,
	}
}

// IsRunning checks if consumer is actively processing
func (c *Consumer) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

// HealthCheck returns consumer health status (for Kubernetes probes)
func (c *Consumer) HealthCheck() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := map[string]interface{}{
		"status":     "healthy",
		"is_running": c.isRunning,
		"topic":      c.config.Topic,
		"group_id":   c.config.GroupID,
	}

	metrics := c.GetMetrics()
	status["metrics"] = map[string]interface{}{
		"processed_count": metrics.ProcessedCount,
		"failed_count":    metrics.FailedCount,
		"last_processed":  metrics.LastProcessed,
	}

	// Check database connection if initialized
	if c.db != nil {
		if err := c.db.Ping(); err != nil {
			status["status"] = "degraded"
			status["db_error"] = err.Error()
		} else {
			status["database"] = "connected"
		}
	}

	return status
}

// ============================================================================
// CLEANUP
// ============================================================================

// Close gracefully shuts down the consumer
func (c *Consumer) Close() error {
	c.logger.Info("🔌 Closing consumer", "topic", c.config.Topic)
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("close kafka reader: %w", err)
	}
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			c.logger.Warn("Failed to close database connection", "error", err)
		}
	}
	return nil
}