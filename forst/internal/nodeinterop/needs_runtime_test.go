package nodeinterop

import (
	"testing"

	"forst/internal/ast"
)

func TestAnalyzeNeedsNodeRuntime_optedInTSImport(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{
		"legacy/payment.ts": "export async function create() {}",
	})

	nodes := []ast.Node{
		ast.ImportNode{Path: "./legacy/payment", NodeOptIn: true},
	}
	got := AnalyzeNeedsNodeRuntime(nodes, root)

	if !got.Needed {
		t.Fatal("Needed = false, want true")
	}
	if len(got.Modules) != 1 || got.Modules[0] != "legacy/payment.ts" {
		t.Fatalf("Modules = %v, want [legacy/payment.ts]", got.Modules)
	}
}

func TestAnalyzeNeedsNodeRuntime_withoutOptIn(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{
		"legacy/payment.ts": "export {}",
	})

	nodes := []ast.Node{
		ast.ImportNode{Path: "./legacy/payment", NodeOptIn: false},
	}
	got := AnalyzeNeedsNodeRuntime(nodes, root)

	if got.Needed {
		t.Fatal("Needed = true, want false")
	}
	if len(got.Modules) != 0 {
		t.Fatalf("Modules = %v, want nil", got.Modules)
	}
}

func TestAnalyzeNeedsNodeRuntime_bareSpecifier(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{})

	nodes := []ast.Node{
		ast.ImportNode{Path: "effect", NodeOptIn: true},
	}
	got := AnalyzeNeedsNodeRuntime(nodes, root)

	if !got.Needed {
		t.Fatal("Needed = false, want true")
	}
	if len(got.Modules) != 1 || got.Modules[0] != "effect" {
		t.Fatalf("Modules = %v, want [effect]", got.Modules)
	}
}

func TestAnalyzeNeedsNodeRuntime_importGroup(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{
		"legacy/payment.ts": "export {}",
		"legacy/billing.ts": "export {}",
	})

	nodes := []ast.Node{
		ast.ImportGroupNode{Imports: []ast.ImportNode{
			{Path: "./legacy/payment", NodeOptIn: true},
			{Path: "./legacy/billing", NodeOptIn: true},
		}},
	}
	got := AnalyzeNeedsNodeRuntime(nodes, root)

	if !got.Needed {
		t.Fatal("Needed = false, want true")
	}
	want := []string{"legacy/billing.ts", "legacy/payment.ts"}
	if len(got.Modules) != len(want) {
		t.Fatalf("Modules = %v, want %v", got.Modules, want)
	}
	for i, mod := range want {
		if got.Modules[i] != mod {
			t.Fatalf("Modules[%d] = %q, want %q", i, got.Modules[i], mod)
		}
	}
}
