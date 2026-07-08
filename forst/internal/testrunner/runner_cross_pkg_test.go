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

	apiDir := filepath.Join(root, "api")
	pkg := PackageUnderTest{
		Dir:     apiDir,
		RelPath: "api",
		FtPaths: []string{
			filepath.Join(apiDir, "handle.ft"),
			filepath.Join(apiDir, "handle_test.ft"),
		},
	}
	code, err := emitPackageGo(root, pkg, modResult, EmitOptions{}, log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code, "func HandleRequest(providers ") {
		t.Fatalf("HandleRequest missing providers param:\n%s", code)
	}
	if !strings.Contains(code, "auth.LogEvent(") {
		t.Fatalf("missing auth.LogEvent call:\n%s", code)
	}
	if !strings.Contains(code, "auth.LogEvent(auth.Providers_") {
		t.Fatalf("expected providers forwarded to auth.LogEvent, got:\n%s", code)
	}
}
