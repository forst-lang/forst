package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/codegen/layout"
	"forst/internal/compiler"

	"github.com/sirupsen/logrus"
)

func testDevServer(t *testing.T) *DevServer {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	cfg := DefaultConfig()
	cfg.Server.ReadTimeout = 1
	cfg.Server.WriteTimeout = 1
	falseVal := false
	cfg.Dev.WatchGenerate = &falseVal
	dir := t.TempDir()
	comp := compiler.New(cfg.ToCompilerArgs(), log)
	return NewHTTPServer("0", comp, log, cfg, dir)
}

func writeTestGeneratedTypes(t *testing.T, root, content string) {
	t.Helper()
	typesDir := filepath.Join(layout.NewRoot(root).ClientDir(), "dist")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(typesDir, "types.d.ts"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNewHTTPServer_initializesTypesCache(t *testing.T) {
	s := testDevServer(t)
	if s.typesCache == nil {
		t.Fatal("typesCache must be non-nil for /types caching")
	}
}
