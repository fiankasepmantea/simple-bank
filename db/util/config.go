package util

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application
// The values are read by viper from a config file or environment variable
type Config struct {
	DBSource            string        `mapstructure:"DB_SOURCE"`
	ServerAddress       string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey   string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
}

// LoadConfig reads configuration from file or environment variables
// func LoadConfig(path string) (config Config, err error) {
// 	viper.AddConfigPath(path)
// 	viper.SetConfigName("app")
// 	viper.SetConfigType("env")

// 	viper.AutomaticEnv()

// 	err = viper.ReadInConfig()
// 	if err != nil {
// 		return
// 	}

// 	err = viper.Unmarshal(&config)
// 	return
// }

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	// Try to read config file, but don't fail if it's missing
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, proceed with env vars only
		} else {
			return config, fmt.Errorf("error reading config: %w", err)
		}
	}

	err = viper.Unmarshal(&config)
	return
}
