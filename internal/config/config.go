package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Chatbot   ChatbotConfig   `mapstructure:"chatbot"`
	LLM       LLMConfig       `mapstructure:"llm"`
	Events    EventsConfig    `mapstructure:"events"`
	Nudge     NudgeConfig     `mapstructure:"nudge"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
}

type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Environment  string `mapstructure:"environment"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type ChatbotConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
	Token      string `mapstructure:"token"`
	Timeout    int    `mapstructure:"timeout"`
}

type LLMConfig struct {
	APIEndpoint string `mapstructure:"api_endpoint"`
	APIKey      string `mapstructure:"api_key"`
	Timeout     int    `mapstructure:"timeout"`
	MaxRetries  int    `mapstructure:"max_retries"`
	Model       string `mapstructure:"model"`
}

type EventsConfig struct {
	BufferSize      int `mapstructure:"buffer_size"`
	WorkerCount     int `mapstructure:"worker_count"`
	ShutdownTimeout int `mapstructure:"shutdown_timeout"`
}

type NudgeConfig struct {
	DefaultReminderInterval int `mapstructure:"default_reminder_interval"`
	MaxNudges               int `mapstructure:"max_nudges"`
	CleanupInterval         int `mapstructure:"cleanup_interval"`
}

type SchedulerConfig struct {
	PollInterval    int  `mapstructure:"poll_interval"`
	NudgeDelay      int  `mapstructure:"nudge_delay"`
	WorkerCount     int  `mapstructure:"worker_count"`
	ShutdownTimeout int  `mapstructure:"shutdown_timeout"`
	Enabled         bool `mapstructure:"enabled"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults()

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.environment", "development")
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "nudgebot")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 300)

	viper.SetDefault("chatbot.webhook_url", "/webhook")
	viper.SetDefault("chatbot.token", "")
	viper.SetDefault("chatbot.timeout", 30)

	viper.SetDefault("llm.api_endpoint", "https://generativelanguage.googleapis.com/v1beta/models/gemma-2-27b-it:generateContent")
	viper.SetDefault("llm.api_key", "")
	viper.SetDefault("llm.timeout", 30)
	viper.SetDefault("llm.max_retries", 3)
	viper.SetDefault("llm.model", "gemma-2-27b-it")

	viper.SetDefault("events.buffer_size", 1000)
	viper.SetDefault("events.worker_count", 4)
	viper.SetDefault("events.shutdown_timeout", 30)

	viper.SetDefault("nudge.default_reminder_interval", 3600) // 1 hour in seconds
	viper.SetDefault("nudge.max_nudges", 3)
	viper.SetDefault("nudge.cleanup_interval", 86400) // 24 hours in seconds

	viper.SetDefault("scheduler.poll_interval", 30) // 30 seconds
	viper.SetDefault("scheduler.nudge_delay", 7200) // 2 hours
	viper.SetDefault("scheduler.worker_count", 2)
	viper.SetDefault("scheduler.shutdown_timeout", 30)
	viper.SetDefault("scheduler.enabled", true)
}
