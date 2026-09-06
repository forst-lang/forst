package depoverlay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/codegen/layout"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func TestEmit_writesGenGoUnderOverlay(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	consumer := filepath.Join(root, "consumer")
	lib := filepath.Join(root, "acme-lib")
	mustMkdirAll(t, consumer)
	mustMkdirAll(t, lib)
	writeFile(t, filepath.Join(lib, "go.mod"), "module github.com/acme/lib\n\ngo "+testmod.GoVersion+"\n")
	writeFile(t, filepath.Join(lib, "add.ft"), `package lib

func Add(a Int, b Int) {
	return a + b
}
`)
	writeFile(t, filepath.Join(consumer, "go.mod"), testmod.GoModContent("testmod")+"\nrequire github.com/acme/lib v0.0.0\n\nreplace github.com/acme/lib => ../acme-lib\n")
	writeFile(t, filepath.Join(consumer, "main.ft"), `package main

import "github.com/acme/lib"

func main() {
	_ = lib.Add(1, 2)
}
`)
	goload.ClearLoadCacheForTest()
	log := logrus.New()
	log.SetOutput(ioDiscard{})
	result, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{ModuleRoot: consumer})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	replaces, err := Emit(log, consumer, result, false)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if len(replaces) != 1 {
		t.Fatalf("replaces=%v want 1", replaces)
	}
	if replaces[0].ImportPath != "github.com/acme/lib" {
		t.Fatalf("replace path=%q", replaces[0].ImportPath)
	}
	overlayRoot := replaces[0].Dir
	if !strings.HasPrefix(overlayRoot, filepath.Join(consumer, ".forst", "overlay")) {
		t.Fatalf("overlay dir %q not under .forst/overlay", overlayRoot)
	}
	entries, err := os.ReadDir(overlayRoot)
	if err != nil {
		t.Fatal(err)
	}
	var foundGen bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), layout.SuffixGen) {
			foundGen = true
			break
		}
	}
	if !foundGen {
		t.Fatalf("expected *.gen.go in overlay %s, entries=%v", overlayRoot, entries)
	}
	if _, err := os.Stat(filepath.Join(lib, "lib.gen.go")); !os.IsNotExist(err) {
		t.Fatal("must not write gen.go into source module dir")
	}
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
