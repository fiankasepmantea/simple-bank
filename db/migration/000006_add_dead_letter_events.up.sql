CREATE TABLE dead_letter_events (
    id BIGSERIAL PRIMARY KEY,
    original_event_id BIGINT NOT NULL,      
    event_type VARCHAR NOT NULL,
    payload JSONB NOT NULL,
    error_message TEXT NOT NULL,            
    retry_count INT NOT NULL DEFAULT 0,     
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dead_letter_created ON dead_letter_events(created_at);
CREATE INDEX idx_dead_letter_original_event ON dead_letter_events(original_event_id);