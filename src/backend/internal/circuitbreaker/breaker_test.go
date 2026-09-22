package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedNormalExecution(t *testing.T) {
	cb := NewCircuitBreaker("test-cb", 0.5, 5*time.Second, 1*time.Second)

	executed := 0
	err := cb.Execute(func() error {
		executed++
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if executed != 1 {
		t.Fatalf("expected executed == 1, got %d", executed)
	}
	if cb.GetState() != StateClosed {
		t.Fatalf("expected StateClosed, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_TripsToOpenOnFailures(t *testing.T) {
	cb := NewCircuitBreaker("test-cb", 0.5, 5*time.Second, 1*time.Second)
	cb.SetMinRequests(5)

	customErr := errors.New("upstream service down")

	// 5 requests: 3 failures, 2 successes -> error rate 60% >= 50%
	for i := 0; i < 5; i++ {
		if i < 3 {
			_ = cb.Execute(func() error { return customErr })
		} else {
			_ = cb.Execute(func() error { return nil })
		}
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("expected StateOpen after 60%% failure rate, got %v", cb.GetState())
	}

	// 6th request should be rejected immediately without calling fn
	called := false
	err := cb.Execute(func() error {
		called = true
		return nil
	})

	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	if called {
		t.Fatalf("fn should not have been called when circuit is OPEN")
	}
}

func TestCircuitBreaker_HalfOpenConcurrencySafe_MaxProbes(t *testing.T) {
	// sleepWindow = 50ms, maxProbes = 3
	cb := NewCircuitBreaker("test-probe", 0.5, 5*time.Second, 50*time.Millisecond)
	cb.SetMinRequests(5)
	cb.SetMaxProbes(3)

	// Trip to OPEN
	for i := 0; i < 5; i++ {
		_ = cb.Execute(func() error { return errors.New("err") })
	}
	if cb.GetState() != StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.GetState())
	}

	// Wait for sleep window to expire
	time.Sleep(70 * time.Millisecond)

	// Launch 20 concurrent goroutines at once
	var executedCount int32
	var rejectedCount int32
	var wg sync.WaitGroup

	gate := make(chan struct{})

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate // Release all at once

			err := cb.Execute(func() error {
				atomic.AddInt32(&executedCount, 1)
				// Simulate some work so that all goroutines hit beforeRequest
				time.Sleep(20 * time.Millisecond)
				return nil
			})

			if errors.Is(err, ErrCircuitOpen) {
				atomic.AddInt32(&rejectedCount, 1)
			}
		}()
	}

	// Open the gate
	close(gate)
	wg.Wait()

	// Exactly 3 probes must have been allowed through, and 17 rejected!
	if atomic.LoadInt32(&executedCount) != 3 {
		t.Fatalf("Race condition detected! Expected exactly 3 probes to execute, got %d", executedCount)
	}
	if atomic.LoadInt32(&rejectedCount) != 17 {
		t.Fatalf("Expected 17 requests to be rejected with ErrCircuitOpen, got %d", rejectedCount)
	}

	// Since 3 probes succeeded (>= requiredSuccesses 2), state should now be StateClosed
	if cb.GetState() != StateClosed {
		t.Fatalf("expected StateClosed after successful probes, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker("test-probe-fail", 0.5, 5*time.Second, 50*time.Millisecond)
	cb.SetMinRequests(5)
	cb.SetMaxProbes(3)

	// Trip to OPEN
	for i := 0; i < 5; i++ {
		_ = cb.Execute(func() error { return errors.New("err") })
	}
	time.Sleep(70 * time.Millisecond)

	// Probe fails
	err := cb.Execute(func() error {
		return errors.New("probe still broken")
	})

	if err == nil || err.Error() != "probe still broken" {
		t.Fatalf("expected probe error, got %v", err)
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("expected StateOpen immediately after probe failed, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_IgnoredErrors(t *testing.T) {
	cb := NewCircuitBreaker("test-ignored", 0.5, 5*time.Second, 1*time.Second)
	cb.SetMinRequests(5)

	// Custom ignore function: ignore client cancellation
	cb.SetIgnoredError(func(err error) bool {
		return errors.Is(err, context.Canceled)
	})

	// 10 consecutive client cancellations
	for i := 0; i < 10; i++ {
		_ = cb.Execute(func() error {
			return context.Canceled
		})
	}

	// State should remain CLOSED because context.Canceled is ignored
	if cb.GetState() != StateClosed {
		t.Fatalf("circuit should remain CLOSED for ignored errors, got %v", cb.GetState())
	}

	total, failures, _ := cb.Counts()
	if failures != 0 || total != 0 {
		t.Fatalf("expected 0 recorded failures/total for ignored errors, got failures=%d, total=%d", failures, total)
	}
}

func TestCircuitBreaker_PanicRecovery(t *testing.T) {
	cb := NewCircuitBreaker("test-panic", 0.5, 5*time.Second, 1*time.Second)
	cb.SetMinRequests(1)
	cb.threshold = 0.1 // Trip on 1 failure

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic to be re-thrown")
			}
		}()

		_ = cb.Execute(func() error {
			panic("nil pointer dereference inside downstream")
		})
	}()

	// Breaker should have safely recorded the failure and transitioned to OPEN
	if cb.GetState() != StateOpen {
		t.Fatalf("expected StateOpen after downstream panic, got %v", cb.GetState())
	}
}
