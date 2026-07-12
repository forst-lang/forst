package layout

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLayout_runSession_pathsUnderDotForst(t *testing.T) {
	r := NewRoot("/app/forst")
	p := r.RunSession("sess1")
	for _, path := range []string{p.Dir, p.GoMod, p.HostMain, p.InvokeServer, p.NodeRuntime} {
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
