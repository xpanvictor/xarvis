package config

import (
    "fmt"
    "os"

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
    // MySQL/TiDB DSN
    // username:password@tcp(host:port)/dbname?params
    if d.Password == "" {
        return fmt.Sprintf("%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
            d.Username, d.Host, d.Port, d.Name)
    }
    return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        d.Username, d.Password, d.Host, d.Port, d.Name)
}

type AssistantKeysObj struct {
    OpenAiApiKey string `mapstructure:"open_ai_api_key"`
}

type BrainConfig struct {
	MaxToolCallLimit int `mapstructure:"max_tool_call_limit"`
}

type Settings struct {
	DB            DBConfig         `mapstructure:"database"`
	AssistantKeys AssistantKeysObj `mapstructure:"assistantKeys"`
	Env           string           `mapstructure:"env"`
	Debug         bool             `mapstructure:"debug" default:"false"`
	BrainConfig   BrainConfig
}

func Load() (*Settings, error) {
    // Prefer explicit config file via env var
    if cfgPath := os.Getenv("XARVIS_CONFIG"); cfgPath != "" {
        viper.SetConfigFile(cfgPath)
    } else {
        // Load settings from conventional locations: current dir, ./config, /etc/xarvis
        viper.SetConfigName("config_" + genEnv())
        viper.SetConfigType("yaml")
        viper.AddConfigPath(".")
        viper.AddConfigPath("./config")
        viper.AddConfigPath("/etc/xarvis")
    }

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
