package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/goload"
	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func TestEmitGoSources_writesOverlayAndGoWork_noUserGoModEdit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	consumer := filepath.Join(root, "consumer")
	lib := filepath.Join(root, "acme-lib")
	if err := os.MkdirAll(consumer, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(lib, "go.mod"), "module github.com/acme/lib\n\ngo "+testmod.GoVersion+"\n")
	writeFile(t, filepath.Join(lib, "add.ft"), `package lib

func Add(a Int, b Int) {
	return a + b
}
`)
	consumerGoMod := testmod.GoModContent("testmod") + "\nrequire github.com/acme/lib v0.0.0\n\nreplace github.com/acme/lib => ../acme-lib\n"
	writeFile(t, filepath.Join(consumer, "go.mod"), consumerGoMod)
	writeFile(t, filepath.Join(consumer, "main.ft"), `package main

import "github.com/acme/lib"

func main() {
	_ = lib.Add(1, 2)
}
`)
	goload.ClearLoadCacheForTest()
	outPath := filepath.Join(consumer, "out", "main.go")
	log := logrus.New()
	log.SetOutput(ioDiscard{})
	err := EmitGoSources(Args{
		Command:     "generate",
		FilePath:    filepath.Join(consumer, "main.ft"),
		OutputPath:  outPath,
		PackageRoot: consumer,
		LogLevel:    "error",
	}, log)
	if err != nil {
		t.Fatalf("EmitGoSources: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected generated go: %v", err)
	}
	overlayDir := filepath.Join(consumer, ".forst", "overlay")
	entries, err := os.ReadDir(overlayDir)
	if err != nil {
		t.Fatalf("expected .forst/overlay: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected overlay module directory")
	}
	foundGen := false
	_ = filepath.Walk(overlayDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".gen.go") {
			foundGen = true
		}
		return nil
	})
	if !foundGen {
		t.Fatal("expected *.gen.go under overlay")
	}
	workPath := filepath.Join(consumer, ".forst", "go.work")
	work, err := os.ReadFile(workPath)
	if err != nil {
		t.Fatalf("expected .forst/go.work: %v", err)
	}
	if !strings.Contains(string(work), "replace github.com/acme/lib =>") {
		t.Fatalf("go.work missing overlay replace:\n%s", work)
	}
	after, err := os.ReadFile(filepath.Join(consumer, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != consumerGoMod {
		t.Fatalf("user go.mod was modified:\nbefore:\n%s\nafter:\n%s", consumerGoMod, after)
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
