CREATE TABLE IF NOT EXISTS dead_letter_events (
    id BIGSERIAL PRIMARY KEY,
    aggregate_type VARCHAR NOT NULL,    
    aggregate_id BIGINT NOT NULL,      
    event_type VARCHAR NOT NULL,
    payload JSONB NOT NULL,
    error_message TEXT NOT NULL,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    consumer_group VARCHAR NOT NULL,
    topic VARCHAR NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    next_retry_at TIMESTAMPTZ NOT NULL,
    processing_stage VARCHAR NOT NULL,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dead_letter_unresolved ON dead_letter_events(resolved, next_retry_at) WHERE resolved = FALSE;
CREATE INDEX idx_dead_letter_aggregate ON dead_letter_events(aggregate_type, aggregate_id);
CREATE INDEX idx_dead_letter_failed_at ON dead_letter_events(failed_at);

COMMENT ON TABLE dead_letter_events IS 'Events that failed processing and need manual review or retry';