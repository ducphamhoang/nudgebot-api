package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultConfig(t *testing.T) {
	// Save original environment
	originalConfigPath := os.Getenv("CONFIG_PATH")
	defer func() {
		if originalConfigPath != "" {
			os.Setenv("CONFIG_PATH", originalConfigPath)
		} else {
			os.Unsetenv("CONFIG_PATH")
		}
	}()

	// Clear CONFIG_PATH to test default
	os.Unsetenv("CONFIG_PATH")

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Test default values are set
	assert.NotZero(t, cfg.Server.Port)
	assert.NotEmpty(t, cfg.Server.Environment)
}

func TestLoad_CustomConfigPath(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  port: 9999
  environment: "test"
  
database:
  host: "test-db"
  port: 5433
  dbname: "test_nudgebot"
  user: "test_user"
  password: "test_pass"
  sslmode: "disable"
  
chatbot:
  token: "test-token"
  webhook_url: "/test-webhook"
  timeout: 45
  
llm:
  api_endpoint: "https://test-api.example.com"
  api_key: "test-key"
  model: "test-model"
  timeout: 60
  max_retries: 5
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to temp directory so config file is found
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Test custom values are loaded
	assert.Equal(t, 9999, cfg.Server.Port)
	assert.Equal(t, "test", cfg.Server.Environment)
	assert.Equal(t, "test-db", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "test_nudgebot", cfg.Database.DBName)
	assert.Equal(t, "test-token", cfg.Chatbot.Token)
	assert.Equal(t, "https://test-api.example.com", cfg.LLM.APIEndpoint)
	assert.Equal(t, 60, cfg.LLM.Timeout)
	assert.Equal(t, 5, cfg.LLM.MaxRetries)
}

func TestLoad_InvalidConfigPath(t *testing.T) {
	// Change to a directory without config file
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	cfg, err := Load()
	// Should still load with defaults when file doesn't exist
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should have default values
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "development", cfg.Server.Environment)
}

func TestLoad_MalformedYAML(t *testing.T) {
	// Create a temporary malformed config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	malformedContent := `
server:
  port: 8080
  environment: "test"
invalid_yaml: [
  - missing_closing_bracket
`

	err := os.WriteFile(configFile, []byte(malformedContent), 0644)
	require.NoError(t, err)

	// Change to temp directory so config file is found
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	cfg, err := Load()
	// Should return error for malformed YAML
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestConfig_ServerDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	// Test that server has sensible defaults
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "development", cfg.Server.Environment)
	assert.Equal(t, 30, cfg.Server.ReadTimeout)
	assert.Equal(t, 30, cfg.Server.WriteTimeout)
}

func TestConfig_DatabaseDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	// Test that database has required fields with defaults
	assert.NotNil(t, cfg.Database)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "postgres", cfg.Database.User)
	assert.Equal(t, "nudgebot", cfg.Database.DBName)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
}

func TestConfig_ChatbotDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	// Test that chatbot config exists with defaults
	assert.NotNil(t, cfg.Chatbot)
	assert.Equal(t, "/webhook", cfg.Chatbot.WebhookURL)
	assert.Equal(t, "", cfg.Chatbot.Token) // Empty by default
	assert.Equal(t, 30, cfg.Chatbot.Timeout)
}

func TestConfig_LLMDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	// Test that LLM config exists with defaults
	assert.NotNil(t, cfg.LLM)
	assert.Contains(t, cfg.LLM.APIEndpoint, "generativelanguage.googleapis.com")
	assert.Equal(t, "", cfg.LLM.APIKey) // Empty by default
	assert.Equal(t, 30, cfg.LLM.Timeout)
	assert.Equal(t, 3, cfg.LLM.MaxRetries)
	assert.Equal(t, "gemma-2-27b-it", cfg.LLM.Model)
}

func TestConfig_EventsDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	// Test that events config exists with defaults
	assert.NotNil(t, cfg.Events)
	assert.Equal(t, 1000, cfg.Events.BufferSize)
	assert.Equal(t, 4, cfg.Events.WorkerCount)
	assert.Equal(t, 30, cfg.Events.ShutdownTimeout)
}

func TestConfig_NudgeDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	// Test that nudge config exists with defaults
	assert.NotNil(t, cfg.Nudge)
	assert.Equal(t, 3600, cfg.Nudge.DefaultReminderInterval) // 1 hour
	assert.Equal(t, 3, cfg.Nudge.MaxNudges)
	assert.Equal(t, 86400, cfg.Nudge.CleanupInterval) // 24 hours
}

func TestConfig_SchedulerDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	// Test that scheduler config exists with defaults
	assert.NotNil(t, cfg.Scheduler)
	assert.Equal(t, 30, cfg.Scheduler.PollInterval)
	assert.Equal(t, 7200, cfg.Scheduler.NudgeDelay) // 2 hours
	assert.Equal(t, 2, cfg.Scheduler.WorkerCount)
	assert.Equal(t, 30, cfg.Scheduler.ShutdownTimeout)
	assert.True(t, cfg.Scheduler.Enabled)
}

func TestConfig_EnvironmentOverrides(t *testing.T) {
	// Save original environment
	originalVars := map[string]string{
		"SERVER_PORT":       os.Getenv("SERVER_PORT"),
		"DATABASE_HOST":     os.Getenv("DATABASE_HOST"),
		"CHATBOT_TOKEN":     os.Getenv("CHATBOT_TOKEN"),
		"LLM_API_KEY":       os.Getenv("LLM_API_KEY"),
		"SCHEDULER_ENABLED": os.Getenv("SCHEDULER_ENABLED"),
	}
	defer func() {
		for key, value := range originalVars {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Set environment variables
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DATABASE_HOST", "env-db-host")
	os.Setenv("CHATBOT_TOKEN", "env-token")
	os.Setenv("LLM_API_KEY", "env-api-key")
	os.Setenv("SCHEDULER_ENABLED", "false")

	cfg, err := Load()
	require.NoError(t, err)

	// Test that environment variables override config file values
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "env-db-host", cfg.Database.Host)
	assert.Equal(t, "env-token", cfg.Chatbot.Token)
	assert.Equal(t, "env-api-key", cfg.LLM.APIKey)
	assert.False(t, cfg.Scheduler.Enabled)
}

func TestConfig_RequiredFieldsValidation(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	// Test that critical configuration sections exist
	assert.NotNil(t, cfg.Server, "Server config should exist")
	assert.NotNil(t, cfg.Database, "Database config should exist")
	assert.NotNil(t, cfg.Chatbot, "Chatbot config should exist")
	assert.NotNil(t, cfg.LLM, "LLM config should exist")
	assert.NotNil(t, cfg.Events, "Events config should exist")
	assert.NotNil(t, cfg.Nudge, "Nudge config should exist")
	assert.NotNil(t, cfg.Scheduler, "Scheduler config should exist")
}

func TestConfig_TypeValidation(t *testing.T) {
	// Create a temporary config file with wrong types
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  port: "not_a_number"  # This should be an integer
  environment: "test"
  
database:
  host: "localhost"
  port: "not_a_number"  # This should be an integer
  dbname: "test_db"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to temp directory so config file is found
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	cfg, err := Load()
	// Viper should handle type conversion, but if it fails, we should get an error
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, cfg)
	} else {
		// If viper converts strings to 0 for invalid integers
		assert.NotNil(t, cfg)
		assert.Equal(t, 0, cfg.Server.Port) // Would be 0 for invalid conversion
	}
}

func TestConfig_PartialConfig(t *testing.T) {
	// Create a temporary config file with only some sections
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  port: 8080
  environment: "test"
# Missing other sections - should use defaults
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to temp directory so config file is found
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should load with provided values and defaults for missing sections
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "test", cfg.Server.Environment)

	// Missing sections should have defaults
	assert.Equal(t, "localhost", cfg.Database.Host) // Default value
	assert.Equal(t, 30, cfg.Chatbot.Timeout)        // Default value
	assert.Equal(t, 3, cfg.LLM.MaxRetries)          // Default value
	assert.True(t, cfg.Scheduler.Enabled)           // Default value
}

func TestConfig_EmptyConfig(t *testing.T) {
	// Create a temporary empty config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	// Change to temp directory so config file is found
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should load with all defaults
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "development", cfg.Server.Environment)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "/webhook", cfg.Chatbot.WebhookURL)
	assert.Contains(t, cfg.LLM.APIEndpoint, "generativelanguage.googleapis.com")
	assert.Equal(t, 1000, cfg.Events.BufferSize)
	assert.Equal(t, 3600, cfg.Nudge.DefaultReminderInterval)
	assert.True(t, cfg.Scheduler.Enabled)
}

func TestConfig_ComplexConfig(t *testing.T) {
	// Create a comprehensive config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  port: 9000
  environment: "production"
  read_timeout: 60
  write_timeout: 60

database:
  host: "prod-db.example.com"
  port: 5432
  user: "app_user"
  password: "secure_password"
  dbname: "nudgebot_prod"
  sslmode: "require"
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 600

chatbot:
  token: "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk"
  webhook_url: "/api/v1/telegram/webhook"
  timeout: 45

llm:
  api_endpoint: "https://custom-llm.example.com/v1/generate"
  api_key: "sk-1234567890abcdef"
  model: "custom-model-v2"
  timeout: 120
  max_retries: 5

events:
  buffer_size: 2000
  worker_count: 8
  shutdown_timeout: 60

nudge:
  default_reminder_interval: 7200
  max_nudges: 5
  cleanup_interval: 43200

scheduler:
  poll_interval: 15
  nudge_delay: 3600
  worker_count: 4
  shutdown_timeout: 45
  enabled: true
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to temp directory so config file is found
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Validate all sections are loaded correctly
	assert.Equal(t, 9000, cfg.Server.Port)
	assert.Equal(t, "production", cfg.Server.Environment)
	assert.Equal(t, 60, cfg.Server.ReadTimeout)

	assert.Equal(t, "prod-db.example.com", cfg.Database.Host)
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.Equal(t, 50, cfg.Database.MaxOpenConns)

	assert.Equal(t, "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk", cfg.Chatbot.Token)
	assert.Equal(t, "/api/v1/telegram/webhook", cfg.Chatbot.WebhookURL)

	assert.Equal(t, "https://custom-llm.example.com/v1/generate", cfg.LLM.APIEndpoint)
	assert.Equal(t, "custom-model-v2", cfg.LLM.Model)
	assert.Equal(t, 120, cfg.LLM.Timeout)

	assert.Equal(t, 2000, cfg.Events.BufferSize)
	assert.Equal(t, 8, cfg.Events.WorkerCount)

	assert.Equal(t, 7200, cfg.Nudge.DefaultReminderInterval)
	assert.Equal(t, 5, cfg.Nudge.MaxNudges)

	assert.Equal(t, 15, cfg.Scheduler.PollInterval)
	assert.Equal(t, 4, cfg.Scheduler.WorkerCount)
	assert.True(t, cfg.Scheduler.Enabled)
}
