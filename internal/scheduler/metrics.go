package scheduler

import (
	"sync"
	"time"
)

// SchedulerMetrics tracks performance and health metrics for the scheduler
type SchedulerMetrics struct {
	mu                    sync.RWMutex
	RemindersProcessed    int64
	NudgesCreated         int64
	ProcessingErrors      int64
	AverageProcessingTime time.Duration
	LastProcessingTime    time.Time
	WorkerUtilization     map[int]float64
	totalProcessingTime   time.Duration
	processingCycles      int64
}

// HealthStatus represents the health status of the scheduler
type HealthStatus struct {
	IsHealthy             bool      `json:"is_healthy"`
	LastProcessingTime    time.Time `json:"last_processing_time"`
	ProcessingErrors      int64     `json:"processing_errors"`
	AverageProcessingTime string    `json:"average_processing_time"`
	ErrorRate             float64   `json:"error_rate"`
}

// MetricsSummary provides a summary of scheduler metrics
type MetricsSummary struct {
	RemindersProcessed    int64           `json:"reminders_processed"`
	NudgesCreated         int64           `json:"nudges_created"`
	ProcessingErrors      int64           `json:"processing_errors"`
	AverageProcessingTime string          `json:"average_processing_time"`
	LastProcessingTime    time.Time       `json:"last_processing_time"`
	WorkerUtilization     map[int]float64 `json:"worker_utilization"`
	ProcessingRate        float64         `json:"processing_rate_per_minute"`
	ErrorRate             float64         `json:"error_rate_percentage"`
}

// NewSchedulerMetrics creates a new metrics instance
func NewSchedulerMetrics() *SchedulerMetrics {
	return &SchedulerMetrics{
		WorkerUtilization: make(map[int]float64),
	}
}

// RecordReminderProcessed records a successful reminder processing with duration
func (m *SchedulerMetrics) RecordReminderProcessed(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RemindersProcessed++
	m.LastProcessingTime = time.Now()
	m.totalProcessingTime += duration
	m.processingCycles++

	if m.processingCycles > 0 {
		m.AverageProcessingTime = m.totalProcessingTime / time.Duration(m.processingCycles)
	}
}

// RecordNudgeCreated increments the nudge creation counter
func (m *SchedulerMetrics) RecordNudgeCreated() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.NudgesCreated++
}

// RecordProcessingError increments the error counter
func (m *SchedulerMetrics) RecordProcessingError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ProcessingErrors++
}

// RecordWorkerActivity updates worker utilization metrics
func (m *SchedulerMetrics) RecordWorkerActivity(workerID int, active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if active {
		m.WorkerUtilization[workerID] = 1.0
	} else {
		m.WorkerUtilization[workerID] = 0.0
	}
}

// IsHealthy determines if the scheduler is healthy based on metrics
func (m *SchedulerMetrics) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Consider healthy if:
	// 1. Processing occurred within the last 5 minutes
	// 2. Error rate is below 50%

	recentProcessing := time.Since(m.LastProcessingTime) < 5*time.Minute

	errorRate := m.calculateErrorRate()
	lowErrorRate := errorRate < 0.5

	return recentProcessing && lowErrorRate
}

// GetHealthStatus returns detailed health information
func (m *SchedulerMetrics) GetHealthStatus() HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return HealthStatus{
		IsHealthy:             m.IsHealthy(),
		LastProcessingTime:    m.LastProcessingTime,
		ProcessingErrors:      m.ProcessingErrors,
		AverageProcessingTime: m.AverageProcessingTime.String(),
		ErrorRate:             m.calculateErrorRate(),
	}
}

// GetMetricsSummary returns a comprehensive metrics summary
func (m *SchedulerMetrics) GetMetricsSummary() MetricsSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	processingRate := m.calculateProcessingRate()
	errorRate := m.calculateErrorRate()

	return MetricsSummary{
		RemindersProcessed:    m.RemindersProcessed,
		NudgesCreated:         m.NudgesCreated,
		ProcessingErrors:      m.ProcessingErrors,
		AverageProcessingTime: m.AverageProcessingTime.String(),
		LastProcessingTime:    m.LastProcessingTime,
		WorkerUtilization:     m.copyWorkerUtilization(),
		ProcessingRate:        processingRate,
		ErrorRate:             errorRate * 100, // Convert to percentage
	}
}

// calculateErrorRate computes the error rate as a percentage
func (m *SchedulerMetrics) calculateErrorRate() float64 {
	total := m.RemindersProcessed + m.ProcessingErrors
	if total == 0 {
		return 0.0
	}
	return float64(m.ProcessingErrors) / float64(total)
}

// calculateProcessingRate computes reminders processed per minute
func (m *SchedulerMetrics) calculateProcessingRate() float64 {
	if m.LastProcessingTime.IsZero() {
		return 0.0
	}

	// Calculate rate based on time since first processing
	duration := time.Since(m.LastProcessingTime)
	if duration == 0 {
		return 0.0
	}

	minutes := duration.Minutes()
	if minutes == 0 {
		return 0.0
	}

	return float64(m.RemindersProcessed) / minutes
}

// copyWorkerUtilization creates a copy of worker utilization map
func (m *SchedulerMetrics) copyWorkerUtilization() map[int]float64 {
	copy := make(map[int]float64)
	for k, v := range m.WorkerUtilization {
		copy[k] = v
	}
	return copy
}

// Reset resets all metrics to zero
func (m *SchedulerMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RemindersProcessed = 0
	m.NudgesCreated = 0
	m.ProcessingErrors = 0
	m.AverageProcessingTime = 0
	m.LastProcessingTime = time.Time{}
	m.totalProcessingTime = 0
	m.processingCycles = 0
	m.WorkerUtilization = make(map[int]float64)
}
