package presence

import (
	"context"
	"testing"
)

func TestNewPresenceTracker_Defaults(t *testing.T) {
	pt := NewPresenceTracker(nil, 0)
	if pt.windowMinutes != 3 {
		t.Fatalf("expected default windowMinutes=3, got %d", pt.windowMinutes)
	}

	ptCustom := NewPresenceTracker(nil, 5)
	if ptCustom.windowMinutes != 5 {
		t.Fatalf("expected windowMinutes=5, got %d", ptCustom.windowMinutes)
	}
}

func TestPresenceTracker_NilRedisSafety(t *testing.T) {
	pt := NewPresenceTracker(nil, 3)
	ctx := context.Background()

	// Should safely return nil / 0 without panicking when Redis is nil
	if err := pt.RecordPresence(ctx, "user-123", "ws-456"); err != nil {
		t.Fatalf("unexpected error on nil redis: %v", err)
	}

	globalCount, err := pt.GetGlobalActiveUsers(ctx)
	if err != nil || globalCount != 0 {
		t.Fatalf("expected 0 count with nil redis, got %d (err: %v)", globalCount, err)
	}

	wsCount, err := pt.GetWorkshopActiveUsers(ctx, "ws-456")
	if err != nil || wsCount != 0 {
		t.Fatalf("expected 0 count with nil redis, got %d (err: %v)", wsCount, err)
	}

	workshops, err := pt.GetActiveWorkshops(ctx)
	if err != nil || len(workshops) != 0 {
		t.Fatalf("expected empty workshops with nil redis, got %v (err: %v)", workshops, err)
	}
}
