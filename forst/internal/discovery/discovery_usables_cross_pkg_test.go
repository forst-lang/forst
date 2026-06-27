package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func writeCrossPackageUsablesFixture(t *testing.T, dir string) (alphaFt, betaFt string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	alphaDir := filepath.Join(dir, "alpha")
	betaDir := filepath.Join(dir, "beta")
	if err := os.MkdirAll(alphaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(betaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Go stub so beta resolves alpha.ExpireToken and records cross-package Usables calls.
	if err := os.WriteFile(filepath.Join(alphaDir, "stub.go"), []byte(`package alpha

func ExpireToken() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	alphaFt = filepath.Join(alphaDir, "expire.ft")
	if err := os.WriteFile(alphaFt, []byte(`package alpha

type Logger = { info(msg String) }

func ExpireToken() {
	use logger: Logger
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	betaFt = filepath.Join(betaDir, "handle.ft")
	if err := os.WriteFile(betaFt, []byte(`package beta

import "testmod/alpha"

func Handle() {
	alpha.ExpireToken()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return alphaFt, betaFt
}

func TestDiscoverer_DiscoverFunctions_crossPackageUsablesPropagation(t *testing.T) {
	dir := t.TempDir()
	alphaFt, betaFt := writeCrossPackageUsablesFixture(t, dir)

	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	logger.SetLevel(logrus.PanicLevel)
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{alphaFt, betaFt}})

	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	handle, ok := functions["beta"]["Handle"]
	if !ok {
		t.Fatalf("expected beta.Handle, got %+v", functions)
	}
	if handle.Runnable {
		t.Fatal("Handle should not be runnable after cross-package propagation")
	}
	if len(handle.Usables) != 1 || handle.Usables[0] != "Logger" {
		t.Fatalf("Handle usables = %v, want [Logger]", handle.Usables)
	}
}

func TestDiscoverer_DiscoverUsablesJSONV1_crossPackage(t *testing.T) {
	dir := t.TempDir()
	alphaFt, betaFt := writeCrossPackageUsablesFixture(t, dir)

	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	logger.SetLevel(logrus.PanicLevel)
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{alphaFt, betaFt}})

	doc, err := discoverer.DiscoverUsablesJSONV1()
	if err != nil {
		t.Fatalf("DiscoverUsablesJSONV1: %v", err)
	}
	if doc.Version != 1 {
		t.Fatalf("version = %d", doc.Version)
	}
	betaPkg := doc.Packages["beta"]
	handle := betaPkg.Functions["Handle"]
	if handle.Runnable || len(handle.Usables) != 1 || handle.Usables[0] != "Logger" {
		t.Fatalf("Handle v1 doc = %+v", handle)
	}
}
