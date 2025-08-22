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

func (d DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		d.Host, d.Port, d.Username, d.Password, d.Name)
}

type AssistantKeysObj struct {
	OpenAiApiKey string `mapstructure:"open_ai_api_key`
}

type Settings struct {
	DB            DBConfig         `mapstructure:"database"`
	AssistantKeys AssistantKeysObj `mapstructure:"assistantKeys"`
	Env           string           `mapstructure:"env"`
	Debug         bool             `mapstructure:"debug" default:"false"`
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
