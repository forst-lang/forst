package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig_saneDefaults(t *testing.T) {
	c := DefaultConfig()
	if c.Compiler.Target != "go" {
		t.Fatalf("Compiler.Target: %s", c.Compiler.Target)
	}
	if c.Server.Port != "8080" || c.Server.CORS != true {
		t.Fatalf("Server: %+v", c.Server)
	}
	if len(c.Files.Include) == 0 || c.Dev.LogLevel != "info" {
		t.Fatalf("Files/Dev defaults unexpected: %+v %+v", c.Files, c.Dev)
	}
}

func TestLoadConfig_fromJSONFile_mergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ftconfig.json")
	json := `{
  "server": { "port": "3000", "cors": false },
  "dev": { "logLevel": "debug" }
}`
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != "3000" || cfg.Server.CORS != false {
		t.Fatalf("server override: %+v", cfg.Server)
	}
	if cfg.Dev.LogLevel != "debug" {
		t.Fatalf("dev override: %+v", cfg.Dev)
	}
	// Unset in JSON should retain default from DefaultConfig
	if cfg.Compiler.Target != "go" {
		t.Fatalf("expected default compiler target, got %q", cfg.Compiler.Target)
	}
}

func TestLoadConfig_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "failed to load config") {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestLoadConfig_missingFile(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil || !strings.Contains(err.Error(), "failed to read config file") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestFindConfigFile_foundInAncestor(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, "ftconfig.json")
	if err := os.WriteFile(cfgPath, []byte(`{"server":{"port":"9000"},"dev":{"logLevel":"info"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	found, err := FindConfigFile(sub)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(found) != filepath.Clean(cfgPath) {
		t.Fatalf("got %q want %q", found, cfgPath)
	}
}

func TestFindConfigFile_notFound(t *testing.T) {
	dir := t.TempDir()
	found, err := FindConfigFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if found != "" {
		t.Fatalf("expected empty, got %q", found)
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
	args := c.ToCompilerArgs()
	if args.LogLevel != "warn" || !args.ReportPhases || !args.ReportMemoryUsage {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestForstConfig_FindForstFiles_respectsIncludeExclude(t *testing.T) {
	root := t.TempDir()
	includeDir := filepath.Join(root, "src")
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(includeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nm, 0755); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(includeDir, "ok.ft")
	bad := filepath.Join(nm, "skip.ft")
	for _, p := range []string{good, bad} {
		if err := os.WriteFile(p, []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := DefaultConfig()
	files, err := cfg.FindForstFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("want 1 file, got %v", files)
	}
	if filepath.Base(files[0]) != "ok.ft" {
		t.Fatalf("unexpected: %v", files)
	}
}

func TestForstConfig_matchesIncludePatterns_emptyIncludeMatchesAll(t *testing.T) {
	c := DefaultConfig()
	c.Files.Include = nil
	if !c.matchesIncludePatterns("/any/path/foo.ft") {
		t.Fatal("nil include should match")
	}
}

func TestForstConfig_matchesPattern_invalidGlob(t *testing.T) {
	c := DefaultConfig()
	if c.matchesPattern("x", "[") {
		t.Fatal("invalid pattern should not match")
	}
}
