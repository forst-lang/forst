package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func writeCrossPackageProvidersFixture(t *testing.T, dir string) (authFt, apiFt string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("testmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(dir, "auth")
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	authFt = filepath.Join(authDir, "log.ft")
	if err := os.WriteFile(authFt, []byte(`package auth

type Logger = { Info(msg String) }

func LogEvent() {
	use logger: Logger
	logger.Info("expire")
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	apiFt = filepath.Join(apiDir, "handle.ft")
	if err := os.WriteFile(apiFt, []byte(`package api

import "testmod/auth"

func HandleRequest() {
	auth.LogEvent()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return authFt, apiFt
}

func TestDiscoverer_DiscoverFunctions_crossPackageProvidersPropagation(t *testing.T) {
	dir := t.TempDir()
	authFt, apiFt := writeCrossPackageProvidersFixture(t, dir)

	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	logger.SetLevel(logrus.PanicLevel)
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{authFt, apiFt}})

	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	handle, ok := functions["api"]["HandleRequest"]
	if !ok {
		t.Fatalf("expected api.HandleRequest, got %+v", functions)
	}
	if handle.Runnable {
		t.Fatal("HandleRequest should not be runnable after cross-package propagation")
	}
	if len(handle.Providers) != 1 || handle.Providers[0] != "Logger" {
		t.Fatalf("HandleRequest providers = %v, want [Logger]", handle.Providers)
	}
}

func TestDiscoverer_DiscoverProvidersJSONV1_crossPackage(t *testing.T) {
	dir := t.TempDir()
	authFt, apiFt := writeCrossPackageProvidersFixture(t, dir)

	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	logger.SetLevel(logrus.PanicLevel)
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{authFt, apiFt}})

	doc, err := discoverer.DiscoverProvidersJSONV1()
	if err != nil {
		t.Fatalf("DiscoverProvidersJSONV1: %v", err)
	}
	if doc.Version != 1 {
		t.Fatalf("version = %d", doc.Version)
	}
	apiPkg := doc.Packages["api"]
	handle := apiPkg.Functions["HandleRequest"]
	if handle.Runnable || len(handle.Providers) != 1 || handle.Providers[0] != "Logger" {
		t.Fatalf("HandleRequest v1 doc = %+v", handle)
	}
}
