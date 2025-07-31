package middleware

import (
    "time"

    "nudgebot-api/pkg/logger"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func RequestLogging(logger *logger.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Generate request ID
        requestID := uuid.New().String()
        c.Set("request_id", requestID)

        // Create logger with request ID
        reqLogger := logger.WithRequestID(requestID)
        c.Set("logger", reqLogger)

        start := time.Now()
        path := c.Request.URL.Path
        method := c.Request.Method

        reqLogger.Info("Request started",
            "method", method,
            "path", path,
            "client_ip", c.ClientIP(),
        )

        c.Next()

        duration := time.Since(start)
        statusCode := c.Writer.Status()

        reqLogger.Info("Request completed",
            "method", method,
            "path", path,
            "status_code", statusCode,
            "duration_ms", duration.Milliseconds(),
        )
    }
}