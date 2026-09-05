package unixpath

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureLength_shortPathUnchanged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix path shortening is a no-op on Windows")
	}
	short := filepath.Join("/tmp", "forst.sock")
	if got := EnsureLength(short, "forst-inv-"); got != short {
		t.Fatalf("EnsureLength(%q) = %q", short, got)
	}
}

func TestEnsureLength_longPathUsesTmpPrefix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix path shortening is a no-op on Windows")
	}
	long := strings.Repeat("a", MaxLen+20)
	got := EnsureLength(long, "forst-inv-")
	if len(got) > MaxLen {
		t.Fatalf("len(%q) = %d, want <= %d", got, len(got), MaxLen)
	}
	if !strings.HasPrefix(filepath.Base(got), "forst-inv-") {
		t.Fatalf("base = %q, want forst-inv- prefix", filepath.Base(got))
	}
	if EnsureLength(long, "forst-inv-") != got {
		t.Fatal("hash truncation must be stable")
	}
	other := EnsureLength(long+"x", "forst-inv-")
	if other == got {
		t.Fatal("different inputs must not collide")
	}
}
