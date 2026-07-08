package modulecheck_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/modulecheck"
	"forst/internal/testmod"
)

func TestCheckModuleProviders_unusedWiringKey_intraPackage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), testmod.GoModContent("unused_intra"))
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
	writeFile(t, filepath.Join(dir, "go.mod"), testmod.GoModContent("unused_cross"))
	authDir := filepath.Join(dir, "auth")
	apiDir := filepath.Join(dir, "api")
	for _, d := range []string{authDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(authDir, "log.ft"), `package auth

type Logger = { Info(msg String) }

func LogEvent(id String) {
	use logger: Logger
	logger.Info(id)
}
`)
	writeFile(t, filepath.Join(apiDir, "handle.ft"), `package api

import "unused_cross/auth"

type Logger = { Info(msg String) }
type Clock = { now(): Int }

func HandleRequest(id String) {
	auth.LogEvent(id)
}
`)
	writeFile(t, filepath.Join(apiDir, "handle_test.ft"), `package api

import "testing"

type NopLogger = {}

func (NopLogger) Info(msg String) {}

type FakeClock = { fixedMs: Int }

func (c FakeClock) now(): Int { return c.fixedMs }

func TestHandleRequest(t *testing.T) {
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 1 },
	} {
		HandleRequest("tok")
	}
}
`)
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	apiTC := result.ForstPackageTypeChecker("api")
	if apiTC == nil {
		t.Fatal("missing api tc")
	}
	foundClock := false
	for _, w := range apiTC.Warnings {
		if w.Code == "providers-unused-key" && strings.Contains(w.Msg, "Clock") {
			foundClock = true
		}
		if w.Code == "providers-unused-key" && strings.Contains(w.Msg, "Logger") {
			t.Fatalf("Logger required via cross-package HandleRequest → auth.LogEvent, got warning: %s", w.Msg)
		}
	}
	if !foundClock {
		t.Fatalf("expected unused Clock warning, got: %+v", apiTC.Warnings)
	}
}
