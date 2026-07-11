package nodeinterop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTSImport_relativePathWithoutExtension(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{
		"legacy/payment.ts": "export async function create() {}",
	})
	entryDir := root

	moduleID, absPath, err := ResolveTSImport(entryDir, "./legacy/payment")
	if err != nil {
		t.Fatalf("ResolveTSImport: %v", err)
	}
	if moduleID != "legacy/payment.ts" {
		t.Fatalf("moduleId = %q, want legacy/payment.ts", moduleID)
	}
	wantAbs := filepath.Join(root, "legacy", "payment.ts")
	if absPath != wantAbs {
		t.Fatalf("absPath = %q, want %q", absPath, wantAbs)
	}
}

func TestResolveTSImport_explicitTSExtension(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{
		"legacy/payment.ts": "export {}",
	})

	moduleID, _, err := ResolveTSImport(root, "./legacy/payment.ts")
	if err != nil {
		t.Fatalf("ResolveTSImport: %v", err)
	}
	if moduleID != "legacy/payment.ts" {
		t.Fatalf("moduleId = %q, want legacy/payment.ts", moduleID)
	}
}

func TestResolveTSImport_indexFile(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{
		"legacy/payment/index.ts": "export {}",
	})

	moduleID, _, err := ResolveTSImport(root, "./legacy/payment")
	if err != nil {
		t.Fatalf("ResolveTSImport: %v", err)
	}
	if moduleID != "legacy/payment/index.ts" {
		t.Fatalf("moduleId = %q, want legacy/payment/index.ts", moduleID)
	}
}

func TestResolveTSImport_forstFileShadowsTS(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{
		"legacy/payment.ft":   "package legacy",
		"legacy/payment.ts":   "export {}",
	})

	_, _, err := ResolveTSImport(root, "./legacy/payment")
	if !errors.Is(err, ErrResolvedToNonTS) {
		t.Fatalf("err = %v, want ErrResolvedToNonTS", err)
	}
}

func TestResolveTSImport_outsideBoundary(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{})
	outside := filepath.Join(filepath.Dir(root), "outside.ts")
	if err := os.WriteFile(outside, []byte("export {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })

	_, _, err := ResolveTSImport(root, "../outside")
	if !errors.Is(err, ErrOutsideBoundary) && !errors.Is(err, ErrTSModuleNotFound) {
		t.Fatalf("err = %v, want outside boundary or not found", err)
	}
}

func TestResolveTSImport_rejectsBareSpecifier(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{})

	_, _, err := ResolveTSImport(root, "effect")
	if !errors.Is(err, ErrNotRelativeImport) {
		t.Fatalf("err = %v, want ErrNotRelativeImport", err)
	}
}

func TestResolveTSImport_noBoundary(t *testing.T) {
	dir := t.TempDir()

	_, _, err := ResolveTSImport(dir, "./missing")
	if !errors.Is(err, ErrNoBoundary) {
		t.Fatalf("err = %v, want ErrNoBoundary", err)
	}
}

func TestTSImportHintFile_returnsBasename(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{
		"legacy/payment.ts": "export {}",
	})
	if got := TSImportHintFile(root, "./legacy/payment"); got != "payment.ts" {
		t.Fatalf("TSImportHintFile = %q, want payment.ts", got)
	}
}

func TestTSImportHintFile_emptyWhenForstShadows(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{
		"legacy/payment.ft": "package legacy",
		"legacy/payment.ts": "export {}",
	})
	if got := TSImportHintFile(root, "./legacy/payment"); got != "" {
		t.Fatalf("TSImportHintFile = %q, want empty when .ft shadows", got)
	}
}

func TestTSImportHintFile_emptyForBareSpecifier(t *testing.T) {
	root := writeBoundaryFixture(t, map[string]string{})
	if got := TSImportHintFile(root, "effect"); got != "" {
		t.Fatalf("TSImportHintFile = %q, want empty for bare specifier", got)
	}
}

func writeBoundaryFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ftconfig.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
