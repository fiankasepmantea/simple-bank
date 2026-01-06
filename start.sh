#!/bin/sh
set -e

echo "Running DB migrations..."

/app/migrate \
  -path /app/migration \
  -database "$DB_SOURCE" \
  up

echo "Starting API..."
exec /app/main
