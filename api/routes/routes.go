package routes

import (
	"nudgebot-api/api/handlers"
	"nudgebot-api/api/middleware"
	"nudgebot-api/internal/chatbot"
	"nudgebot-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB, logger *logger.Logger, chatbotService chatbot.ChatbotService) {
	// Add middleware
	router.Use(middleware.RequestLogging(logger))
	router.Use(gin.Recovery())

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db, logger)
	webhookHandler := handlers.NewWebhookHandler(chatbotService, logger)

	// Setup routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.Check)

		// Telegram webhook endpoints
		v1.POST("/telegram/webhook", webhookHandler.HandleTelegramWebhook)
		v1.POST("/telegram/setup-webhook", webhookHandler.SetupWebhook)
		v1.GET("/telegram/webhook-info", webhookHandler.GetWebhookInfo)
	}

	// Root health check
	router.GET("/health", healthHandler.Check)
}
