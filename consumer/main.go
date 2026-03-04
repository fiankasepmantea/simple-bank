package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load config from environment
	cfg := ConsumerConfig{
		// ✅ FIX: Parse comma-separated brokers string into []string
		Brokers:          parseBrokers(getEnv("KAFKA_BROKERS", "simple-bank-kafka:9092")),
		Topic:            getEnv("KAFKA_TOPIC", "transaction-events"),
		GroupID:          getEnv("KAFKA_GROUP_ID", "transaction-consumer-group"),
		DatabaseDSN:      getEnv("DATABASE_DSN", ""),
		EmailEnabled:     getEnvBool("EMAIL_ENABLED", false),
		AnalyticsEnabled: getEnvBool("ANALYTICS_ENABLED", false),
		PushEnabled:      getEnvBool("PUSH_ENABLED", false),
	}

	slog.Info("Starting consumer service", "config", cfg)

	// Create consumer with dependencies
	consumer, err := NewConsumer(cfg)
	if err != nil {
		slog.Error("Failed to create consumer", "error", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// Setup context with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		slog.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	// Start consuming
	if err := consumer.Start(ctx); err != nil && err != context.Canceled {
		slog.Error("Consumer stopped with error", "error", err)
		os.Exit(1)
	}

	slog.Info("Consumer shutdown complete")
}

// ✅ HELPER: Parse comma-separated brokers string into []string
func parseBrokers(brokersStr string) []string {
	if brokersStr == "" {
		return []string{"simple-bank-kafka:9092"}
	}
	// Split by comma and trim spaces: "host1:9092,host2:9092" -> ["host1:9092", "host2:9092"]
	parts := strings.Split(brokersStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}