package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "nudgebot-api/api/routes"
    "nudgebot-api/internal/config"
    "nudgebot-api/internal/database"
    "nudgebot-api/pkg/logger"

    "github.com/gin-gonic/gin"
)

func main() {
    // Initialize logger
    logger := logger.New()
    defer logger.Sync()

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Initialize database
    db, err := database.NewPostgresConnection(cfg.Database)
    if err != nil {
        logger.Fatal("Failed to connect to database", "error", err)
    }

    // Setup Gin router
    if cfg.Server.Environment == "production" {
        gin.SetMode(gin.ReleaseMode)
    }

    router := gin.New()
    routes.SetupRoutes(router, db, logger)

    // Create HTTP server
    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
        Handler:      router,
        ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
        WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
    }

    // Start server in goroutine
    go func() {
        logger.Info("Starting server", "port", cfg.Server.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Failed to start server", "error", err)
        }
    }()

    // Wait for interrupt signal for graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        logger.Fatal("Server forced to shutdown", "error", err)
    }

    logger.Info("Server exited")
}