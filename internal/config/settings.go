package config

import (
	"fmt"
	"net/url"
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
	TLS      bool   `mapstructure:"tls"`
}

func (d DBConfig) DSN() string {
	// MySQL/TiDB DSN
	// username:password@tcp(host:port)/dbname?params
	base := "charset=utf8mb4&parseTime=True&loc=Local"
	if d.TLS {
		base += "&tls=true"
	}
	if d.Password == "" {
		return fmt.Sprintf("%s@tcp(%s:%d)/%s?%s",
			d.Username, d.Host, d.Port, d.Name, base)
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		d.Username, d.Password, d.Host, d.Port, d.Name, base)
}

type LLMModelConfig struct {
	Name     string
	Url      url.URL
	Password string
	// others
}

type OllamaConfig struct {
	LLamaModels []LLMModelConfig
}

type GeminiConfig struct {
	APIKey string `mapstructure:"gemini_api_key"`
}

type AssistantKeysObj struct {
	OpenAiApiKey      string `mapstructure:"open_ai_api_key"`
	OllamaCredentials OllamaConfig
	Gemini            GeminiConfig `mapstructure:"gemini"`
}

type BrainConfig struct {
	MaxToolCallLimit int   `mapstructure:"max_tool_call_limit"`
	MsgTTLMins       int64 `mapstructure:"msg_ttl_mins"`
}

type SysModelsConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

type VoiceConfig struct {
	STTURL string `mapstructure:"stt_url"`
	TTSURL string `mapstructure:"tts_url"`
}

type AuthConfig struct {
	JWTSecret     string `mapstructure:"jwt_secret"`
	TokenTTLHours int    `mapstructure:"token_ttl_hours"`
}

type Settings struct {
	DB            DBConfig         `mapstructure:"database"`
	AssistantKeys AssistantKeysObj `mapstructure:"assistantKeys"`
	Env           string           `mapstructure:"env"`
	Debug         bool             `mapstructure:"debug" default:"false"`
	BrainConfig   BrainConfig
	SysModels     SysModelsConfig `mapstructure:"sys_models"`
	Voice         VoiceConfig     `mapstructure:"voice"`
	Auth          AuthConfig      `mapstructure:"auth"`
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
