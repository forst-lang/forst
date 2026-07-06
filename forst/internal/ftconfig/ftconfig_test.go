package ftconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault_saneDefaults(t *testing.T) {
	c := Default()
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

func TestLoad_fromJSONFile_mergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	json := `{
  "server": { "port": "3000", "cors": false },
  "dev": { "logLevel": "debug" }
}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != "3000" || cfg.Server.CORS != false {
		t.Fatalf("server override: %+v", cfg.Server)
	}
	if cfg.Dev.LogLevel != "debug" {
		t.Fatalf("dev override: %+v", cfg.Dev)
	}
	if cfg.Compiler.Target != "go" {
		t.Fatalf("expected default compiler target, got %q", cfg.Compiler.Target)
	}
	if cfg.Compiler.ExportStructFields {
		t.Fatal("expected default exportStructFields false")
	}
}

func TestLoad_compilerExportStructFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	json := `{"compiler": { "exportStructFields": true }}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Compiler.ExportStructFields {
		t.Fatalf("expected exportStructFields true, got %+v", cfg.Compiler)
	}
}

func TestLoad_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "failed to load config") {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestLoad_missingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil || !strings.Contains(err.Error(), "failed to read config file") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestLoad_emptyPath_findsConfigInWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	json := `{"server": { "port": "7777" }}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
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
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != "7777" {
		t.Fatalf("expected port from cwd config, got %+v", cfg.Server)
	}
}

func TestLoad_negativeMaxRequestSize_clampedToDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	json := `{"server": { "maxRequestSize": -1 }}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.MaxRequestSize != 10*1024*1024 {
		t.Fatalf("expected default max request size, got %d", cfg.Server.MaxRequestSize)
	}
}

func TestLoad_zeroMaxRequestSize_clampedToDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	json := `{"server": { "maxRequestSize": 0 }}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.MaxRequestSize != 10*1024*1024 {
		t.Fatalf("expected default max request size, got %d", cfg.Server.MaxRequestSize)
	}
}

func TestConfig_FindForstFiles_nonexistentRoot_returnsError(t *testing.T) {
	cfg := Default()
	root := filepath.Join(t.TempDir(), "nope_subdir_missing")
	_, err := cfg.FindForstFiles(root)
	if err == nil {
		t.Fatal("expected error for missing root")
	}
}

func TestConfig_FindForstFiles_includeRestrictsToMatchingPaths(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	other := filepath.Join(root, "other")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	inSrc := filepath.Join(src, "keep.ft")
	outSrc := filepath.Join(other, "skip.ft")
	for _, p := range []string{inSrc, outSrc} {
		if err := os.WriteFile(p, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := Default()
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
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, configFileName)
	if err := os.WriteFile(cfgPath, []byte(`{"server":{"port":"9000"},"dev":{"logLevel":"info"}}`), 0o644); err != nil {
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

func TestFindConfigFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, configFileName)
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	found, err := FindConfigFile(filepath.Join(root, "a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	if found != path {
		t.Fatalf("FindConfigFile = %q want %q", found, path)
	}
}

func TestConfig_FindForstFiles_respectsIncludeExclude(t *testing.T) {
	root := t.TempDir()
	includeDir := filepath.Join(root, "src")
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(includeDir, "ok.ft")
	bad := filepath.Join(nm, "skip.ft")
	for _, p := range []string{good, bad} {
		if err := os.WriteFile(p, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := Default()
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

func TestConfig_matchesIncludePatterns_emptyIncludeMatchesAll(t *testing.T) {
	c := Default()
	c.Files.Include = nil
	if !c.matchesIncludePatterns("/any/path/foo.ft") {
		t.Fatal("nil include should match")
	}
}

func TestConfig_matchesPattern_invalidGlob(t *testing.T) {
	c := Default()
	if c.matchesPattern("x", "[") {
		t.Fatal("invalid pattern should not match")
	}
}

func TestConfig_matchesExcludePatterns(t *testing.T) {
	cfg := Default()
	cfg.Files.Exclude = []string{"**/skip/**"}
	if !cfg.matchesExcludePatterns("/proj/skip/hidden.ft") {
		t.Fatal("expected exclude match")
	}
	if cfg.matchesExcludePatterns("/proj/keep/ok.ft") {
		t.Fatal("expected non-excluded path")
	}
}

func TestConfig_matchesIncludePatterns_noMatch(t *testing.T) {
	cfg := Default()
	cfg.Files.Include = []string{"**/only/**"}
	if cfg.matchesIncludePatterns("/proj/other/file.ft") {
		t.Fatal("expected no include match")
	}
}

func TestExportStructFieldsFromDir_defaultFalse(t *testing.T) {
	dir := t.TempDir()
	if ExportStructFieldsFromDir(dir) {
		t.Fatal("expected false without ftconfig.json")
	}
}

func TestExportStructFieldsFromDir_readsNestedConfig(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "pkg", "tests")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"compiler":{"exportStructFields":true}}`
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if !ExportStructFieldsFromDir(sub) {
		t.Fatal("expected true from parent ftconfig.json")
	}
}

func TestLoadFromDir_usesDefaultWhenMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != "8080" {
		t.Fatalf("expected default port, got %+v", cfg.Server)
	}
}

func TestLoadFromDir_loadsAncestorConfig(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte(`{"server":{"port":"4002"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromDir(sub)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != "4002" {
		t.Fatalf("expected parent ftconfig, got %+v", cfg.Server)
	}
}
