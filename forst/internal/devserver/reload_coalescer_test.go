package devserver

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestReloadCoalescer_burstSchedulesOneFollowUp(t *testing.T) {
	var runs atomic.Int32
	block := make(chan struct{})

	c := newReloadCoalescer(func() {
		runs.Add(1)
		<-block
	})

	go c.schedule()
	time.Sleep(30 * time.Millisecond)
	if runs.Load() != 1 {
		t.Fatalf("running count=%d want 1", runs.Load())
	}

	if c.schedule() {
		t.Fatal("expected coalesced schedule while running")
	}
	c.schedule()
	c.schedule()

	close(block)
	deadline := time.Now().Add(500 * time.Millisecond)
	for runs.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if runs.Load() != 2 {
		t.Fatalf("follow-up reload count=%d want 2", runs.Load())
	}

	time.Sleep(50 * time.Millisecond)
	if runs.Load() != 2 {
		t.Fatalf("extra reloads after coalesced burst: %d", runs.Load())
	}
}

func TestReloadCoalescer_runSync(t *testing.T) {
	var runs atomic.Int32
	c := newReloadCoalescer(func() { runs.Add(1) })
	c.runSync()
	if runs.Load() != 1 {
		t.Fatalf("sync count=%d want 1", runs.Load())
	}
}

func TestReloadCoalescer_scheduleReturnsFalseWhileRunning(t *testing.T) {
	block := make(chan struct{})
	c := newReloadCoalescer(func() {
		<-block
	})

	go func() {
		c.schedule()
	}()

	time.Sleep(20 * time.Millisecond)
	if c.schedule() {
		t.Fatal("expected schedule to coalesce while running")
	}
	close(block)
	time.Sleep(20 * time.Millisecond)
}
