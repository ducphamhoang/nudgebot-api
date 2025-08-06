package common

import "time"

// Clock provides an abstraction over time operations to enable deterministic testing
type Clock interface {
	// Now returns the current time
	Now() time.Time
	// After returns a channel that delivers the current time after the specified duration
	After(duration time.Duration) <-chan time.Time
	// Sleep pauses the current goroutine for the specified duration
	Sleep(duration time.Duration)
}

// RealClock implements Clock using the standard time package
type RealClock struct{}

// NewRealClock creates a new RealClock instance
func NewRealClock() *RealClock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}

func (c *RealClock) After(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}

func (c *RealClock) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// MockClock implements Clock for testing with controllable time
type MockClock struct {
	currentTime time.Time
	timers      []*mockTimer
}

type mockTimer struct {
	deadline time.Time
	channel  chan time.Time
}

// NewMockClock creates a new MockClock with the specified initial time
func NewMockClock(initialTime time.Time) *MockClock {
	return &MockClock{
		currentTime: initialTime,
		timers:      make([]*mockTimer, 0),
	}
}

func (c *MockClock) Now() time.Time {
	return c.currentTime
}

func (c *MockClock) After(duration time.Duration) <-chan time.Time {
	deadline := c.currentTime.Add(duration)
	ch := make(chan time.Time, 1)

	timer := &mockTimer{
		deadline: deadline,
		channel:  ch,
	}

	c.timers = append(c.timers, timer)

	// If the deadline is already past, fire immediately
	if !deadline.After(c.currentTime) {
		go func() {
			ch <- c.currentTime
		}()
	}

	return ch
}

func (c *MockClock) Sleep(duration time.Duration) {
	// In tests, we don't actually want to sleep, so this is a no-op
	// The test can call Advance() to simulate time passing
}

// Advance moves the mock clock forward by the specified duration
// and triggers any timers that should fire
func (c *MockClock) Advance(duration time.Duration) {
	c.currentTime = c.currentTime.Add(duration)

	// Fire any timers that should trigger
	for _, timer := range c.timers {
		if !timer.deadline.After(c.currentTime) {
			select {
			case timer.channel <- c.currentTime:
			default:
				// Channel already has a value or is closed
			}
		}
	}
}

// SetTime sets the mock clock to a specific time
func (c *MockClock) SetTime(t time.Time) {
	c.currentTime = t

	// Fire any timers that should trigger
	for _, timer := range c.timers {
		if !timer.deadline.After(c.currentTime) {
			select {
			case timer.channel <- c.currentTime:
			default:
				// Channel already has a value or is closed
			}
		}
	}
}

// FastForward advances the clock to trigger the next pending timer
// Returns true if a timer was triggered, false if no timers are pending
func (c *MockClock) FastForward() bool {
	if len(c.timers) == 0 {
		return false
	}

	// Find the earliest timer
	var earliest *mockTimer
	for _, timer := range c.timers {
		if timer.deadline.After(c.currentTime) {
			if earliest == nil || timer.deadline.Before(earliest.deadline) {
				earliest = timer
			}
		}
	}

	if earliest == nil {
		return false
	}

	// Advance to the timer's deadline
	c.SetTime(earliest.deadline)
	return true
}
