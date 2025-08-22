package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	PoolSize int    `mapstructure:"pool_size"`
}

type Settings struct {
	DB DBConfig `mapstructure:"database"`
}

func Load() (*Settings, error) {
	// Load settings from a configuration file or environment variables
	viper.SetConfigName("config_" + genEnv())
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var settings Settings
	if err := viper.Unmarshal(&settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &settings, nil
}

func genEnv() string {
	env := viper.GetString("ENV")
	if env == "" {
		return "dev"
	}
	return env
}
