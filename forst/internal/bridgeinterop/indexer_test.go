package bridgeinterop

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"
)

func TestFindBridgeRuntimeIndexerCLI_prefersProjectNodeModules(t *testing.T) {
	dir := t.TempDir()
	cliJS := filepath.Join(dir, "node_modules", "@forst", "runtime", "dist", "indexer", "cli.js")
	if err := os.MkdirAll(filepath.Dir(cliJS), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cliJS, []byte("// cli\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	shim := filepath.Join(binDir, "forst-runtime-index")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho shim\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	got, err := findBridgeRuntimeIndexerCLI(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != cliJS {
		t.Fatalf("got %q want %q", got, cliJS)
	}
	if strings.Contains(got, ".bin") {
		t.Fatalf("must not use shell shim: %q", got)
	}
}

func TestFindBridgeRuntimeIndexerCLI_skipsShellShimOnPath(t *testing.T) {
	mono := filepath.Join(repoRoot(), "packages", "runtime", "dist", "indexer", "cli.js")
	if _, err := os.Stat(mono); err == nil {
		t.Skip("monorepo @forst/runtime present; shell shim avoidance covered by prefersProjectNodeModules")
	}
	dir := t.TempDir()
	binDir := t.TempDir()
	shim := filepath.Join(binDir, "forst-runtime-index")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho shim\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	_, err := findBridgeRuntimeIndexerCLI(dir)
	if err == nil {
		t.Fatal("expected error when only shell shim on PATH")
	}
	if !strings.Contains(err.Error(), "@forst/runtime CLI not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindBridgeRuntimeIndexerCLI_monorepoFallback(t *testing.T) {
	mono := filepath.Join(repoRoot(), "packages", "runtime", "dist", "indexer", "cli.js")
	if _, err := os.Stat(mono); err != nil {
		t.Skip("monorepo @forst/runtime not built:", err)
	}
	got, err := findBridgeRuntimeIndexerCLI(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != mono {
		t.Fatalf("got %q want %q", got, mono)
	}
}

func TestResolveIndexerInterpreter_ignoresHostModeAppBinary(t *testing.T) {
	root := t.TempDir()
	remixBin := filepath.Join(root, "node_modules", ".bin", "remix-serve")
	if err := os.MkdirAll(filepath.Dir(remixBin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(remixBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH")
	}
	bridge := ftconfig.Bridge{Host: ftconfig.BridgeHostNode}
	got, err := resolveIndexerInterpreter(root, bridge, remixBin)
	if err != nil {
		t.Fatal(err)
	}
	if got != nodePath {
		t.Fatalf("interpreter = %q want node %q", got, nodePath)
	}
}
