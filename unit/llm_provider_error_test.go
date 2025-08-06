package unit

import (
"testing"

"github.com/stretchr/testify/assert"
"go.uber.org/zap/zaptest"

"nudgebot-api/internal/common"
"nudgebot-api/internal/config"
"nudgebot-api/internal/events"
"nudgebot-api/internal/llm"
)

func TestLLMProvider_NetworkConnectionFailure(t *testing.T) {
logger := zaptest.NewLogger(t)
eventBus := events.NewEventBus(logger)

llmConfig := config.LLMConfig{
APIEndpoint: "http://non-existent-host.invalid",
APIKey:      "test-api-key",
Timeout:     1000,
MaxRetries:  0,
}

service := llm.NewLLMService(eventBus, logger, llmConfig)
result, err := service.ParseTask("test message", common.UserID("user123"))

assert.Error(t, err)
assert.Nil(t, result)
}
