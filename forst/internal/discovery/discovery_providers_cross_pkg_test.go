package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func writeCrossPackageProvidersFixture(t *testing.T, dir string) (alphaFt, betaFt string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("testmod")), 0o644); err != nil {
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
	alphaFt = filepath.Join(alphaDir, "expire.ft")
	if err := os.WriteFile(alphaFt, []byte(`package alpha

type Logger = { Info(msg String) }

func ExpireToken() {
	use logger: Logger
	logger.Info("expire")
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

func TestDiscoverer_DiscoverFunctions_crossPackageProvidersPropagation(t *testing.T) {
	dir := t.TempDir()
	alphaFt, betaFt := writeCrossPackageProvidersFixture(t, dir)

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
	if len(handle.Providers) != 1 || handle.Providers[0] != "Logger" {
		t.Fatalf("Handle providers = %v, want [Logger]", handle.Providers)
	}
}

func TestDiscoverer_DiscoverProvidersJSONV1_crossPackage(t *testing.T) {
	dir := t.TempDir()
	alphaFt, betaFt := writeCrossPackageProvidersFixture(t, dir)

	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	logger.SetLevel(logrus.PanicLevel)
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{alphaFt, betaFt}})

	doc, err := discoverer.DiscoverProvidersJSONV1()
	if err != nil {
		t.Fatalf("DiscoverProvidersJSONV1: %v", err)
	}
	if doc.Version != 1 {
		t.Fatalf("version = %d", doc.Version)
	}
	betaPkg := doc.Packages["beta"]
	handle := betaPkg.Functions["Handle"]
	if handle.Runnable || len(handle.Providers) != 1 || handle.Providers[0] != "Logger" {
		t.Fatalf("Handle v1 doc = %+v", handle)
	}
}
