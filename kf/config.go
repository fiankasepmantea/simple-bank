package kf

import "github.com/spf13/viper"

type Config struct {
	Brokers []string `mapstructure:"KAFKA_BROKERS"`
	Topic   string   `mapstructure:"KAFKA_TOPIC"`
}

// DefaultConfig returns default Kafka configuration
func LoadConfig() Config {
    viper.AutomaticEnv()
    viper.BindEnv("KAFKA_BROKERS")
    viper.BindEnv("KAFKA_TOPIC")
    
    brokers := viper.GetStringSlice("KAFKA_BROKERS")
    if len(brokers) == 0 {
        brokers = []string{"localhost:9092"} // fallback
    }
    
    topic := viper.GetString("KAFKA_TOPIC")
    if topic == "" {
        topic = "transaction-events"
    }
    
    return Config{
        Brokers: brokers,
        Topic:   topic,
    }
}