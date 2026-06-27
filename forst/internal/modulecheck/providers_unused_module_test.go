package modulecheck_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/modulecheck"
)

func TestCheckModuleProviders_unusedWiringKey_intraPackage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module unused_intra\n\ngo 1.23\n")
	writeFile(t, filepath.Join(dir, "demo.ft"), `package main

import "testing"

type Logger = { info(msg String) }
type Clock = { now(): Int }

type NopLogger = {}
type FakeClock = { fixedMs: Int }

func (NopLogger) info(msg String) {}
func (c FakeClock) now(): Int { return c.fixedMs }

func expireToken() {
	use logger: Logger
}

func TestX(t *testing.T) {
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 1 },
	} {
		expireToken()
	}
}
`)
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	tc := result.ForstPackageTypeChecker("main")
	if tc == nil {
		t.Fatal("missing main tc")
	}
	foundClock := false
	for _, w := range tc.Warnings {
		if w.Code == "providers-unused-key" && strings.Contains(w.Msg, "Clock") {
			foundClock = true
		}
		if w.Code == "providers-unused-key" && strings.Contains(w.Msg, "Logger") {
			t.Fatalf("Logger should be required, got warning: %s", w.Msg)
		}
	}
	if !foundClock {
		t.Fatalf("expected unused Clock warning, got: %+v", tc.Warnings)
	}
}

func TestCheckModuleProviders_unusedWiringKey_crossPackage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module unused_cross\n\ngo 1.23\n")
	alphaDir := filepath.Join(dir, "alpha")
	betaDir := filepath.Join(dir, "beta")
	for _, d := range []string{alphaDir, betaDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(alphaDir, "log.ft"), `package alpha

type Logger = { Info(msg String) }

func LogExpiry(id String) {
	use logger: Logger
	logger.Info(id)
}
`)
	writeFile(t, filepath.Join(betaDir, "handle.ft"), `package beta

import "unused_cross/alpha"

type Logger = { Info(msg String) }
type Clock = { now(): Int }

func Handle(id String) {
	alpha.LogExpiry(id)
}
`)
	writeFile(t, filepath.Join(betaDir, "handle_test.ft"), `package beta

import "testing"

type NopLogger = {}

func (NopLogger) Info(msg String) {}

type FakeClock = { fixedMs: Int }

func (c FakeClock) now(): Int { return c.fixedMs }

func TestHandle(t *testing.T) {
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 1 },
	} {
		Handle("tok")
	}
}
`)
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	beta := result.ForstPackageTypeChecker("beta")
	if beta == nil {
		t.Fatal("missing beta tc")
	}
	foundClock := false
	for _, w := range beta.Warnings {
		if w.Code == "providers-unused-key" && strings.Contains(w.Msg, "Clock") {
			foundClock = true
		}
		if w.Code == "providers-unused-key" && strings.Contains(w.Msg, "Logger") {
			t.Fatalf("Logger required via cross-package Handle → alpha.LogExpiry, got warning: %s", w.Msg)
		}
	}
	if !foundClock {
		t.Fatalf("expected unused Clock warning, got: %+v", beta.Warnings)
	}
}
