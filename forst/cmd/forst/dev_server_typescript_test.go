package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/discovery"

	"github.com/sirupsen/logrus"
)

func TestNewTypeScriptGenerator_GenerateTypesForFunctions_emptyDiscoveryReturnsHeaderOnly(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	tg := NewTypeScriptGenerator(log)
	out, err := tg.GenerateTypesForFunctions(nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Auto-generated types for Forst client") {
		t.Fatalf("expected header in types output, got %q", out)
	}
}

func TestTypeScriptGenerator_GenerateTypesForFunctions_nonEmptyDiscovery_mergesPerFileOutputs(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "echo.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	tg := NewTypeScriptGenerator(log)
	functions := map[string]map[string]discovery.FunctionInfo{
		"main": {
			"Echo": {
				Package:  "main",
				Name:     "Echo",
				FilePath: ft,
			},
		},
	}

	out, err := tg.GenerateTypesForFunctions(functions, dir)
	if err != nil {
		t.Fatalf("GenerateTypesForFunctions: %v", err)
	}
	if !strings.Contains(out, "EchoRequest") || !strings.Contains(out, "Echo") {
		t.Fatalf("expected Echo/EchoRequest in merged TS, got:\n%s", out)
	}
}

func TestTypeScriptGenerator_GenerateTypesForFunctions_allFilesFail_returnsHeaderOnly(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	tg := NewTypeScriptGenerator(log)
	missing := filepath.Join(t.TempDir(), "missing.ft")
	functions := map[string]map[string]discovery.FunctionInfo{
		"main": {
			"Echo": {
				Package:  "main",
				Name:     "Echo",
				FilePath: missing,
			},
		},
	}

	out, err := tg.GenerateTypesForFunctions(functions, t.TempDir())
	if err != nil {
		t.Fatalf("GenerateTypesForFunctions: %v", err)
	}
	if !strings.Contains(out, "Auto-generated types for Forst client") {
		t.Fatalf("expected header when no outputs merge, got %q", out)
	}
	if strings.Contains(out, "EchoRequest") {
		t.Fatalf("did not expect types from failed transform, got %q", out)
	}
}

func TestTypeScriptGenerator_generateTypesForFile_readsParsesAndTransforms(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "echo.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	tg := NewTypeScriptGenerator(log)
	types, funcs, pkg, err := tg.generateTypesForFile(ft)
	if err != nil {
		t.Fatal(err)
	}
	if pkg != "main" {
		t.Fatalf("package: %q", pkg)
	}
	if len(funcs) == 0 || len(types) == 0 {
		t.Fatalf("expected types and functions, got types=%d funcs=%d", len(types), len(funcs))
	}
}

func TestTypeScriptGenerator_generateTypesForFile_missingFile(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	tg := NewTypeScriptGenerator(log)
	_, _, _, err := tg.generateTypesForFile(filepath.Join(t.TempDir(), "nope.ft"))
	if err == nil || !strings.Contains(err.Error(), "failed to read") {
		t.Fatalf("expected read error, got %v", err)
	}
}
