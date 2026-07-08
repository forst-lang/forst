package goload

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func moduleRootFromWDForDocTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

func TestFormatDocMarkdown_nonempty(t *testing.T) {
	t.Parallel()
	got := FormatDocMarkdown("Println formats using the default formats for its operands.")
	if got == "" {
		t.Fatal("expected non-empty markdown")
	}
	if !strings.Contains(got, "Println") {
		t.Fatalf("markdown should preserve text: %q", got)
	}
}

func TestDocForFunc_fmtPrintln(t *testing.T) {
	t.Parallel()
	moduleRoot := moduleRootFromWDForDocTest(t)
	docPkg, _, err := LoadDocPackage(moduleRoot, "fmt")
	if err != nil {
		t.Fatalf("LoadDocPackage(fmt): %v", err)
	}
	raw := DocForFunc(docPkg, "Println")
	if raw == "" {
		t.Fatal("expected godoc for fmt.Println")
	}
	if !strings.Contains(raw, "Println") {
		t.Fatalf("doc should mention Println: %q", raw)
	}
}

func TestPkgGoDevURL(t *testing.T) {
	t.Parallel()
	if got := PkgGoDevURL("fmt", "Println"); got != "https://pkg.go.dev/fmt#Println" {
		t.Fatalf("unexpected URL: %q", got)
	}
	if got := PkgGoDevURL("example.com/pkg", "Foo"); got != "https://pkg.go.dev/example.com/pkg#Foo" {
		t.Fatalf("unexpected URL: %q", got)
	}
}

func TestIsStdlibImportPath(t *testing.T) {
	t.Parallel()
	if !IsStdlibImportPath("fmt") {
		t.Fatal("fmt should be stdlib")
	}
	if IsStdlibImportPath("example.com/pkg") {
		t.Fatal("example.com/pkg should not be stdlib")
	}
}
