package lsp

import "testing"

func TestPackageAnalysisLRU_evictsOldest(t *testing.T) {
	t.Parallel()
	c := newPackageAnalysisLRU(3)
	snap := &packageSnapshot{uris: []string{"a"}}
	c.put("k1", snap)
	c.put("k2", snap)
	c.put("k3", snap)
	c.put("k4", snap)
	if c.get("k1") != nil {
		t.Fatal("expected k1 evicted")
	}
	if c.get("k2") == nil || c.get("k3") == nil || c.get("k4") == nil {
		t.Fatal("expected k2,k3,k4 retained")
	}
}

func TestPackageAnalysisLRU_getMovesToMRU(t *testing.T) {
	t.Parallel()
	c := newPackageAnalysisLRU(3)
	s := &packageSnapshot{}
	c.put("a", s)
	c.put("b", s)
	c.put("c", s)
	_ = c.get("a")
	c.put("d", s)
	if c.get("b") != nil {
		t.Fatal("expected b evicted after touching a")
	}
}
