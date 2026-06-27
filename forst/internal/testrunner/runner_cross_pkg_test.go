package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/modulecheck"

	"github.com/sirupsen/logrus"
)

func TestEmit_crossPkgHandleForwardsProviders(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.ErrorLevel)

	modResult, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{ModuleRoot: root})
	if err != nil {
		t.Fatal(err)
	}

	betaDir := filepath.Join(root, "beta")
	pkg := PackageUnderTest{
		Dir:     betaDir,
		RelPath: "beta",
		FtPaths: []string{
			filepath.Join(betaDir, "handle.ft"),
			filepath.Join(betaDir, "handle_test.ft"),
		},
	}
	code, err := emitPackageGo(root, pkg, modResult, log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code, "func Handle(providers ") {
		t.Fatalf("Handle missing providers param:\n%s", code)
	}
	if !strings.Contains(code, "alpha.LogExpiry(") {
		t.Fatalf("missing alpha.LogExpiry call:\n%s", code)
	}
	if !strings.Contains(code, "alpha.LogExpiry(alpha.Providers_") {
		t.Fatalf("expected providers forwarded to alpha.LogExpiry, got:\n%s", code)
	}
}
