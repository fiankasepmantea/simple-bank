-- name: CreateUser :one
INSERT INTO users (
  username, hashed_password, full_name, email
) VALUES (
  $1, $2, $3, $4
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET
    hashed_password = COALESCE(sqlc.narg(hashed_password), hashed_password),
    password_changed_at = COALESCE(sqlc.narg(password_changed_at), password_changed_at),
    full_name       = COALESCE(sqlc.narg(full_name), full_name),
    email           = COALESCE(sqlc.narg(email), email)
WHERE
    username = sqlc.arg(username)
RETURNING *;

-- name: UpdateUserID :one
UPDATE users
SET id = $2
WHERE username = $1
  AND id IS NULL
RETURNING username, hashed_password, full_name, email, password_changed_at, created_at, id;

