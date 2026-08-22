package nodert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"
)

func TestSpawnHooks_nodeSourceInjectsTsxArgv(t *testing.T) {
	root := t.TempDir()
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	hooks, err := spawnHooks(SpawnHookInput{
		BoundaryRoot: root,
		Bridge:       testBridgeNodeTypeScript(),
		ModuleIDs:    []string{"legacy/payment.ts"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hooks.PrefixArgs) < 2 || hooks.PrefixArgs[0] != "--import" {
		t.Fatalf("PrefixArgs = %#v want --import tsx", hooks.PrefixArgs)
	}
	if !strings.Contains(hooks.PrefixArgs[1], "loader.mjs") {
		t.Fatalf("PrefixArgs = %#v", hooks.PrefixArgs)
	}
}

func TestSpawnHooks_nodePrecompiledSkipsTsx(t *testing.T) {
	root := t.TempDir()
	hooks, err := spawnHooks(SpawnHookInput{
		BoundaryRoot: root,
		Bridge:       testBridgeNodeCompiled(),
		ModuleIDs:    []string{".forst/js/legacy/payment.js"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, arg := range hooks.PrefixArgs {
		if strings.Contains(arg, "tsx") {
			t.Fatalf("PrefixArgs must not contain tsx: %#v", hooks.PrefixArgs)
		}
	}
}

func TestSpawnHooks_bunStripsTsxFromNodeOptions(t *testing.T) {
	root := t.TempDir()
	bunPath := filepath.Join(root, "bun")
	if err := os.WriteFile(bunPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	hooks, err := spawnHooks(SpawnHookInput{
		BoundaryRoot:     root,
		Bridge:           testBridgeBunTypeScript(),
		ConfiguredBinary: bunPath,
		ParentEnv:        []string{"NODE_OPTIONS=--import /app/node_modules/tsx/dist/loader.mjs"},
	})
	if err != nil {
		t.Fatal(err)
	}
	opts := lookupEnvValue(hooks.ExtraEnv, "NODE_OPTIONS")
	if strings.Contains(opts, "tsx") {
		t.Fatalf("NODE_OPTIONS = %q want tsx stripped for bun", opts)
	}
}

func TestSpawnHooks_denoRequiresEnabledFlag(t *testing.T) {
	ftconfig.SetDenoHostEnabledForTest(false)
	t.Cleanup(func() { ftconfig.SetDenoHostEnabledForTest(false) })

	root := t.TempDir()
	_, err := ftconfig.EffectiveJSBridge(&ftconfig.Config{
		Javascript: ftconfig.JavascriptConfig{Host: ftconfig.JSHostDeno},
	})
	if err == nil || !strings.Contains(err.Error(), "deno is not enabled") {
		t.Fatalf("EffectiveJSBridge err = %v", err)
	}

	ftconfig.SetDenoHostEnabledForTest(true)
	bridge, err := ftconfig.EffectiveJSBridge(&ftconfig.Config{
		Javascript: ftconfig.JavascriptConfig{Host: ftconfig.JSHostDeno},
	})
	if err != nil {
		t.Fatal(err)
	}
	hooks, err := spawnHooks(SpawnHookInput{
		BoundaryRoot: root,
		Bridge:       bridge,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hooks.PrefixArgs) == 0 || hooks.PrefixArgs[0] != "run" {
		t.Fatalf("PrefixArgs = %#v want deno run prefix", hooks.PrefixArgs)
	}
	if !strings.Contains(strings.Join(hooks.PrefixArgs, " "), "--unstable-detect-cjs") {
		t.Fatalf("PrefixArgs = %#v want --unstable-detect-cjs", hooks.PrefixArgs)
	}
}

func TestBuildHostSpawnCommand_precompiledOmitsTsx(t *testing.T) {
	root := t.TempDir()
	nodePath := filepath.Join(root, "shim")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot: root,
		Executable:   nodePath,
		ShimArgs:     []string{"server.js"},
		WorkDir:      root,
		Bridge:       testBridgeNodeCompiled(),
		ModuleIDs:    []string{".forst/js/legacy/payment.js"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, arg := range cmd.Args {
		if strings.Contains(arg, "tsx") {
			t.Fatalf("args must not contain tsx: %#v", cmd.Args)
		}
	}
}
