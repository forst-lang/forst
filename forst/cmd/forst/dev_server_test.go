package main

import (
	"io"
	"testing"

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
	comp := compiler.New(cfg.ToCompilerArgs(), log)
	return NewHTTPServer("0", comp, log, cfg, t.TempDir())
}

func TestNewHTTPServer_initializesTypesCache(t *testing.T) {
	s := testDevServer(t)
	if s.typesCache == nil {
		t.Fatal("typesCache must be non-nil for /types caching")
	}
}
