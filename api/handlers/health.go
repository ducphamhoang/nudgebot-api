package handlers

import (
    "net/http"

    "nudgebot-api/internal/database"
    "nudgebot-api/pkg/logger"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

type HealthHandler struct {
    db     *gorm.DB
    logger *logger.Logger
}

func NewHealthHandler(db *gorm.DB, logger *logger.Logger) *HealthHandler {
    return &HealthHandler{
        db:     db,
        logger: logger,
    }
}

func (h *HealthHandler) Check(c *gin.Context) {
    status := "ok"
    statusCode := http.StatusOK

    // Check database connection
    if err := database.HealthCheck(h.db); err != nil {
        h.logger.Error("Database health check failed", "error", err)
        status = "error"
        statusCode = http.StatusServiceUnavailable
    }

    response := gin.H{
        "status":    status,
        "timestamp": gin.H{},
        "service":   "nudgebot-api",
    }

    c.JSON(statusCode, response)
}