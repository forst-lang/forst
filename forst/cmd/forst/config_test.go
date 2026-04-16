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
	if cfg.Compiler.ExportStructFields {
		t.Fatal("expected default exportStructFields false")
	}
}

func TestLoadConfig_compilerExportStructFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ftconfig.json")
	json := `{"compiler": { "exportStructFields": true }}`
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
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

func TestLoadConfig_emptyPath_findsConfigInWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ftconfig.json")
	json := `{"server": { "port": "7777" }}`
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != "7777" {
		t.Fatalf("expected port from cwd config, got %+v", cfg.Server)
	}
}

func TestForstConfig_FindForstFiles_nonexistentRoot_returnsError(t *testing.T) {
	cfg := DefaultConfig()
	root := filepath.Join(t.TempDir(), "nope_subdir_missing")
	_, err := cfg.FindForstFiles(root)
	if err == nil {
		t.Fatal("expected error for missing root")
	}
}

func TestForstConfig_FindForstFiles_includeRestrictsToMatchingPaths(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	other := filepath.Join(root, "other")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(other, 0755); err != nil {
		t.Fatal(err)
	}
	inSrc := filepath.Join(src, "keep.ft")
	outSrc := filepath.Join(other, "skip.ft")
	for _, p := range []string{inSrc, outSrc} {
		if err := os.WriteFile(p, []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := DefaultConfig()
	cfg.Files.Include = []string{"**/src/**/*.ft"}
	cfg.Files.Exclude = nil
	files, err := cfg.FindForstFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "keep.ft" {
		t.Fatalf("want only src/keep.ft, got %v", files)
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
	c.Compiler.ExportStructFields = true
	args := c.ToCompilerArgs()
	if args.LogLevel != "warn" || !args.ReportPhases || !args.ReportMemoryUsage || !args.ExportStructFields {
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

func TestLoadConfigForGenerate_explicitConfigPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ftconfig.json")
	json := `{"server": { "port": "3001" }}`
	if err := os.WriteFile(cfgPath, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}
	ftFile := filepath.Join(dir, "a.ft")
	if err := os.WriteFile(ftFile, []byte("package main\n"), 0644); err != nil {
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
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, "ftconfig.json")
	json := `{"server": { "port": "4002" }}`
	if err := os.WriteFile(cfgPath, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}
	ftFile := filepath.Join(sub, "a.ft")
	if err := os.WriteFile(ftFile, []byte("package main\n"), 0644); err != nil {
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
	if err := os.WriteFile(ftFile, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfigForGenerate("", ftFile, false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != "8080" {
		t.Fatalf("expected default port, got %+v", cfg.Server)
	}
}
