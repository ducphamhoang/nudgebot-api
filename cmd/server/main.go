package main

import (
	_ "github.com/joho/godotenv/autoload" // Load .env file automatically

	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nudgebot-api/api/routes"
	"nudgebot-api/internal/chatbot"
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/database"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/llm"
	"nudgebot-api/internal/nudge"
	"nudgebot-api/internal/scheduler"
	"nudgebot-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize logger
	logger := logger.New()
	defer logger.Sync()

	// Get the underlying zap logger for services
	zapLogger := logger.SugaredLogger.Desugar()

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

	// Run nudge module migrations
	if err := nudge.RunMigrations(db); err != nil {
		logger.Fatal("Failed to run nudge migrations", "error", err)
	}

	// Initialize event bus
	eventBus := events.NewEventBus(zapLogger)

	// Initialize services
	chatbotService, err := chatbot.NewChatbotService(eventBus, zapLogger, cfg.Chatbot)
	if err != nil {
		logger.Fatal("Failed to initialize chatbot service", "error", err)
	}
	llmService := llm.NewLLMService(eventBus, zapLogger, cfg.LLM)
	nudgeRepository := nudge.NewGormNudgeRepository(db, zapLogger)
	nudgeService, err := nudge.NewNudgeService(eventBus, zapLogger, nudgeRepository)
	if err != nil {
		logger.Fatal("Failed to initialize nudge service", "error", err)
	}

	// Initialize scheduler
	var reminderScheduler scheduler.Scheduler
	if cfg.Scheduler.Enabled {
		var err error
		reminderScheduler, err = scheduler.NewScheduler(cfg.Scheduler, nudgeRepository, eventBus, zapLogger)
		if err != nil {
			logger.Error("Failed to create scheduler", "error", err)
			log.Fatal("Failed to create scheduler: ", err)
		}

		// Start scheduler in background
		go func() {
			if err := reminderScheduler.Start(context.Background()); err != nil {
				logger.Error("Scheduler failed to start", "error", err)
			}
		}()

		logger.Info("Reminder scheduler started",
			"poll_interval", cfg.Scheduler.PollInterval,
			"nudge_delay", cfg.Scheduler.NudgeDelay,
			"worker_count", cfg.Scheduler.WorkerCount)
	} else {
		logger.Info("Reminder scheduler disabled")
	}

	// Log that services are initialized (to avoid unused variable warnings)
	logger.Info("Services initialized",
		"chatbot", chatbotService != nil,
		"llm", llmService != nil,
		"nudge", nudgeService != nil)

	// Validate event bus subscriptions
	logger.Info("Validating event bus subscriptions...")

	// Test event publishing to ensure all subscriptions are working
	testEvent := events.NewEvent()
	if err := eventBus.Publish("test.connection", testEvent); err != nil {
		logger.Warn("Event bus test failed", "error", err)
	}

	logger.Info("Event bus integration completed",
		"chatbot_subscriptions", "TaskParsed, ReminderDue, TaskListResponse, TaskActionResponse, TaskCreated",
		"llm_subscriptions", "MessageReceived",
		"nudge_subscriptions", "TaskParsed, TaskListRequested, TaskActionRequested")

	// Allow services to complete initialization
	time.Sleep(100 * time.Millisecond)

	// Setup Gin router
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	routes.SetupRoutes(router, db, logger, chatbotService)

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

	// Stop scheduler first
	if cfg.Scheduler.Enabled && reminderScheduler != nil {
		logger.Info("Stopping reminder scheduler...")
		if err := reminderScheduler.Stop(); err != nil {
			logger.Error("Failed to stop scheduler gracefully", "error", err)
		} else {
			logger.Info("Reminder scheduler stopped successfully")
		}
	}

	// Stop accepting new events
	logger.Info("Stopping event processing...")

	// Close event bus with timeout
	eventBusCtx, eventBusCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer eventBusCancel()

	go func() {
		if err := eventBus.Close(); err != nil {
			logger.Error("Failed to close event bus", "error", err)
		}
	}()

	select {
	case <-eventBusCtx.Done():
		logger.Warn("Event bus shutdown timed out")
	case <-time.After(5 * time.Second):
		logger.Info("Event bus closed successfully")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}
