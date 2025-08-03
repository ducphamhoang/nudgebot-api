package main

import (
	"fmt"
	"strings"

	"nudgebot-api/internal/events"
	"nudgebot-api/internal/nudge"

	"go.uber.org/zap"
)

func main() {
	fmt.Println("🧪 Demonstrating Enhanced Event Subscription Error Handling")
	fmt.Println(strings.Repeat("=", 60))

	logger := zap.NewNop()

	fmt.Println("\n1. Testing with working event bus:")
	workingEventBus := events.NewEventBus(logger)
	service, err := nudge.NewNudgeService(workingEventBus, logger, nil)
	if err != nil {
		fmt.Printf("❌ Service creation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Service created successfully\n")

		// Test health check
		if err := service.CheckSubscriptionHealth(); err != nil {
			fmt.Printf("❌ Health check failed: %v\n", err)
		} else {
			fmt.Printf("✅ Health check passed - all subscriptions active\n")
		}
	}
	workingEventBus.Close()

	fmt.Println("\n2. Testing with failing event bus:")
	failingEventBus := &FailingEventBus{}
	service, err = nudge.NewNudgeService(failingEventBus, logger, nil)
	if err != nil {
		fmt.Printf("✅ Service creation correctly failed: %v\n", err)
		fmt.Printf("   Error handling working properly!\n")
	} else {
		fmt.Printf("❌ Service creation should have failed\n")
	}

	fmt.Println("\n📋 Summary of Improvements:")
	fmt.Println("   • Added proper error return from setupEventSubscriptions()")
	fmt.Println("   • Implemented retry logic with exponential backoff")
	fmt.Println("   • Added subscription health check method")
	fmt.Println("   • Created custom error types for better error handling")
	fmt.Println("   • Service creation now fails fast if critical subscriptions fail")
}

// FailingEventBus for demonstration
type FailingEventBus struct{}

func (f *FailingEventBus) Subscribe(topic string, handler interface{}) error {
	return fmt.Errorf("simulated subscription failure for topic: %s", topic)
}

func (f *FailingEventBus) Unsubscribe(topic string, handler interface{}) error {
	return nil
}

func (f *FailingEventBus) Publish(topic string, event interface{}) error {
	return nil
}

func (f *FailingEventBus) Close() error {
	return nil
}
