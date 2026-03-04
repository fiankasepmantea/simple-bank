-- name: CreateDeadLetterEvent :exec
INSERT INTO dead_letter_events (
    original_event_id,
    event_type,
    payload,
    error_message,
    retry_count
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: ListDeadLetterEvents :many
SELECT * FROM dead_letter_events
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: DeleteDeadLetterEvent :exec
DELETE FROM dead_letter_events
WHERE id = $1;