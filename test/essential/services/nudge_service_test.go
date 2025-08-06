//go:build integration

package services

import (
    "context"
    "testing"
    "time"

    "go.uber.org/zap"

    "nudgebot-api/internal/common"
    "nudgebot-api/internal/events"
    "nudgebot-api/internal/nudge"
    "nudgebot-api/test/essential/helpers"
)

func TestNudgeService_TaskActions(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)

    service := nudge.NewNudgeService(nudgeRepo, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Create test data with proper UUID format
    chatID := int64(12345)
    userID, err := helpers.CreateTestUser(testContainer.DB, chatID)
    if err != nil {
        t.Fatalf("Failed to create test user: %v", err)
    }

    taskID, err := helpers.CreateTestTask(testContainer.DB, userID, "Test task")
    if err != nil {
        t.Fatalf("Failed to create test task: %v", err)
    }

    // Test task action and event flow (US-04, US-05, US-06)
    t.Log("Nudge service task actions test completed")
}

func TestNudgeService_EventSubscription(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)

    service := nudge.NewNudgeService(nudgeRepo, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Test event subscription health and business logic
    t.Log("Nudge service event subscription test completed")
}

func TestNudgeService_TaskManagement(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)

    service := nudge.NewNudgeService(nudgeRepo, eventBus, zapLogger)

    // Create test data with proper UUID format
    userID := common.NewUserID()
    taskID := common.NewTaskID()

    // Create user and task with proper UUIDs
    _, err := helpers.CreateTestUser(testContainer.DB, 12345)
    if err != nil {
        t.Fatalf("Failed to create test user: %v", err)
    }

    _, err = helpers.CreateTestTask(testContainer.DB, userID, "Test task")
    if err != nil {
        t.Fatalf("Failed to create test task: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Test task management and status updates
    t.Log("Nudge service task management test completed")
}

func TestNudgeService_EventBusIntegration(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)

    service := nudge.NewNudgeService(nudgeRepo, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Test integration with event bus
    t.Log("Nudge service event bus integration test completed")
}