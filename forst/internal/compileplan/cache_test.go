package compileplan

import "testing"

func TestCompiledFunctionCache_invalidateDropsStaleGeneration(t *testing.T) {
	c := NewMemoryFunctionCache()
	c.Put(1, "auth", "Expire", "code-v1")
	c.Invalidate(2)
	if _, ok := c.Get(1, "auth", "Expire"); ok {
		t.Fatal("generation 1 entry should be gone after invalidate(2)")
	}
	c.Put(2, "auth", "Expire", "code-v2")
	got, ok := c.Get(2, "auth", "Expire")
	if !ok || got != "code-v2" {
		t.Fatalf("want code-v2, got %q ok=%v", got, ok)
	}
}
