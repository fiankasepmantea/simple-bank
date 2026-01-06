#!/bin/sh
set -e
echo "Running DB migrations..."
source /app/app.env
/app/migrate \
  -path /app/migration \
  -database "$DB_SOURCE" \
  up
echo "Starting API..."
exec /app/main
