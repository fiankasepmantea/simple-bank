#!/bin/sh
set -e

# echo "Running DB migrations..."

# /app/migrate \
#   -path /app/migration \
#   -database "$DB_SOURCE" \
#   up

echo "TOKEN_SYMMETRIC_KEY length: ${#TOKEN_SYMMETRIC_KEY}"

echo "Starting API..."

exec /app/main
