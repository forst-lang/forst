package lsp

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

const defaultFileParseCacheMaxEntries = 128

// fileParseCacheEntry stores a parse result keyed by content hash for one URI.
type fileParseCacheEntry struct {
	contentHash string
	result      fileParseResult
}

// fileParseCache reuses lex/parse output for unchanged package-group members.
type fileParseCache struct {
	mu    sync.Mutex
	max   int
	m     map[string]fileParseCacheEntry
	order []string
}

func newFileParseCache(maxEntries int) *fileParseCache {
	if maxEntries < 1 {
		maxEntries = defaultFileParseCacheMaxEntries
	}
	return &fileParseCache{
		max:   maxEntries,
		m:     make(map[string]fileParseCacheEntry),
		order: make([]string, 0, maxEntries),
	}
}

func contentSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func (c *fileParseCache) get(uri, contentHash string) (fileParseResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	canon := canonicalFileURI(uri)
	e, ok := c.m[canon]
	if !ok || e.contentHash != contentHash {
		return fileParseResult{}, false
	}
	for i, k := range c.order {
		if k == canon {
			copy(c.order[i:], c.order[i+1:])
			c.order = c.order[:len(c.order)-1]
			break
		}
	}
	c.order = append(c.order, canon)
	return e.result, true
}

func (c *fileParseCache) put(uri, contentHash string, result fileParseResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	canon := canonicalFileURI(uri)
	if _, exists := c.m[canon]; exists {
		for i, k := range c.order {
			if k == canon {
				copy(c.order[i:], c.order[i+1:])
				c.order = c.order[:len(c.order)-1]
				break
			}
		}
	} else if len(c.order) >= c.max {
		evict := c.order[0]
		c.order = c.order[1:]
		delete(c.m, evict)
	}
	c.m[canon] = fileParseCacheEntry{contentHash: contentHash, result: result}
	c.order = append(c.order, canon)
}

func (c *fileParseCache) remove(uri string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	canon := canonicalFileURI(uri)
	if _, ok := c.m[canon]; !ok {
		return
	}
	delete(c.m, canon)
	for i, k := range c.order {
		if k == canon {
			copy(c.order[i:], c.order[i+1:])
			c.order = c.order[:len(c.order)-1]
			return
		}
	}
}
