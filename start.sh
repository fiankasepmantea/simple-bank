#!/bin/sh
set -e

echo "⏳ Waiting for Postgres..."
/app/wait-for.sh postgres:5432 -- echo "✅ Postgres is up"

echo "⏳ Waiting for RabbitMQ..."
/app/wait-for.sh rabbitmq:5672

# Run main binary (migration + API)
echo "🌟 Starting API server..."
/app/main
