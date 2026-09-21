package waitingroom

import (
	"testing"
)

func TestQueueStatus_String(t *testing.T) {
	tests := []struct {
		status   QueueStatus
		expected string
	}{
		{QueueGranted, "GRANTED"},
		{QueueWaiting, "WAITING"},
		{QueueAlreadyActive, "ALREADY_ACTIVE"},
		{QueueAlreadyQueued, "ALREADY_QUEUED"},
		{QueueStatus(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("QueueStatus.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWaitingRoom_SetAlgorithmParams(t *testing.T) {
	wr := NewWaitingRoom(nil, 100, 300, 3600)
	if wr.heartbeatTTL != 60 || wr.weight != 100 || wr.maxRandom != 3000 {
		t.Errorf("unexpected defaults: heartbeatTTL=%d, weight=%d, maxRandom=%d",
			wr.heartbeatTTL, wr.weight, wr.maxRandom)
	}

	wr.SetAlgorithmParams(120, 200, 5000)
	if wr.heartbeatTTL != 120 || wr.weight != 200 || wr.maxRandom != 5000 {
		t.Errorf("SetAlgorithmParams failed: heartbeatTTL=%d, weight=%d, maxRandom=%d",
			wr.heartbeatTTL, wr.weight, wr.maxRandom)
	}
}
