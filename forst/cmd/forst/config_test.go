package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"
)

func TestLoadConfig_compilerExportStructFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ftconfig.json")
	json := `{"compiler": { "exportStructFields": true }}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Compiler.ExportStructFields {
		t.Fatalf("expected exportStructFields true, got %+v", cfg.Compiler)
	}
	args := cfg.ToCompilerArgs()
	if !args.ExportStructFields {
		t.Fatalf("ToCompilerArgs: %+v", args)
	}
}

func TestForstConfig_Validate(t *testing.T) {
	c := DefaultConfig()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}

	c2 := DefaultConfig()
	c2.Server.Port = ""
	if err := c2.Validate(); err == nil || !strings.Contains(err.Error(), "port") {
		t.Fatalf("expected port error, got %v", err)
	}

	c3 := DefaultConfig()
	c3.Dev.LogLevel = "invalid"
	if err := c3.Validate(); err == nil || !strings.Contains(err.Error(), "log level") {
		t.Fatalf("expected log level error, got %v", err)
	}
}

func TestForstConfig_ToCompilerArgs(t *testing.T) {
	c := DefaultConfig()
	c.Dev.LogLevel = "warn"
	c.Compiler.ReportPhases = true
	c.Compiler.ReportMemoryUsage = true
	c.Compiler.ExportStructFields = true
	args := c.ToCompilerArgs()
	if args.LogLevel != "warn" || !args.ReportPhases || !args.ReportMemoryUsage || !args.ExportStructFields {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestLoadConfigForGenerate_explicitConfigPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ftconfig.json")
	json := `{"server": { "port": "3001" }}`
	if err := os.WriteFile(cfgPath, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	ftFile := filepath.Join(dir, "a.ft")
	if err := os.WriteFile(ftFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfigForGenerate(cfgPath, ftFile, false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != "3001" {
		t.Fatalf("server: %+v", cfg.Server)
	}
}

func TestLoadConfigForGenerate_findsConfigWalkingUpFromFileDir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, "ftconfig.json")
	json := `{"server": { "port": "4002" }}`
	if err := os.WriteFile(cfgPath, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	ftFile := filepath.Join(sub, "a.ft")
	if err := os.WriteFile(ftFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfigForGenerate("", ftFile, false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != "4002" {
		t.Fatalf("expected parent ftconfig, got %+v", cfg.Server)
	}
}

func TestLoadConfigForGenerate_usesDefaultWhenNoConfigFile(t *testing.T) {
	dir := t.TempDir()
	ftFile := filepath.Join(dir, "only.ft")
	if err := os.WriteFile(ftFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfigForGenerate("", ftFile, false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != ftconfig.DefaultDevExecutorPort {
		t.Fatalf("expected default port, got %+v", cfg.Server)
	}
}
