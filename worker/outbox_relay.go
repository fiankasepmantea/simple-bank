package worker

import (
	"context"
	"log"
	"time"

	db "simple-bank/db/sqlc"
	"simple-bank/kf"
	"simple-bank/metrics"
	"github.com/jackc/pgx/v5/pgtype"
)

type OutboxRelay struct {
	store     db.Store
	publisher kf.Publisher
	topic     string
	interval  time.Duration
}

func NewOutboxRelay(store db.Store, publisher kf.Publisher, topic string, interval time.Duration) *OutboxRelay {
	return &OutboxRelay{
		store:     store,
		publisher: publisher,
		topic:     topic,
		interval:  interval,
	}
}

func (r *OutboxRelay) Start() {
	log.Printf("🚀 OutboxRelay started - polling every %v", r.interval)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		events, err := r.store.ListUnprocessedOutboxEvents(ctx, 10)
		if err != nil {
			log.Printf("❌ failed to fetch outbox events: %v", err)
			continue
		}

		if len(events) == 0 {
			continue
		}

		log.Printf("📦 Found %d unprocessed outbox events", len(events))

		for _, event := range events {
			if err := r.processEvent(ctx, event); err != nil {
				log.Printf("❌ failed to process event %d: %v", event.ID, err)
				continue // retry next tick
			}
		}
	}
}

func (r *OutboxRelay) processEvent(ctx context.Context, event db.OutboxEvent) error {
	start := time.Now()
	defer func() {
		metrics.OutboxProcessingDuration.WithLabelValues(event.EventType).
			Observe(time.Since(start).Seconds())
	}()

	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := r.publisher.Publish(ctx, r.topic, event.EventType, event.Payload)
		if err == nil {
			metrics.KafkaPublishAttempts.WithLabelValues(r.topic, "success").Inc()
			metrics.OutboxEventsProcessed.WithLabelValues(event.EventType, "success").Inc()
			return r.store.MarkOutboxEventProcessed(ctx, event.ID)
		}
		
		lastErr = err
		metrics.KafkaPublishAttempts.WithLabelValues(r.topic, "retry").Inc()
		
		if attempt < maxRetries {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("⚠️  Attempt %d/%d failed for event %d: %v. Retrying in %v...", 
				attempt, maxRetries, event.ID, err, backoff)
			time.Sleep(backoff)
			continue
		}
	}

	// All retries failed → move to dead letter queue
	log.Printf("❌ Event %d failed after %d attempts, moving to dead letter queue", event.ID, maxRetries)
	metrics.KafkaPublishAttempts.WithLabelValues(r.topic, "failed").Inc()
	metrics.OutboxEventsProcessed.WithLabelValues(event.EventType, "dead_letter").Inc()
	
	// ✅ FIX: Convert time.Time → pgtype.Timestamptz
	now := time.Now()
	err := r.store.CreateDeadLetterEvent(ctx, db.CreateDeadLetterEventParams{
		AggregateType:   "transfer",
		AggregateID:     event.AggregateID,
		EventType:       event.EventType,
		Payload:         event.Payload,
		ErrorMessage:    lastErr.Error(),
		FailedAt:        pgtype.Timestamptz{Time: now, Valid: true},              // ✅ FIX 1
		ConsumerGroup:   "outbox-relay",
		Topic:           r.topic,
		RetryCount:      int32(maxRetries),
		MaxRetries:      int32(maxRetries),
		NextRetryAt:     pgtype.Timestamptz{Time: now.Add(1 * time.Hour), Valid: true}, // ✅ FIX 2
		ProcessingStage: "outbox_relay_publish",
		Resolved:        false,
	})
	if err != nil {
		log.Printf("❌ Failed to create dead letter record for event %d: %v", event.ID, err)
		return err
	}
	
	return r.store.MarkOutboxEventProcessed(ctx, event.ID)
}

func (r *OutboxRelay) Stop() {
	log.Println("🛑 Stopping OutboxRelay...")
	if r.publisher != nil {
		r.publisher.Close()
	}
}