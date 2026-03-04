-- name: CreateDeadLetterEvent :exec
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
);

-- name: ListUnresolvedDeadLetters :many
SELECT * FROM dead_letter_events
WHERE resolved = FALSE AND next_retry_at <= NOW()
ORDER BY failed_at ASC
LIMIT $1;

-- name: MarkDeadLetterResolved :exec
UPDATE dead_letter_events
SET resolved = TRUE, resolved_at = NOW()
WHERE id = $1;

-- name: IncrementDeadLetterRetry :exec
UPDATE dead_letter_events
SET retry_count = retry_count + 1,
    next_retry_at = NOW() + INTERVAL '5 minutes' * (retry_count + 1)
WHERE id = $1;

-- name: GetDeadLetterEvent :one
SELECT * FROM dead_letter_events
WHERE id = $1;

-- name: ListDeadLetterEvents :many
SELECT * FROM dead_letter_events
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: DeleteDeadLetterEvent :exec
DELETE FROM dead_letter_events
WHERE id = $1;

-- name: CountUnresolvedDeadLetters :one
SELECT COUNT(*) FROM dead_letter_events
WHERE resolved = FALSE;