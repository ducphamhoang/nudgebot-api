package handlers

import (
	"fmt"
	"io"
	"net/http"

	"nudgebot-api/internal/chatbot"
	"nudgebot-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// WebhookHandler handles Telegram webhook requests
type WebhookHandler struct {
	chatbotService chatbot.ChatbotService
	logger         *logger.Logger
}

// NewWebhookHandler creates a new WebhookHandler instance
func NewWebhookHandler(chatbotService chatbot.ChatbotService, logger *logger.Logger) *WebhookHandler {
	return &WebhookHandler{
		chatbotService: chatbotService,
		logger:         logger,
	}
}

// HandleTelegramWebhook processes incoming Telegram webhook updates
func (h *WebhookHandler) HandleTelegramWebhook(c *gin.Context) {
	// Generate correlation ID for tracking
	correlationID := fmt.Sprintf("webhook_%s_%d", c.ClientIP(), c.Request.ContentLength)

	h.logger.Info("Received Telegram webhook",
		"correlation_id", correlationID,
		"content_length", c.Request.ContentLength,
		"content_type", c.GetHeader("Content-Type"),
		"user_agent", c.GetHeader("User-Agent"))

	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body",
			"correlation_id", correlationID,
			"error", err)
		// Always return 200 as per Telegram webhook requirements
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	// Validate body size
	if len(body) == 0 {
		h.logger.Warn("Received empty webhook body",
			"correlation_id", correlationID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	// Optional: Validate Content-Type
	contentType := c.GetHeader("Content-Type")
	if contentType != "application/json" {
		h.logger.Warn("Unexpected content type",
			"correlation_id", correlationID,
			"content_type", contentType)
	}

	// Process the webhook through the chatbot service
	err = h.chatbotService.HandleWebhook(body)
	if err != nil {
		h.logger.Error("Failed to process webhook",
			"correlation_id", correlationID,
			"error", err,
			"body_size", len(body))
		// Still return 200 to prevent Telegram from retrying
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	h.logger.Debug("Webhook processed successfully",
		"correlation_id", correlationID,
		"body_size", len(body))

	// Always return 200 OK as required by Telegram
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SetupWebhook configures the webhook URL with Telegram (for development)
func (h *WebhookHandler) SetupWebhook(c *gin.Context) {
	var request struct {
		WebhookURL string `json:"webhook_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Error("Invalid webhook setup request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Setting up webhook", "webhook_url", request.WebhookURL)

	// Note: This would require extending the chatbot service interface
	// to expose webhook setup functionality. For now, return a placeholder response.

	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"message":     "Webhook setup requested",
		"webhook_url": request.WebhookURL,
	})
}

// GetWebhookInfo returns information about the current webhook (for debugging)
func (h *WebhookHandler) GetWebhookInfo(c *gin.Context) {
	h.logger.Info("Webhook info requested")

	// This would require extending the chatbot service to get webhook info
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"ok":        true,
		"message":   "Webhook info endpoint",
		"timestamp": gin.H{},
	})
}
