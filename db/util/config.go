package util

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DBSource            string        `mapstructure:"DB_SOURCE"`
	ServerAddress       string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey   string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	// Try to read config file, but don't fail if it's missing
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found — that's OK, we'll use environment variables
			fmt.Println("⚠️  Config file 'app.env' not found. Using environment variables only.")
		} else {
			return config, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Always unmarshal, even if config file was missing
	err = viper.Unmarshal(&config)
	return config, err
}