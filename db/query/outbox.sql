-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (
    aggregate_type,
    aggregate_id,
    event_type,
    payload
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: ListUnprocessedOutboxEvents :many
SELECT * FROM outbox_events
WHERE processed = FALSE
ORDER BY id
LIMIT $1;

-- name: MarkOutboxEventProcessed :exec
UPDATE outbox_events
SET processed = TRUE
WHERE id = $1;