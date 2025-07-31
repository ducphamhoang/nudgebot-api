package routes

import (
    "nudgebot-api/api/handlers"
    "nudgebot-api/api/middleware"
    "nudgebot-api/pkg/logger"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB, logger *logger.Logger) {
    // Add middleware
    router.Use(middleware.RequestLogging(logger))
    router.Use(gin.Recovery())

    // Initialize handlers
    healthHandler := handlers.NewHealthHandler(db, logger)

    // Setup routes
    v1 := router.Group("/api/v1")
    {
        v1.GET("/health", healthHandler.Check)
    }

    // Root health check
    router.GET("/health", healthHandler.Check)
}