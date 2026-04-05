package lsp

import "sync"

const defaultPackageAnalysisCacheMaxEntries = 64

// packageAnalysisLRU stores merged package snapshots keyed by content fingerprint.
// When full, the least-recently-used entry is evicted.
type packageAnalysisLRU struct {
	mu     sync.Mutex
	max    int
	m      map[string]*packageSnapshot
	order  []string
}

func newPackageAnalysisLRU(maxEntries int) *packageAnalysisLRU {
	if maxEntries < 1 {
		maxEntries = defaultPackageAnalysisCacheMaxEntries
	}
	return &packageAnalysisLRU{
		max:   maxEntries,
		m:     make(map[string]*packageSnapshot),
		order: make([]string, 0, maxEntries),
	}
}

func (c *packageAnalysisLRU) get(key string) *packageSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	snap, ok := c.m[key]
	if !ok {
		return nil
	}
	// move key to MRU end
	for i, k := range c.order {
		if k == key {
			copy(c.order[i:], c.order[i+1:])
			c.order = c.order[:len(c.order)-1]
			break
		}
	}
	c.order = append(c.order, key)
	return snap
}

func (c *packageAnalysisLRU) put(key string, snap *packageSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.m[key]; exists {
		for i, k := range c.order {
			if k == key {
				copy(c.order[i:], c.order[i+1:])
				c.order = c.order[:len(c.order)-1]
				break
			}
		}
	}
	c.m[key] = snap
	c.order = append(c.order, key)
	for len(c.order) > c.max {
		evict := c.order[0]
		c.order = c.order[1:]
		delete(c.m, evict)
	}
}

func (c *packageAnalysisLRU) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[string]*packageSnapshot)
	c.order = c.order[:0]
}
