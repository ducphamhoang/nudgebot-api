package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"nudgebot-api/internal/config"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/nudge"

	"go.uber.org/zap"
)

// Scheduler defines the interface for the background reminder scheduler
type Scheduler interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
	GetMetrics() *SchedulerMetrics
}

// scheduler implements the Scheduler interface
type scheduler struct {
	config     config.SchedulerConfig
	repository nudge.NudgeRepository
	eventBus   events.EventBus
	logger     *zap.Logger
	metrics    *SchedulerMetrics

	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Goroutine management
	wg      sync.WaitGroup
	ticker  *time.Ticker
	running atomic.Bool
}

// NewScheduler creates a new scheduler instance
func NewScheduler(cfg config.SchedulerConfig, repository nudge.NudgeRepository, eventBus events.EventBus, logger *zap.Logger) (Scheduler, error) {
	// Validate configuration
	if cfg.PollInterval <= 0 {
		return nil, NewConfigurationError("poll_interval", cfg.PollInterval, "must be greater than 0")
	}
	if cfg.WorkerCount <= 0 {
		return nil, NewConfigurationError("worker_count", cfg.WorkerCount, "must be greater than 0")
	}
	if cfg.NudgeDelay < 60 {
		return nil, NewConfigurationError("nudge_delay", cfg.NudgeDelay, "must be at least 60 seconds")
	}
	if cfg.ShutdownTimeout <= 0 {
		return nil, NewConfigurationError("shutdown_timeout", cfg.ShutdownTimeout, "must be greater than 0")
	}

	return &scheduler{
		config:     cfg,
		repository: repository,
		eventBus:   eventBus,
		logger:     logger,
		metrics:    NewSchedulerMetrics(),
	}, nil
}

// Start begins the scheduler operation with worker goroutines
func (s *scheduler) Start(ctx context.Context) error {
	if s.running.Load() {
		return NewSchedulerError("scheduler_already_running", "scheduler is already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.ticker = time.NewTicker(time.Duration(s.config.PollInterval) * time.Second)
	s.running.Store(true)

	s.logger.Info("Starting reminder scheduler",
		zap.Int("poll_interval_seconds", s.config.PollInterval),
		zap.Int("nudge_delay_seconds", s.config.NudgeDelay),
		zap.Int("worker_count", s.config.WorkerCount))

	// Start worker goroutines
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	s.logger.Info("Reminder scheduler started successfully")
	return nil
}

// Stop gracefully shuts down the scheduler
func (s *scheduler) Stop() error {
	if !s.running.Load() {
		return NewSchedulerError("scheduler_not_running", "scheduler is not running")
	}

	s.logger.Info("Stopping reminder scheduler...")

	// Signal shutdown
	if s.cancel != nil {
		s.cancel()
	}

	// Stop ticker
	if s.ticker != nil {
		s.ticker.Stop()
	}

	// Wait for workers to complete with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("All scheduler workers stopped successfully")
	case <-time.After(time.Duration(s.config.ShutdownTimeout) * time.Second):
		s.logger.Warn("Scheduler shutdown timed out, some workers may still be running")
		return NewShutdownError("shutdown timeout exceeded", s.config.ShutdownTimeout)
	}

	s.running.Store(false)
	s.logger.Info("Reminder scheduler stopped successfully")
	return nil
}

// IsRunning returns true if the scheduler is currently running
func (s *scheduler) IsRunning() bool {
	return s.running.Load()
}

// GetMetrics returns the current scheduler metrics
func (s *scheduler) GetMetrics() *SchedulerMetrics {
	return s.metrics
}

// worker is the main worker goroutine that processes reminders
func (s *scheduler) worker(workerID int) {
	defer s.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Worker panic recovered, restarting worker",
				zap.Int("worker_id", workerID),
				zap.Any("panic", r))
			// Restart the worker
			s.wg.Add(1)
			go s.worker(workerID)
		}
	}()

	workerLogger := s.logger.With(zap.Int("worker_id", workerID))
	workerLogger.Info("Starting scheduler worker")

	worker := &reminderWorker{
		scheduler: s,
		workerID:  workerID,
		logger:    workerLogger,
	}

	for {
		select {
		case <-s.ctx.Done():
			workerLogger.Info("Worker stopping due to context cancellation")
			s.metrics.RecordWorkerActivity(workerID, false)
			return
		case <-s.ticker.C:
			s.metrics.RecordWorkerActivity(workerID, true)
			if err := worker.processReminders(); err != nil {
				workerLogger.Error("Failed to process reminders", zap.Error(err))
				s.metrics.RecordProcessingError(err)
			}
			s.metrics.RecordWorkerActivity(workerID, false)
		}
	}
}
