package main

import (
	"fmt"
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/scheduler"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()

	// Test 1: Invalid PollInterval (0)
	cfg1 := config.SchedulerConfig{
		PollInterval:    0,
		WorkerCount:     1,
		NudgeDelay:      60,
		ShutdownTimeout: 10,
	}

	_, err1 := scheduler.NewScheduler(cfg1, nil, nil, logger)
	if err1 != nil {
		fmt.Printf("✅ PollInterval validation: %v\n", err1)
	}

	// Test 2: Invalid WorkerCount (0)
	cfg2 := config.SchedulerConfig{
		PollInterval:    30,
		WorkerCount:     0,
		NudgeDelay:      60,
		ShutdownTimeout: 10,
	}

	_, err2 := scheduler.NewScheduler(cfg2, nil, nil, logger)
	if err2 != nil {
		fmt.Printf("✅ WorkerCount validation: %v\n", err2)
	}

	// Test 3: Invalid NudgeDelay (< 60)
	cfg3 := config.SchedulerConfig{
		PollInterval:    30,
		WorkerCount:     1,
		NudgeDelay:      30,
		ShutdownTimeout: 10,
	}

	_, err3 := scheduler.NewScheduler(cfg3, nil, nil, logger)
	if err3 != nil {
		fmt.Printf("✅ NudgeDelay validation: %v\n", err3)
	}

	// Test 4: Invalid ShutdownTimeout (0)
	cfg4 := config.SchedulerConfig{
		PollInterval:    30,
		WorkerCount:     1,
		NudgeDelay:      60,
		ShutdownTimeout: 0,
	}

	_, err4 := scheduler.NewScheduler(cfg4, nil, nil, logger)
	if err4 != nil {
		fmt.Printf("✅ ShutdownTimeout validation: %v\n", err4)
	}

	// Test 5: Valid configuration
	cfg5 := config.SchedulerConfig{
		PollInterval:    30,
		WorkerCount:     2,
		NudgeDelay:      60,
		ShutdownTimeout: 10,
	}

	_, err5 := scheduler.NewScheduler(cfg5, nil, nil, logger)
	if err5 == nil {
		fmt.Printf("✅ Valid configuration: No error (expected)\n")
	}
}
