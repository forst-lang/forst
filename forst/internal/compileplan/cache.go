package compileplan

import (
	"sync"
)

// CompiledFunctionCache stores emitted Go per (generation, package, function).
type CompiledFunctionCache interface {
	Get(gen uint64, pkg, fn string) (string, bool)
	Put(gen uint64, pkg, fn, code string)
	Invalidate(generation uint64)
}

// MemoryFunctionCache is an in-memory CompiledFunctionCache.
type MemoryFunctionCache struct {
	mu    sync.RWMutex
	items map[cacheKey]string
	gen   uint64
}

type cacheKey struct {
	gen uint64
	pkg string
	fn  string
}

// NewMemoryFunctionCache returns an empty function cache.
func NewMemoryFunctionCache() *MemoryFunctionCache {
	return &MemoryFunctionCache{items: make(map[cacheKey]string)}
}

func (c *MemoryFunctionCache) Get(gen uint64, pkg, fn string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if gen < c.gen {
		return "", false
	}
	code, ok := c.items[cacheKey{gen: gen, pkg: pkg, fn: fn}]
	return code, ok
}

func (c *MemoryFunctionCache) Put(gen uint64, pkg, fn, code string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if gen < c.gen {
		return
	}
	c.items[cacheKey{gen: gen, pkg: pkg, fn: fn}] = code
}

func (c *MemoryFunctionCache) Invalidate(generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if generation <= c.gen {
		return
	}
	c.gen = generation
	c.items = make(map[cacheKey]string)
}
