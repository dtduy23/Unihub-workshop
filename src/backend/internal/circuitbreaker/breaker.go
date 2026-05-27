package circuitbreaker

import (
	"errors"
	"log"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed   State = iota // Normal operation
	StateOpen                  // Rejecting all requests
	StateHalfOpen              // Allowing probe requests
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitBreaker implements the Circuit Breaker pattern
type CircuitBreaker struct {
	mu sync.Mutex

	name          string
	state         State
	failureCount  int
	successCount  int
	totalCount    int
	threshold     float64       // Error rate threshold (0.5 = 50%)
	windowSize    time.Duration // Evaluation window
	sleepWindow   time.Duration // Time to wait before half-open
	maxProbes     int           // Max requests allowed in half-open state

	lastFailureTime time.Time
	lastStateChange time.Time
	windowStart     time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, threshold float64, windowSize, sleepWindow time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:        name,
		state:       StateClosed,
		threshold:   threshold,
		windowSize:  windowSize,
		sleepWindow: sleepWindow,
		maxProbes:   3,
		windowStart: time.Now(),
		lastStateChange: time.Now(),
	}
}

// Execute runs the given function through the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	err := fn()

	cb.afterRequest(err)
	return err
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		// Reset window if expired
		if now.Sub(cb.windowStart) > cb.windowSize {
			cb.resetCounters()
			cb.windowStart = now
		}
		return nil

	case StateOpen:
		// Check if sleep window has elapsed
		if now.Sub(cb.lastStateChange) > cb.sleepWindow {
			cb.setState(StateHalfOpen)
			cb.resetCounters()
			log.Printf("[CIRCUIT_BREAKER] %s: transitioning to HALF_OPEN", cb.name)
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		if cb.totalCount >= cb.maxProbes {
			return ErrCircuitOpen
		}
		return nil
	}

	return nil
}

func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalCount++

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		switch cb.state {
		case StateClosed:
			if cb.totalCount >= 5 { // Minimum sample size
				errorRate := float64(cb.failureCount) / float64(cb.totalCount)
				if errorRate >= cb.threshold {
					cb.setState(StateOpen)
					log.Printf("[CIRCUIT_BREAKER] %s: OPENED (error rate: %.2f%%)", cb.name, errorRate*100)
				}
			}
		case StateHalfOpen:
			cb.setState(StateOpen)
			log.Printf("[CIRCUIT_BREAKER] %s: probe failed, returning to OPEN", cb.name)
		}
	} else {
		cb.successCount++

		if cb.state == StateHalfOpen {
			if cb.successCount >= 2 { // Need 2 consecutive successes to close
				cb.setState(StateClosed)
				cb.resetCounters()
				log.Printf("[CIRCUIT_BREAKER] %s: CLOSED (probes successful)", cb.name)
			}
		}
	}
}

func (cb *CircuitBreaker) setState(state State) {
	cb.state = state
	cb.lastStateChange = time.Now()
}

func (cb *CircuitBreaker) resetCounters() {
	cb.failureCount = 0
	cb.successCount = 0
	cb.totalCount = 0
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// IsOpen returns true if the circuit breaker is in the Open state
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.GetState() == StateOpen
}
