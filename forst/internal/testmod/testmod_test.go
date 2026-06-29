package testmod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoModContent(t *testing.T) {
	got := GoModContent("example.com/mod")
	want := "module example.com/mod\n\ngo 1.26.0\n"
	if got != want {
		t.Fatalf("GoModContent=%q want %q", got, want)
	}
}

func TestWriteGoMod(t *testing.T) {
	dir := t.TempDir()
	WriteGoMod(t, dir, "writetest")
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "module writetest") {
		t.Fatalf("content=%q", data)
	}
	if !strings.Contains(string(data), GoVersion) {
		t.Fatalf("content=%q", data)
	}
}

func TestWriteGoMod_writeError(t *testing.T) {
	st := &stubFatal{}
	WriteGoMod(st, filepath.Join(t.TempDir(), "missing", "nested"), "mod")
	if !st.failed {
		t.Fatal("expected WriteGoMod to call Fatal on write error")
	}
}

type stubFatal struct {
	failed bool
}

func (s *stubFatal) Helper() {}

func (s *stubFatal) Fatal(_ ...interface{}) {
	s.failed = true
}
