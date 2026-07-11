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
	if c.Node.Enabled || c.Node.RuntimeEnabled {
		t.Fatalf("Node defaults should be disabled: %+v", c.Node)
	}
	if c.Node.ImportPolicy != "explicit" {
		t.Fatalf("Node.ImportPolicy: %q", c.Node.ImportPolicy)
	}
	if c.Node.Binary != "node" {
		t.Fatalf("Node.Binary: %q", c.Node.Binary)
	}
	if c.Node.Bootstrap != "node_modules/@forst/node-runtime/dist/bootstrap.js" {
		t.Fatalf("Node.Bootstrap: %q", c.Node.Bootstrap)
	}
	if c.Node.Loader != "tsx" {
		t.Fatalf("Node.Loader: %q", c.Node.Loader)
	}
	if c.Node.RPC.MaxMessageBytes != 16<<20 {
		t.Fatalf("Node.RPC.MaxMessageBytes: %d", c.Node.RPC.MaxMessageBytes)
	}
	if c.Node.RPC.CallTimeoutSeconds != 120 {
		t.Fatalf("Node.RPC.CallTimeoutSeconds: %d", c.Node.RPC.CallTimeoutSeconds)
	}
	if c.Node.HostSocket != ".forst/node.sock" {
		t.Fatalf("Node.HostSocket: %q", c.Node.HostSocket)
	}
	if c.Node.HostReadyTimeoutSeconds != 120 {
		t.Fatalf("Node.HostReadyTimeoutSeconds: %d", c.Node.HostReadyTimeoutSeconds)
	}
}

func TestLoad_nodeStanza_parsesAndMergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	json := `{
  "node": {
    "enabled": true,
    "importPolicy": "implicit",
    "runtimeEnabled": true,
    "binary": "/usr/local/bin/node",
    "bootstrap": "node_modules/@forst/node-runtime/dist/bootstrap.js",
    "loader": "tsx",
    "rpc": {
      "maxMessageBytes": 1048576,
      "callTimeoutSeconds": 30
    }
  }
}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Node.Enabled || !cfg.Node.RuntimeEnabled {
		t.Fatalf("node overrides: %+v", cfg.Node)
	}
	if cfg.Node.ImportPolicy != "implicit" {
		t.Fatalf("importPolicy: %q", cfg.Node.ImportPolicy)
	}
	if cfg.Node.Binary != "/usr/local/bin/node" {
		t.Fatalf("binary: %q", cfg.Node.Binary)
	}
	if cfg.Node.RPC.MaxMessageBytes != 1048576 {
		t.Fatalf("rpc.maxMessageBytes: %d", cfg.Node.RPC.MaxMessageBytes)
	}
	if cfg.Node.RPC.CallTimeoutSeconds != 30 {
		t.Fatalf("rpc.callTimeoutSeconds: %d", cfg.Node.RPC.CallTimeoutSeconds)
	}
	if cfg.Compiler.Target != "go" {
		t.Fatalf("expected default compiler target, got %q", cfg.Compiler.Target)
	}
}

func TestLoad_nodeRPCZeroValues_normalizedToDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	json := `{"node": {"rpc": {"maxMessageBytes": 0, "callTimeoutSeconds": 0}}}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Node.RPC.MaxMessageBytes != 16<<20 {
		t.Fatalf("maxMessageBytes: %d", cfg.Node.RPC.MaxMessageBytes)
	}
	if cfg.Node.RPC.CallTimeoutSeconds != 120 {
		t.Fatalf("callTimeoutSeconds: %d", cfg.Node.RPC.CallTimeoutSeconds)
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

func TestBoundaryRootFromDir_foundInAncestor(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, configFileName)
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := BoundaryRootFromDir(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("BoundaryRootFromDir = %q want %q", got, root)
	}
}

func TestBoundaryRootFromDir_notFound(t *testing.T) {
	dir := t.TempDir()
	_, err := BoundaryRootFromDir(dir)
	if err == nil {
		t.Fatal("expected error when ftconfig.json missing")
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

func TestImportPolicyFromDir_defaultExplicit(t *testing.T) {
	dir := t.TempDir()
	if got := ImportPolicyFromDir(dir); got != "explicit" {
		t.Fatalf("ImportPolicyFromDir = %q, want explicit", got)
	}
}

func TestImportPolicyFromDir_readsNestedConfig(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "pkg", "tests")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"node":{"importPolicy":"implicit"}}`
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ImportPolicyFromDir(sub); got != "implicit" {
		t.Fatalf("ImportPolicyFromDir = %q, want implicit", got)
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

func TestLoad_hostModeRequiresArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	json := `{
  "node": {
    "hostMode": true,
    "args": []
  }
}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "hostMode requires non-empty node.args") {
		t.Fatalf("err = %v", err)
	}
}

func TestNodeConfig_EffectiveHostAutoRegister(t *testing.T) {
	falseVal := false
	cfg := Default()
	if cfg.Node.EffectiveHostAutoRegister() {
		t.Fatal("expected false when hostMode unset and hostAutoRegister unset")
	}
	cfg.Node.HostMode = true
	if !cfg.Node.EffectiveHostAutoRegister() {
		t.Fatal("expected true by default when hostMode enabled")
	}
	cfg.Node.HostAutoRegister = &falseVal
	if cfg.Node.EffectiveHostAutoRegister() {
		t.Fatal("expected false when hostAutoRegister explicitly false")
	}
}

func TestEffectiveInvokeHost_embeddedAlwaysLocalhost(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{"empty", ""},
		{"localhost", "localhost"},
		{"zero", "0.0.0.0"},
		{"public", "192.168.1.1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ServerConfig{Embedded: true, Host: tc.host}.EffectiveInvokeHost()
			if got != "127.0.0.1" {
				t.Fatalf("EffectiveInvokeHost() = %q", got)
			}
		})
	}
}

func TestEffectiveInvokeHost_nonEmbeddedUsesConfiguredHost(t *testing.T) {
	got := ServerConfig{Embedded: false, Host: "0.0.0.0"}.EffectiveInvokeHost()
	if got != "0.0.0.0" {
		t.Fatalf("EffectiveInvokeHost() = %q", got)
	}
	got = ServerConfig{Embedded: false, Host: ""}.EffectiveInvokeHost()
	if got != "localhost" {
		t.Fatalf("EffectiveInvokeHost() = %q", got)
	}
}

func TestEffectiveDevListenHost_defaultsLoopback(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{"empty", ""},
		{"localhost", "localhost"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ServerConfig{Host: tc.host}.EffectiveDevListenHost()
			if got != "127.0.0.1" {
				t.Fatalf("EffectiveDevListenHost() = %q", got)
			}
		})
	}
}

func TestEffectiveDevListenHost_preservesExplicitHost(t *testing.T) {
	got := ServerConfig{Host: "127.0.0.1"}.EffectiveDevListenHost()
	if got != "127.0.0.1" {
		t.Fatalf("EffectiveDevListenHost() = %q", got)
	}
	got = ServerConfig{Host: "0.0.0.0"}.EffectiveDevListenHost()
	if got != "0.0.0.0" {
		t.Fatalf("EffectiveDevListenHost() = %q", got)
	}
}
