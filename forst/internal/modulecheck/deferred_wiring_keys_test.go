package modulecheck_test

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/modulecheck"
)

func TestCheckModuleProviders_crossPackageWiringAtHostEntry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module wiring_sibling\n\ngo 1.23\n")
	libDir := filepath.Join(dir, "lib")
	svcDir := filepath.Join(dir, "svc")
	for _, d := range []string{libDir, svcDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(libDir, "log.ft"), `package lib

type Logger = { Info(msg String) }

func Log(msg String) {
	use logger: Logger
	logger.Info(msg)
}
`)
	writeFile(t, filepath.Join(svcDir, "run.ft"), `package svc

import "wiring_sibling/lib"

type Logger = { Info(msg String) }

type SvcLogger = {}

func (SvcLogger) Info(msg String) {}

func Run() {
	logger := SvcLogger{}
	with { Logger: &logger } {
		lib.Log("hi")
	}
}
`)
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	svc := result.ForstPackageTypeChecker("svc")
	if svc == nil {
		t.Fatal("missing svc tc")
	}
	if len(svc.FunctionProviders["Run"]) != 0 {
		t.Fatalf("Run should be runnable after wiring, providers = %v", svc.FunctionProviders["Run"])
	}
}
