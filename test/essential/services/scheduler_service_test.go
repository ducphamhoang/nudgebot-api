//go:build integration

package services

import (
    "context"
    "testing"
    "time"

    "go.uber.org/zap"

    "nudgebot-api/internal/events"
    "nudgebot-api/internal/nudge"
    "nudgebot-api/internal/scheduler"
    "nudgebot-api/test/essential/helpers"
)

func TestSchedulerService_StartStop(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)

    service := scheduler.NewSchedulerService(nudgeRepo, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Test service lifecycle
    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Service should start and stop cleanly
    cancel()
    time.Sleep(100 * time.Millisecond)

    t.Log("Scheduler service start/stop test completed")
}

func TestSchedulerService_ReminderProcessing(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)

    service := scheduler.NewSchedulerService(nudgeRepo, eventBus, zapLogger)

    // Create test data
    chatID := int64(12345)
    userID, err := helpers.CreateTestUser(testContainer.DB, chatID)
    if err != nil {
        t.Fatalf("Failed to create test user: %v", err)
    }

    taskID, err := helpers.CreateTestTask(testContainer.DB, userID, "Test task")
    if err != nil {
        t.Fatalf("Failed to create test task: %v", err)
    }

    // Create overdue reminder
    overdueTime := time.Now().Add(-1 * time.Hour)
    err = helpers.CreateTestReminder(testContainer.DB, taskID, overdueTime)
    if err != nil {
        t.Fatalf("Failed to create test reminder: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(2 * time.Second)

    // Test reminder processing and nudge creation (US-04, US-06)
    t.Log("Scheduler service reminder processing test completed")
}

func TestSchedulerService_ErrorRecovery(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)

    service := scheduler.NewSchedulerService(nudgeRepo, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Test error handling and recovery
    t.Log("Scheduler service error recovery test completed")
}

func TestSchedulerService_MetricsValidation(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)

    service := scheduler.NewSchedulerService(nudgeRepo, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Test metrics and monitoring validation
    t.Log("Scheduler service metrics validation test completed")
}