package nodeinterop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindNodeRuntimeIndexerCLI_prefersProjectNodeModules(t *testing.T) {
	dir := t.TempDir()
	cliJS := filepath.Join(dir, "node_modules", "@forst", "node-runtime", "dist", "indexer", "cli.js")
	if err := os.MkdirAll(filepath.Dir(cliJS), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cliJS, []byte("// cli\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	shim := filepath.Join(binDir, "forst-node-index")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho shim\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	got, err := findNodeRuntimeIndexerCLI(dir)
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

func TestFindNodeRuntimeIndexerCLI_skipsShellShimOnPath(t *testing.T) {
	mono := filepath.Join(repoRoot(), "packages", "node-runtime", "dist", "indexer", "cli.js")
	if _, err := os.Stat(mono); err == nil {
		t.Skip("monorepo node-runtime present; shell shim avoidance covered by prefersProjectNodeModules")
	}
	dir := t.TempDir()
	binDir := t.TempDir()
	shim := filepath.Join(binDir, "forst-node-index")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho shim\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	_, err := findNodeRuntimeIndexerCLI(dir)
	if err == nil {
		t.Fatal("expected error when only shell shim on PATH")
	}
	if !strings.Contains(err.Error(), "node-runtime CLI not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindNodeRuntimeIndexerCLI_monorepoFallback(t *testing.T) {
	mono := filepath.Join(repoRoot(), "packages", "node-runtime", "dist", "indexer", "cli.js")
	if _, err := os.Stat(mono); err != nil {
		t.Skip("monorepo node-runtime not built:", err)
	}
	got, err := findNodeRuntimeIndexerCLI(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != mono {
		t.Fatalf("got %q want %q", got, mono)
	}
}
