package nodeinterop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunIndexer_indexesPaymentFixture(t *testing.T) {
	root := filepath.Join(repoRoot(), "examples", "in", "rfc", "node-interop")
	if _, err := os.Stat(filepath.Join(root, "legacy", "payment.ts")); err != nil {
		t.Skip("node-interop example fixture not present")
	}

	indexes, err := RunIndexer(root, []string{"legacy/payment.ts"})
	if err != nil {
		t.Fatal(err)
	}
	if len(indexes) != 1 {
		t.Fatalf("indexes = %d", len(indexes))
	}
	idx := indexes[0]
	if idx.ModuleID != "legacy/payment.ts" {
		t.Fatalf("moduleId = %q", idx.ModuleID)
	}
	exp, ok := idx.ExportByName("create")
	if !ok {
		t.Fatal("missing create export")
	}
	if exp.Kind != ExportKindFunction {
		t.Fatalf("kind = %q", exp.Kind)
	}
}

func TestRunIndexer_indexesFromTSSource(t *testing.T) {
	root := t.TempDir()
	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tsFile := filepath.Join(legacyDir, "payment.ts")
	if err := os.WriteFile(tsFile, []byte("export function create(): { fresh: string } { return { fresh: 'x' }; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	indexes, err := RunIndexer(root, []string{"legacy/payment.ts"})
	if err != nil {
		t.Fatal(err)
	}
	exp, ok := indexes[0].ExportByName("create")
	if !ok || exp.ReturnType == nil {
		t.Fatal("missing create")
	}
	if _, fresh := exp.ReturnType.Fields["fresh"]; !fresh {
		t.Fatalf("expected fresh field, got %+v", exp.ReturnType.Fields)
	}
}
