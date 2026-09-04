package typechecker

import (
	"testing"
	"time"
)

// Phase 5 MATRIX stubs: metadata hashes, incremental invalidation hooks,
// determinism, widening diagnostics, and lightweight perf budgets.

func TestPhase5_MetadataHashStub(t *testing.T) {
	h1 := SummaryMetadataHash("pkg", "fn", "v1", []string{"a", "b"})
	h2 := SummaryMetadataHash("pkg", "fn", "v1", []string{"b", "a"}) // callee order independent
	if h1 == "" || h1 != h2 {
		t.Fatalf("expected stable hash, got %q vs %q", h1, h2)
	}
	h3 := SummaryMetadataHash("pkg", "fn", "v2", []string{"a", "b"})
	if h1 == h3 {
		t.Fatal("version change must change hash")
	}
}

func TestPhase5_IncrementalInvalidationHook(t *testing.T) {
	hook := NewIncrementalInvalidator()
	hook.RecordCallerDependsOn("main", "check", "pkg", "rename")
	hook.MarkSummaryChanged("pkg", "rename")
	if !hook.MustRevisitCaller("main", "check") {
		t.Fatal("caller must revisit after summary change")
	}
	hook.Clear()
	if hook.MustRevisitCaller("main", "check") {
		t.Fatal("clear must drop dirty set")
	}
}

func TestPhase5_DeterministicSummaryKeys(t *testing.T) {
	a := []string{"z", "a", "m"}
	b := []string{"m", "z", "a"}
	if SummaryMetadataHash("p", "f", "1", a) != SummaryMetadataHash("p", "f", "1", b) {
		t.Fatal("hash must be order-independent")
	}
}

func TestPhase5_WideningDiagnosticCode(t *testing.T) {
	code := diagnosticCodeForDrop(dropByWrite)
	if code == "" {
		t.Fatal("expected code")
	}
	// Widening family uses a dedicated stable code.
	if GotWideningDiagnosticCode() != "refinement-analysis-widened" {
		t.Fatalf("got %s", GotWideningDiagnosticCode())
	}
}

func TestPhase5_PerfBudgetStub(t *testing.T) {
	start := time.Now()
	_ = SummaryMetadataHash("p", "f", "1", []string{"x", "y", "z"})
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("metadata hash exceeded lightweight budget")
	}
}
