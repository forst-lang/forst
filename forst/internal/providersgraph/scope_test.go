package providersgraph

import (
	"testing"

	"forst/internal/ast"
)

func TestProviderScope_Snapshot_emptyReturnsNil(t *testing.T) {
	t.Parallel()
	if got := NewProviderScope().Snapshot(); got != nil {
		t.Fatalf("empty scope snapshot = %#v, want nil", got)
	}
}

func TestMergeProviderScope_innerShadowsOuterKey(t *testing.T) {
	t.Parallel()
	outer := NewProviderScope()
	outer.Keys["Logger"] = ast.TypeNode{Ident: "Logger"}
	inner := NewProviderScope()
	inner.Keys["Logger"] = ast.TypeNode{Ident: "NopLogger"}
	inner.Shadowed["Logger"] = true

	merged := MergeProviderScope(outer, inner)
	if merged.Keys["Logger"].Ident != ast.TypeIdent("NopLogger") {
		t.Fatalf("inner Logger type = %#v", merged.Keys["Logger"])
	}
}

func TestMergeProviderScope_outerKeyHiddenByInnerShadowed(t *testing.T) {
	t.Parallel()
	outer := NewProviderScope()
	outer.Keys["Clock"] = ast.TypeNode{Ident: "Clock"}
	inner := NewProviderScope()
	inner.Shadowed["Clock"] = true
	inner.Keys["Logger"] = ast.TypeNode{Ident: "Logger"}

	merged := MergeProviderScope(outer, inner)
	if _, ok := merged.Keys["Clock"]; ok {
		t.Fatal("Clock should be omitted when inner marks it shadowed")
	}
	if merged.Keys["Logger"].Ident != ast.TypeIdent("Logger") {
		t.Fatalf("Logger = %#v", merged.Keys["Logger"])
	}
}

func TestMergeScopeStack_emptyReturnsNil(t *testing.T) {
	t.Parallel()
	if got := MergeScopeStack(nil); got != nil {
		t.Fatalf("got %#v", got)
	}
}

func TestMergeScopeStack_outerToInner(t *testing.T) {
	t.Parallel()
	s0 := NewProviderScope()
	s0.Keys["Logger"] = ast.TypeNode{Ident: "Logger"}
	s1 := NewProviderScope()
	s1.Keys["Clock"] = ast.TypeNode{Ident: "Clock"}

	snap := MergeScopeStack([]ProviderScope{s0, s1})
	if len(snap) != 2 {
		t.Fatalf("snapshot len = %d", len(snap))
	}
	if snap["Logger"].Ident != ast.TypeIdent("Logger") || snap["Clock"].Ident != ast.TypeIdent("Clock") {
		t.Fatalf("snapshot = %#v", snap)
	}
}
