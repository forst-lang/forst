package nodert

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveNodeBinary_envOverridesFtconfig(t *testing.T) {
	root := t.TempDir()
	custom := filepath.Join(root, "custom-node")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envNodeBinary, custom)
	got, err := ResolveNodeBinary(root, "from-config")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(custom)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want env override", got)
	}
}

func TestResolveNodeBinary_usesConfiguredWhenEnvUnset(t *testing.T) {
	root := t.TempDir()
	nodePath := filepath.Join(root, "node-bin")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envNodeBinary, "")
	got, err := ResolveNodeBinary(root, nodePath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(nodePath)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestResolveBootstrapPath_relativeToBoundaryRoot(t *testing.T) {
	root := t.TempDir()
	bootstrapDir := filepath.Join(root, "node_modules", "@forst", "node-runtime", "dist")
	if err := os.MkdirAll(bootstrapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bootstrapFile := filepath.Join(bootstrapDir, "bootstrap.js")
	if err := os.WriteFile(bootstrapFile, []byte("// bootstrap"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envNodeBootstrap, "")
	got, err := ResolveBootstrapPath(root, "node_modules/@forst/node-runtime/dist/bootstrap.js")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(bootstrapFile)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveBootstrapPath_envRelativeFallsBackToMonorepo(t *testing.T) {
	repo := t.TempDir()
	bootstrapFile := filepath.Join(repo, "packages", "node-runtime", "dist", "bootstrap.js")
	if err := os.MkdirAll(filepath.Dir(bootstrapFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bootstrapFile, []byte("// bootstrap"), 0o644); err != nil {
		t.Fatal(err)
	}
	boundary := filepath.Join(repo, "examples", "sync")
	if err := os.MkdirAll(boundary, 0o755); err != nil {
		t.Fatal(err)
	}
	sandbox := filepath.Join(boundary, ".forst", "run", "forst-1")
	if err := os.MkdirAll(sandbox, 0o755); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	if err := os.Chdir(sandbox); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envNodeBootstrap, "../packages/node-runtime/dist/bootstrap.js")

	got, err := ResolveBootstrapPath(boundary, "")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(bootstrapFile)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildNodeChildEnv_setsBoundaryProtocolAndExclude(t *testing.T) {
	env := buildNodeChildEnv(ProcessOptions{
		BoundaryRoot: "/tmp/project",
		FilesExclude: []string{"**/node_modules/**", "**/.git/**"},
	})

	assertEnvVar(t, env, "FORST_BOUNDARY_ROOT", "/tmp/project")
	assertEnvVar(t, env, "FORST_NODE_PROTOCOL", envNodeProtocolDefault)

	raw := envValue(t, env, "FORST_FILES_EXCLUDE")
	var patterns []string
	if err := json.Unmarshal([]byte(raw), &patterns); err != nil {
		t.Fatalf("FORST_FILES_EXCLUDE JSON: %v", err)
	}
	if len(patterns) != 2 || patterns[0] != "**/node_modules/**" {
		t.Fatalf("patterns = %#v", patterns)
	}
}

func TestBuildNodeChildEnv_overridesExistingKeys(t *testing.T) {
	env := buildNodeChildEnv(ProcessOptions{
		Env:          []string{"FORST_BOUNDARY_ROOT=old", "PATH=/bin"},
		BoundaryRoot: "/new/root",
	})
	assertEnvVar(t, env, "FORST_BOUNDARY_ROOT", "/new/root")
	assertEnvVar(t, env, "PATH", "/bin")
}

func TestBuildNodeChildEnv_hostAppReadyModulePreservesParentEnviron(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("HOME", "/tmp/home")

	env := buildNodeChildEnv(ProcessOptions{
		Env: []string{"FORST_NODE_APP_READY_MODULE=/tmp/build/server/index.js"},
	})

	assertEnvVar(t, env, "FORST_NODE_APP_READY_MODULE", "/tmp/build/server/index.js")
	assertEnvVar(t, env, "PATH", "/usr/bin:/bin")
	assertEnvVar(t, env, "HOME", "/tmp/home")
}

func assertEnvVar(t *testing.T, env []string, key, want string) {
	t.Helper()
	got := envValue(t, env, key)
	if got != want {
		t.Fatalf("%s = %q want %q", key, got, want)
	}
}

func envValue(t *testing.T, env []string, key string) string {
	t.Helper()
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	t.Fatalf("missing env var %q in %#v", key, env)
	return ""
}

func TestConfigureFromManifest_usesBoundaryRootEnv(t *testing.T) {
	root := t.TempDir()
	bootstrapRel := filepath.Join("custom", "bootstrap.js")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, bootstrapRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, bootstrapRel), []byte("// bootstrap"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(root, "ftconfig.json")
	nodePath := filepath.Join(root, "opt-node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfgJSON := `{
  "node": {
    "binary": "` + filepath.ToSlash(nodePath) + `",
    "bootstrap": "custom/bootstrap.js"
  }
}`
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvBoundaryRoot, root)
	manifestJSON := `{"version":1,"exports":[{"moduleId":"legacy/payment.ts","name":"create","kind":"function"}]}`

	resetSupervisorForTest()
	t.Setenv(envNodeBootstrap, "")
	t.Setenv(envNodeBinary, "")

	if err := configureFromManifest(manifestJSON); err != nil {
		t.Fatal(err)
	}

	wantRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	gotRoot, err := filepath.EvalSymlinks(supervisorCfg.Manifest.BoundaryRoot)
	if err != nil {
		t.Fatal(err)
	}
	if gotRoot != wantRoot {
		t.Fatalf("BoundaryRoot = %q want %q", gotRoot, wantRoot)
	}
}

func TestConfigureFromManifest_discoversBoundaryRootWhenEmbeddedOmitsIt(t *testing.T) {
	root := t.TempDir()
	bootstrapRel := filepath.Join("custom", "bootstrap.js")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, bootstrapRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, bootstrapRel), []byte("// bootstrap"), 0o644); err != nil {
		t.Fatal(err)
	}

	nodePath := filepath.Join(root, "opt-node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(root, "ftconfig.json")
	cfgJSON := `{
  "files": { "exclude": ["**/secret/**"] },
  "node": {
    "binary": "` + filepath.ToSlash(nodePath) + `",
    "bootstrap": "custom/bootstrap.js",
    "loader": "tsx",
    "rpc": {
      "maxMessageBytes": 8192,
      "callTimeoutSeconds": 45
    }
  }
}`
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(root, "src")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}

	manifestJSON := `{"version":1,"exports":[{"moduleId":"legacy/payment.ts","name":"create","kind":"function"}]}`

	resetSupervisorForTest()
	t.Setenv(envNodeBootstrap, "")
	t.Setenv(envNodeBinary, "")

	if err := configureFromManifest(manifestJSON); err != nil {
		t.Fatal(err)
	}

	wantRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	gotRoot, err := filepath.EvalSymlinks(supervisorCfg.Manifest.BoundaryRoot)
	if err != nil {
		t.Fatal(err)
	}
	if gotRoot != wantRoot {
		t.Fatalf("BoundaryRoot = %q want %q", gotRoot, wantRoot)
	}
	wantNode, err := filepath.EvalSymlinks(nodePath)
	if err != nil {
		t.Fatal(err)
	}
	if supervisorCfg.ProcessOptions.NodePath != wantNode {
		t.Fatalf("NodePath = %q want %q", supervisorCfg.ProcessOptions.NodePath, wantNode)
	}
}

func TestConfigureFromManifest_readsFtconfigNodeSection(t *testing.T) {
	root := t.TempDir()
	bootstrapRel := filepath.Join("custom", "bootstrap.js")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, bootstrapRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, bootstrapRel), []byte("// bootstrap"), 0o644); err != nil {
		t.Fatal(err)
	}

	nodePath := filepath.Join(root, "opt-node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(root, "ftconfig.json")
	cfgJSON := `{
  "files": { "exclude": ["**/secret/**"] },
  "node": {
    "binary": "` + filepath.ToSlash(nodePath) + `",
    "bootstrap": "custom/bootstrap.js",
    "loader": "tsx",
    "rpc": {
      "maxMessageBytes": 8192,
      "callTimeoutSeconds": 45
    }
  }
}`
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	t.Setenv(envNodeBootstrap, "")
	t.Setenv(envNodeBinary, "")

	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	wantNode, err := filepath.EvalSymlinks(nodePath)
	if err != nil {
		t.Fatal(err)
	}
	if supervisorCfg.ProcessOptions.NodePath != wantNode {
		t.Fatalf("NodePath = %q want %q", supervisorCfg.ProcessOptions.NodePath, wantNode)
	}
	wantBootstrap, err := filepath.Abs(filepath.Join(root, "custom/bootstrap.js"))
	if err != nil {
		t.Fatal(err)
	}
	if supervisorCfg.ProcessOptions.BootstrapPath != wantBootstrap {
		t.Fatalf("BootstrapPath = %q want %q", supervisorCfg.ProcessOptions.BootstrapPath, wantBootstrap)
	}
	if supervisorCfg.ProcessOptions.Loader != "tsx" {
		t.Fatalf("Loader = %q", supervisorCfg.ProcessOptions.Loader)
	}
	if len(supervisorCfg.ProcessOptions.FilesExclude) != 1 || supervisorCfg.ProcessOptions.FilesExclude[0] != "**/secret/**" {
		t.Fatalf("FilesExclude = %#v", supervisorCfg.ProcessOptions.FilesExclude)
	}
	if supervisorCfg.RPC.MaxMessageBytes != 8192 {
		t.Fatalf("MaxMessageBytes = %d", supervisorCfg.RPC.MaxMessageBytes)
	}
	if supervisorCfg.RPC.CallTimeout.Seconds() != 45 {
		t.Fatalf("CallTimeout = %s", supervisorCfg.RPC.CallTimeout)
	}
}

func TestConfigureFromManifest_hostModeRequiresArgs(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "ftconfig.json")
	cfgJSON := `{
  "node": {
    "hostMode": true,
    "args": []
  }
}`
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	if err := configureFromManifest(string(manifestJSON)); err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "hostMode requires non-empty node.args") {
		t.Fatalf("err = %v", err)
	}
}
