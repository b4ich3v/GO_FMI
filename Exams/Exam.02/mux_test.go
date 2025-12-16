package main

import (
	"context"
	"runtime"
	"sort"
	"testing"
	"time"
)

func drainWithTimeout[T any](t *testing.T, ch <-chan T, timeout time.Duration) []T {
	t.Helper()
	var out []T
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, v)
		case <-timer.C:
			t.Fatalf("timed out while draining channel (%v)", timeout)
		}
	}
}

func TestMultiplex_ForwardsAllAndCloses(t *testing.T) {
	ctx := context.Background()

	ch1 := make(chan string, 4)
	ch2 := make(chan string, 4)

	for _, v := range []string{"1_1", "1_2", "1_3"} {
		ch1 <- v
	}
	close(ch1)

	for _, v := range []string{"2_1", "2_2"} {
		ch2 <- v
	}
	close(ch2)

	out := Multiplex(ctx, ch1, ch2)
	got := drainWithTimeout(t, out, 2*time.Second)

	// Order across different input channels is not deterministic, so compare as sets.
	sort.Strings(got)
	want := []string{"1_1", "1_2", "1_3", "2_1", "2_2"}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d\n got=%v\nwant=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mismatch at %d: got %q, want %q\n got=%v\nwant=%v", i, got[i], want[i], got, want)
		}
	}
}

func TestMultiplex_CancelClosesOutEvenIfInputsNeverClose(t *testing.T) {
	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	never1 := make(chan string)
	never2 := make(chan string)

	out := Multiplex(ctx, never1, never2)

	time.AfterFunc(50*time.Millisecond, cancel)
	_ = drainWithTimeout(t, out, 2*time.Second)

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
