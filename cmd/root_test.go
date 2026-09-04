package cmd

import (
	"context"
	"testing"
	"time"
)

func TestServerQUICConfigDisables0RTT(t *testing.T) {
	if newServerQUICConfig().Allow0RTT {
		t.Fatal("0-RTT must stay disabled for non-idempotent HTTP endpoints")
	}
}

func TestWaitForServerExitJoinsCompletedListener(t *testing.T) {
	done := make(chan error, 1)
	done <- nil
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	waitForServerExit(ctx, "test", done)
}

func TestWaitForServerExitHonorsShutdownDeadline(t *testing.T) {
	done := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	started := time.Now()
	waitForServerExit(ctx, "test", done)
	if elapsed := time.Since(started); elapsed < 20*time.Millisecond {
		t.Fatalf("waitForServerExit returned too early: %s", elapsed)
	}
}
