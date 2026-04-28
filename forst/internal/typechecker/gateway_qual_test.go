package typechecker

import (
	"path/filepath"
	"testing"

	"forst/internal/forstpkg"
	"forst/internal/goload"

	"github.com/sirupsen/logrus"
)

func TestGatewayQualifiedTypes_resolveFromExample(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	path := filepath.Join("..", "..", "..", "examples", "in", "gateway", "hello.ft")
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := forstpkg.ParseForstFile(log, absPath)
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = goload.GoWorkspaceForPackages(absPath)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}
