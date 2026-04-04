package goload

import (
	"os"
	"path/filepath"
	"testing"
)

func moduleRootFromWD(t *testing.T) string {
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

func TestLoadByPkgPath_fmt(t *testing.T) {
	dir := moduleRootFromWD(t)
	m, err := LoadByPkgPath(dir, []string{"fmt"})
	if err != nil {
		t.Fatal(err)
	}
	p := m["fmt"]
	if p == nil || p.Types == nil {
		t.Fatalf("fmt package missing: %#v", m)
	}
	if p.Types.Scope().Lookup("Println") == nil {
		t.Fatal("fmt.Println not in scope")
	}
}

func TestImportPathFromForst(t *testing.T) {
	if got := ImportPathFromForst(`"net/http"`); got != "net/http" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadByPkgPath_emptyPaths(t *testing.T) {
	t.Parallel()
	m, err := LoadByPkgPath(t.TempDir(), nil)
	if err != nil || m != nil {
		t.Fatalf("empty import list: m=%v err=%v", m, err)
	}
}
