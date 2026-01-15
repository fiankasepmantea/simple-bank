#!/bin/sh
set -e

echo "⏳ Waiting for Postgres..."
/app/wait-for.sh postgres:5432 -- echo "✅ Postgres is up"

# Run main binary (migration + API)
echo "🌟 Starting API server..."
/app/main
