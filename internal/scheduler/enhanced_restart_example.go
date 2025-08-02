package scheduler

// This file contains an example of how to implement proper worker restart
// mechanism if automatic restart is desired in the future.
//
// The current implementation (in scheduler.go) removes automatic restart
// to prevent WaitGroup corruption and maintain clean shutdown semantics.
//
// If you want to add automatic restart capability, consider these approaches:

/*
// Enhanced scheduler with restart capability
type enhancedScheduler struct {
	*scheduler
	workerChannels map[int]chan struct{} // For controlled worker shutdown
	restartLimit   int                   // Maximum restart attempts per worker
	restartCounts  map[int]int          // Track restart attempts per worker
}

// Enhanced worker with proper restart mechanism
func (s *enhancedScheduler) workerWithRestart(workerID int) {
	defer s.wg.Done()

	restartCount := 0
	maxRestarts := 3 // configurable limit

	for {
		// Check if we've exceeded restart limit
		if restartCount >= maxRestarts {
			s.logger.Error("Worker exceeded restart limit, permanently stopping",
				zap.Int("worker_id", workerID),
				zap.Int("restart_count", restartCount))
			return
		}

		// Run worker with panic recovery
		workerCompleted := s.runWorkerSafely(workerID, restartCount)

		if workerCompleted {
			// Normal shutdown
			return
		}

		// Worker panicked, increment restart count
		restartCount++
		s.logger.Warn("Restarting worker after panic",
			zap.Int("worker_id", workerID),
			zap.Int("restart_count", restartCount))

		// Brief delay before restart to prevent rapid panic loops
		select {
		case <-time.After(time.Second * time.Duration(restartCount)):
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *enhancedScheduler) runWorkerSafely(workerID, restartCount int) (completed bool) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Worker panic recovered",
				zap.Int("worker_id", workerID),
				zap.Int("restart_count", restartCount),
				zap.Any("panic", r))
			panicErr := NewWorkerError(workerID, "panic_recovery", fmt.Errorf("worker panic: %v", r))
			s.metrics.RecordProcessingError(panicErr)
			completed = false // Signal that restart is needed
		}
	}()

	workerLogger := s.logger.With(
		zap.Int("worker_id", workerID),
		zap.Int("restart_count", restartCount))

	if restartCount > 0 {
		workerLogger.Info("Worker restarted after panic")
	} else {
		workerLogger.Info("Starting scheduler worker")
	}

	worker := &reminderWorker{
		scheduler: s.scheduler,
		workerID:  workerID,
		logger:    workerLogger,
	}

	for {
		select {
		case <-s.ctx.Done():
			workerLogger.Info("Worker stopping due to context cancellation")
			s.metrics.RecordWorkerActivity(workerID, false)
			return true // Normal completion
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

// Alternative approach: Worker manager with supervision
type workerManager struct {
	scheduler     *scheduler
	workers       map[int]*workerState
	restartPolicy RestartPolicy
}

type workerState struct {
	id            int
	restartCount  int
	lastRestart   time.Time
	active        bool
}

type RestartPolicy struct {
	MaxRestarts     int
	RestartDelay    time.Duration
	BackoffMultiplier float64
	MaxDelay        time.Duration
}

func (wm *workerManager) superviseWorker(workerID int) {
	state := &workerState{
		id:     workerID,
		active: true,
	}
	wm.workers[workerID] = state

	for state.active && state.restartCount < wm.restartPolicy.MaxRestarts {
		// Calculate restart delay with exponential backoff
		delay := wm.calculateRestartDelay(state.restartCount)

		if state.restartCount > 0 {
			select {
			case <-time.After(delay):
			case <-wm.scheduler.ctx.Done():
				return
			}
		}

		// Start worker and wait for completion or panic
		workerDone := make(chan bool, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					workerDone <- false // Panic occurred
				} else {
					workerDone <- true // Normal completion
				}
			}()

			wm.runWorker(workerID)
		}()

		select {
		case normal := <-workerDone:
			if normal {
				return // Normal shutdown
			}
			// Panic occurred, increment restart count
			state.restartCount++
			state.lastRestart = time.Now()
		case <-wm.scheduler.ctx.Done():
			return
		}
	}
}
*/
