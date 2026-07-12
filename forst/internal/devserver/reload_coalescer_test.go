package devserver

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
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

func TestReloadCoalescer_runSync_coalescesConcurrentSchedule(t *testing.T) {
	var runs atomic.Int32
	block := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	c := newReloadCoalescer(func() {
		once.Do(func() { close(started) })
		runs.Add(1)
		<-block
	})

	go c.runSync()
	<-started

	if c.schedule() {
		t.Fatal("expected schedule to coalesce during runSync")
	}

	close(block)
	deadline := time.Now().Add(500 * time.Millisecond)
	for runs.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if runs.Load() != 2 {
		t.Fatalf("follow-up after runSync count=%d want 2", runs.Load())
	}
}

func TestIsRelevantWatchOp(t *testing.T) {
	if !isRelevantWatchOp(fsnotify.Write) {
		t.Fatal("Write should be relevant")
	}
	if !isRelevantWatchOp(fsnotify.Create) {
		t.Fatal("Create should be relevant")
	}
	if !isRelevantWatchOp(fsnotify.Rename) {
		t.Fatal("Rename should be relevant")
	}
	if isRelevantWatchOp(fsnotify.Chmod) {
		t.Fatal("Chmod alone should not be relevant")
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
