package layout

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLayout_runSession_pathsUnderDotForst(t *testing.T) {
	r := NewRoot("/app/forst")
	p := r.RunSession("sess1")
	for _, path := range []string{p.Dir, p.GoMod, p.HostMain, p.InvokeServer, p.BridgeRuntime} {
		if !strings.HasPrefix(path, filepath.Join("/app/forst", ".forst", "run")) {
			t.Fatalf("expected under .forst/run, got %s", path)
		}
	}
}

func TestLayout_testRun_noSourceTreeWrites(t *testing.T) {
	r := NewRoot("/app/forst")
	p := r.TestRun("run1", "auth")
	if strings.Contains(p.TestFile, "/app/forst/auth/") && !strings.Contains(p.TestFile, ".forst") {
		t.Fatalf("test file must not be beside sources: %s", p.TestFile)
	}
	if !strings.Contains(p.RunDir, ".forst/gen/test/run1") {
		t.Fatalf("expected session dir under .forst/gen/test, got %s", p.RunDir)
	}
}

func TestLayout_libShim_replacesZPrefix(t *testing.T) {
	r := NewRoot("/app")
	path := r.LibShim("rid", "auth")
	if strings.Contains(filepath.Base(path), "z_") {
		t.Fatalf("must not use z_ prefix: %s", path)
	}
	if filepath.Base(path) != FileLibShim {
		t.Fatalf("want %s, got %s", FileLibShim, filepath.Base(path))
	}
}

func TestClientDir_anchoredAtBoundaryRoot(t *testing.T) {
	r := NewRoot("/app/forst")
	got := r.ClientDir()
	want := filepath.Join("/app/forst", ".forst", "client")
	if got != want {
		t.Fatalf("ClientDir() = %q, want %q", got, want)
	}
}

func TestLayout_noPathHelperCollidesWithReservedEntry(t *testing.T) {
	r := NewRoot("/app/forst")
	dotForst := filepath.Join("/app/forst", ".forst")
	reserved := map[string]bool{}
	for _, name := range reservedDotForstEntries {
		reserved[name] = true
	}

	paths := []string{
		r.RunSession("sess1").Dir,
		r.PackageGo("sess1", "mypkg", "mypkg"),
		r.ExecModule(1, "mypkg", "DoThing").Dir,
		r.TestRun("run1", "mypkg").RunDir,
		r.LibShim("run1", "mypkg"),
	}
	for _, path := range paths {
		rel, err := filepath.Rel(dotForst, path)
		if err != nil {
			t.Fatalf("Rel(%q, %q): %v", dotForst, path, err)
		}
		if rel == "." || strings.HasPrefix(rel, "..") {
			t.Fatalf("path %q is not under .forst", path)
		}
		first := strings.Split(rel, string(filepath.Separator))[0]
		if !reserved[first] {
			t.Fatalf("first segment under .forst must be reserved compiler name, got %q from %q", first, path)
		}
		if first == "mypkg" {
			t.Fatalf("user package name must not be a direct child of .forst: %q", path)
		}
	}
}

func TestLayout_forstPackageNamedClientDoesNotEscapeRunDir(t *testing.T) {
	r := NewRoot("/app/forst")
	const hostile = "client"

	pkgGo := r.PackageGo("sess1", hostile, hostile)
	execDir := r.ExecModule(1, hostile, "Fn").Dir
	testFile := r.TestRun("run1", hostile).TestFile
	libShim := r.LibShim("run1", hostile)

	for _, path := range []string{pkgGo, execDir, testFile, libShim} {
		if path == r.ClientDir() {
			t.Fatalf("hostile package path must not equal ClientDir: %q", path)
		}
		rel, err := filepath.Rel(r.ClientDir(), path)
		if err != nil {
			t.Fatalf("Rel: %v", err)
		}
		// Paths under ClientDir would start without ".." after Rel; nest under run/exec/gen instead.
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			t.Fatalf("hostile package path must not live under ClientDir: %q (rel %q)", path, rel)
		}
	}

	if !strings.HasPrefix(pkgGo, filepath.Join(r.dotForst(), "run", "sess1")) {
		t.Fatalf("PackageGo with package %q must stay under run session: %q", hostile, pkgGo)
	}
	if !strings.HasPrefix(execDir, filepath.Join(r.dotForst(), "exec")) {
		t.Fatalf("ExecModule with package %q must stay under exec: %q", hostile, execDir)
	}
}
