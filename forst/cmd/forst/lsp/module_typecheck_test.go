package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestTypecheckForLSP_crossPkgForstSiblingCall_ignoresEmittedGoArity(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	apiPath := filepath.Join(root, "api", "handle.ft")
	src, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	log.SetOutput(os.Stderr)
	s := NewLSPServer("8080", log)
	uri := mustFileURI(t, apiPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = string(src)
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyzeForstDocument failed")
	}
	if ctx.CheckErr != nil {
		t.Fatalf("expected api.HandleRequest to typecheck via Forst sibling auth.LogEvent, got: %v", ctx.CheckErr)
	}
}

func TestTypecheckForLSP_crossPkgWithEmittedGoStub(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module cross_stub\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(dir, "auth")
	apiDir := filepath.Join(dir, "api")
	for _, d := range []string{authDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	const emittedGo = `package auth

type Providers_stub struct {
	Logger any
}

func LogEvent(providers Providers_stub, id string) {}
`
	if err := os.WriteFile(filepath.Join(authDir, "z_forst_gen.go"), []byte(emittedGo), 0o644); err != nil {
		t.Fatal(err)
	}
	const authFt = `package auth

type Logger = { Info(msg String) }

func LogEvent(id String) {
	use logger: Logger
	logger.Info(id)
}
`
	const apiFt = `package api

import "cross_stub/auth"

func HandleRequest(id String) {
	auth.LogEvent(id)
}
`
	authPath := filepath.Join(authDir, "log.ft")
	apiPath := filepath.Join(apiDir, "handle.ft")
	if err := os.WriteFile(authPath, []byte(authFt), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apiPath, []byte(apiFt), 0o644); err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	log.SetOutput(os.Stderr)
	s := NewLSPServer("8080", log)
	uri := mustFileURI(t, apiPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = apiFt
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyzeForstDocument failed")
	}
	if ctx.CheckErr != nil {
		t.Fatalf("Forst sibling call should win over emitted Go stub arity, got: %v", ctx.CheckErr)
	}
}
