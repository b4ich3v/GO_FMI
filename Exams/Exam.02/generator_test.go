package main

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func waitClosedWithTimeout[T any](t *testing.T, ch <-chan T, timeout time.Duration) {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline.C:
			t.Fatalf("channel did not close within %v", timeout)
		}
	}
}

func TestGenerator_EmitsExpectedPrefixAndStopsOnCancel(t *testing.T) {
	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	out := Generator(ctx, 3)

	select {
	case v, ok := <-out:
		if !ok {
			t.Fatalf("generator output closed unexpectedly")
		}
		if v != "3_1" {
			t.Fatalf("unexpected first value: got %q want %q", v, "3_1")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for first generator value")
	}

	cancel()
	waitClosedWithTimeout(t, out, 500*time.Millisecond)

	for i := 0; i < 50; i++ {
		time.Sleep(5 * time.Millisecond)
		if runtime.NumGoroutine() <= baseline+2 {
			return
		}
	}

	now := runtime.NumGoroutine()
	if now > baseline+2 {
		t.Fatalf("possible goroutine leak: baseline=%d now=%d", baseline, now)
	}
}
