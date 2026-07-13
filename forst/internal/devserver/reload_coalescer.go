package devserver

import "sync"

// reloadCoalescer runs fn serially and collapses burst schedule() calls while
// a reload is in progress into at most one follow-up reload.
type reloadCoalescer struct {
	mu      sync.Mutex
	running bool
	pending bool
	fn      func()
}

func newReloadCoalescer(fn func()) *reloadCoalescer {
	return &reloadCoalescer{fn: fn}
}

// runSync executes fn once on the caller goroutine (initial startup).
// Concurrent schedule() calls coalesce into one follow-up reload.
func (c *reloadCoalescer) runSync() {
	if c == nil || c.fn == nil {
		return
	}
	c.mu.Lock()
	c.running = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		if c.pending {
			c.pending = false
			c.mu.Unlock()
			go c.loop()
			return
		}
		c.running = false
		c.mu.Unlock()
	}()
	c.fn()
}

// schedule starts fn asynchronously, or marks a single pending follow-up if already running.
func (c *reloadCoalescer) schedule() bool {
	if c == nil || c.fn == nil {
		return false
	}
	c.mu.Lock()
	if c.running {
		c.pending = true
		c.mu.Unlock()
		return false
	}
	c.running = true
	c.mu.Unlock()
	go c.loop()
	return true
}

func (c *reloadCoalescer) loop() {
	for {
		c.fn()

		c.mu.Lock()
		if !c.pending {
			c.running = false
			c.mu.Unlock()
			return
		}
		c.pending = false
		c.mu.Unlock()
	}
}
