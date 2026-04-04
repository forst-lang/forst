package typechecker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func moduleRootFromWD(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

func TestGoQualifiedCall_wrongArgType(t *testing.T) {
	dir := moduleRootFromWD(t)
	// Use crypto/md5 (not in BuiltinFunctions): wrong arg types must fail via go/types or unknown identifier.
	src := "package main\nimport \"crypto/md5\"\nfunc main() {\n  md5.Sum(1)\n}\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatalf("expected type error (imports=%d goImportPkgs=%d)", len(tc.imports), len(tc.goPkgsByLocal))
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag == nil {
		t.Fatalf("expected *Diagnostic, got %T: %v", err, err)
	}
	if !diag.Span.IsSet() {
		t.Fatal("expected diagnostic span")
	}
}
