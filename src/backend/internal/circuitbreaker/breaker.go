package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
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

type windowBucket struct {
	startTime time.Time
	failures  int
	successes int
}

// CircuitBreaker implements the Circuit Breaker pattern with sliding window and concurrency-safe half-open state.
type CircuitBreaker struct {
	mu sync.Mutex

	name              string
	state             State
	threshold         float64       // Error rate threshold (0.5 = 50%)
	windowSize        time.Duration // Evaluation window
	sleepWindow       time.Duration // Time to wait before half-open
	maxProbes         int           // Max requests allowed in half-open state
	minRequests       int           // Minimum requests in window before evaluating error rate
	requiredSuccesses int           // Consecutive successes in half-open to close

	// Half-open state tracking (concurrency safe)
	probesSent        int
	halfOpenSuccesses int

	// Sliding window tracking
	buckets        []windowBucket
	bucketDuration time.Duration

	lastFailureTime time.Time
	lastStateChange time.Time

	// Error classifier: if set, returns true if an error should NOT count as failure (ignored)
	isIgnored func(err error) bool
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, threshold float64, windowSize, sleepWindow time.Duration) *CircuitBreaker {
	numBuckets := 6
	bucketDuration := windowSize / time.Duration(numBuckets)
	if bucketDuration <= 0 {
		bucketDuration = time.Second
	}

	return &CircuitBreaker{
		name:              name,
		state:             StateClosed,
		threshold:         threshold,
		windowSize:        windowSize,
		sleepWindow:       sleepWindow,
		maxProbes:         3,
		minRequests:       5,
		requiredSuccesses: 2,
		bucketDuration:    bucketDuration,
		lastStateChange:   time.Now(),
		isIgnored: func(err error) bool {
			// By default, ignore client cancellation
			return errors.Is(err, context.Canceled)
		},
	}
}

// SetIgnoredError sets a custom function to determine if an error should be ignored (not counted as failure)
func (cb *CircuitBreaker) SetIgnoredError(fn func(err error) bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.isIgnored = fn
}

// SetMaxProbes configures max probes in half-open state
func (cb *CircuitBreaker) SetMaxProbes(probes int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.maxProbes = probes
}

// SetMinRequests configures min requests before error rate calculation
func (cb *CircuitBreaker) SetMinRequests(min int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.minRequests = min
}

// Execute runs the given function through the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() error) (err error) {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	panicked := true
	defer func() {
		if panicked {
			if r := recover(); r != nil {
				cb.afterRequest(fmt.Errorf("panic in circuit breaker: %v", r))
				panic(r)
			}
			cb.afterRequest(errors.New("abrupt execution termination"))
		}
	}()

	err = fn()
	panicked = false
	cb.afterRequest(err)
	return err
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if sleep window has elapsed
		if now.Sub(cb.lastStateChange) > cb.sleepWindow {
			cb.setState(StateHalfOpen)
			cb.resetCounters()
			// The request triggering this transition takes probe slot 1
			cb.probesSent = 1
			log.Printf("[CIRCUIT_BREAKER] %s: transitioning to HALF_OPEN (probe 1/%d)", cb.name, cb.maxProbes)
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Atomically check and reserve a probe slot under lock to prevent race condition
		if cb.probesSent >= cb.maxProbes {
			return ErrCircuitOpen
		}
		cb.probesSent++
		log.Printf("[CIRCUIT_BREAKER] %s: probe %d/%d allowed in HALF_OPEN", cb.name, cb.probesSent, cb.maxProbes)
		return nil
	}

	return nil
}

func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	var isSuccess, isFailure, isIgnored bool
	if err == nil {
		isSuccess = true
	} else if cb.isIgnored != nil && cb.isIgnored(err) {
		isIgnored = true
	} else {
		isFailure = true
	}

	if isFailure {
		cb.lastFailureTime = now
	}

	switch cb.state {
	case StateClosed:
		if isIgnored {
			return
		}
		cb.recordResult(now, isFailure)

		total, failures, _ := cb.currentCounts(now)
		if total >= cb.minRequests {
			errorRate := float64(failures) / float64(total)
			if errorRate >= cb.threshold {
				cb.setState(StateOpen)
				cb.resetCounters()
				log.Printf("[CIRCUIT_BREAKER] %s: OPENED (error rate: %.2f%%, failures: %d/%d)",
					cb.name, errorRate*100, failures, total)
			}
		}

	case StateHalfOpen:
		if isIgnored {
			// Ignored error (e.g. client canceled): release probe slot so another request can probe
			if cb.probesSent > 0 {
				cb.probesSent--
			}
			return
		}

		if isFailure {
			// Any probe failure sends circuit immediately back to OPEN
			cb.setState(StateOpen)
			cb.resetCounters()
			log.Printf("[CIRCUIT_BREAKER] %s: probe failed (%v), returning to OPEN", cb.name, err)
		} else if isSuccess {
			cb.halfOpenSuccesses++
			if cb.halfOpenSuccesses >= cb.requiredSuccesses {
				log.Printf("[CIRCUIT_BREAKER] %s: CLOSED (probes successful: %d/%d)",
					cb.name, cb.halfOpenSuccesses, cb.requiredSuccesses)
				cb.setState(StateClosed)
				cb.resetCounters()
			}
		}

	case StateOpen:
		// Stale probe completing after breaker already opened
	}
}

func (cb *CircuitBreaker) recordResult(now time.Time, isFailure bool) {
	cb.pruneBuckets(now)

	if len(cb.buckets) > 0 && now.Sub(cb.buckets[len(cb.buckets)-1].startTime) < cb.bucketDuration {
		lastIdx := len(cb.buckets) - 1
		if isFailure {
			cb.buckets[lastIdx].failures++
		} else {
			cb.buckets[lastIdx].successes++
		}
	} else {
		newBucket := windowBucket{startTime: now}
		if isFailure {
			newBucket.failures = 1
		} else {
			newBucket.successes = 1
		}
		cb.buckets = append(cb.buckets, newBucket)
	}
}

func (cb *CircuitBreaker) pruneBuckets(now time.Time) {
	cutoff := now.Add(-cb.windowSize)
	firstValid := -1
	for i, b := range cb.buckets {
		if b.startTime.After(cutoff) {
			firstValid = i
			break
		}
	}
	if firstValid == -1 {
		cb.buckets = nil
	} else if firstValid > 0 {
		cb.buckets = cb.buckets[firstValid:]
	}
}

func (cb *CircuitBreaker) currentCounts(now time.Time) (total, failures, successes int) {
	cb.pruneBuckets(now)
	for _, b := range cb.buckets {
		failures += b.failures
		successes += b.successes
	}
	total = failures + successes
	return
}

func (cb *CircuitBreaker) setState(state State) {
	cb.state = state
	cb.lastStateChange = time.Now()
}

func (cb *CircuitBreaker) resetCounters() {
	cb.buckets = nil
	cb.probesSent = 0
	cb.halfOpenSuccesses = 0
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

// Counts returns the current total, failures, and successes in the sliding window
func (cb *CircuitBreaker) Counts() (total, failures, successes int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.currentCounts(time.Now())
}
