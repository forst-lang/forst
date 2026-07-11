package typechecker

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testutil"
)

func TestNodeHoverMarkdown_createExport(t *testing.T) {
	root := t.TempDir()
	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tsFile := filepath.Join(legacyDir, "payment.ts")
	if err := os.WriteFile(tsFile, []byte(`export function create(amount: number, currency: string) {
  return { id: "pay_1", amount, currency };
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	src := `package main
import node "./legacy/payment"
func main() {
  payment.create(1.0, "USD")
}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})

	md, ok := tc.NodeHoverMarkdown("payment", "create")
	if !ok {
		t.Fatal("expected hover for payment.create")
	}
	for _, want := range []string{"(alias)", "function create", "amount: number", "currency: string", "id: string"} {
		if !contains(md, want) {
			t.Fatalf("hover missing %q:\n%s", want, md)
		}
	}
	if contains(md, "module ") {
		t.Fatalf("export hover should not include module path:\n%s", md)
	}
}

func TestNodeHoverMarkdown_moduleLocal(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)
	src := `package main
import node "./legacy/payment"
func main() {}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	md, ok := tc.NodeModuleHoverMarkdown("payment")
	if !ok || !contains(md, "module payment") || contains(md, "legacy/payment") {
		t.Fatalf("module hover: ok=%v md=%q", ok, md)
	}
}

func TestNodeImportPathHoverMarkdown_absolutePath(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)
	src := `package main
import node "./legacy/payment"
func main() {}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	md, ok := tc.NodeImportPathHoverMarkdown("./legacy/payment")
	wantPath := filepath.ToSlash(filepath.Join(root, "legacy", "payment"))
	if !ok || !contains(md, fmt.Sprintf(`module "%s"`, wantPath)) || !contains(md, "```typescript") {
		t.Fatalf("path hover: ok=%v md=%q want %q", ok, md, wantPath)
	}
}

func TestNodeTypecheck_indexesFromTSSource(t *testing.T) {
	root := t.TempDir()
	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tsFile := filepath.Join(legacyDir, "payment.ts")
	if err := os.WriteFile(tsFile, []byte("export function create() { return { id: 'x' }; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := `package main
import node "./legacy/payment"
func main() {
  r := payment.create()
}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	mod, ok := tc.nodeModuleForLocal("payment")
	if !ok || mod.Index == nil {
		t.Fatal("missing node module")
	}
	exp, ok := mod.Index.ExportByName("create")
	if !ok || exp.ReturnType == nil {
		t.Fatal("missing create export")
	}
	if _, hasID := exp.ReturnType.Fields["id"]; !hasID {
		t.Fatalf("expected id field, got %+v", exp.ReturnType.Fields)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
