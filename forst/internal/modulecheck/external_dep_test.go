package modulecheck

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/goload"
	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func TestCheckModuleProviders_externalForstDep(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	consumer := filepath.Join(root, "consumer")
	lib := filepath.Join(root, "acme-lib")
	mustMkdir(t, consumer)
	mustMkdir(t, lib)
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
	result, err := CheckModuleProviders(log, Options{ModuleRoot: consumer})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
	if !result.IsExternalImport("github.com/acme/lib") {
		t.Fatal("expected github.com/acme/lib in ExternalImports")
	}
	tc := result.TypeCheckerForImportPath("github.com/acme/lib")
	if tc == nil {
		t.Fatal("missing typechecker for external import")
	}
	if _, ok := tc.Functions["Add"]; !ok {
		t.Fatal("expected Add in external Functions")
	}
	mainTC := result.PerPackage["main"]
	if mainTC == nil {
		t.Fatal("missing main typechecker")
	}
}

func TestCheckModuleProviders_unwiredDepExportDoesNotFailConsumer(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	consumer := filepath.Join(root, "consumer")
	lib := filepath.Join(root, "acme-lib")
	mustMkdir(t, consumer)
	mustMkdir(t, lib)
	writeFile(t, filepath.Join(lib, "go.mod"), "module github.com/acme/lib\n\ngo "+testmod.GoVersion+"\n")
	writeFile(t, filepath.Join(lib, "lib.ft"), `package lib

type Logger = {Log(msg String)}

func Admin(msg String) {
	use logger: Logger
	logger.Log(msg)
}

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
	if _, err := CheckModuleProviders(log, Options{ModuleRoot: consumer}); err != nil {
		t.Fatalf("consumer should not fail for unused Admin use: %v", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
