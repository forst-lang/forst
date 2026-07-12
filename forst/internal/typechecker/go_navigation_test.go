package typechecker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func TestGoSamePackageFuncDefinitionLocation_add(t *testing.T) {
	t.Parallel()
	root, importPath := testutil.WriteMixedGoForstModule(t, "memos")
	tc := New(nil, false)
	tc.GoWorkspaceDir = root
	tc.ForstFileDir = filepath.Join(root, "memos")
	tc.SetSamePackageGoImportPath(importPath)
	tc.initSamePackageGoExports()
	if !tc.SamePackageGoLoaded() {
		t.Skip("same-package Go not loaded")
	}
	loc, ok := tc.GoSamePackageFuncDefinitionLocation("Add")
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.HasSuffix(loc.File, "helpers.go") {
		t.Fatalf("file = %q", loc.File)
	}
	if loc.Line <= 0 {
		t.Fatalf("line = %d", loc.Line)
	}
}

func TestGoQualifiedExportDefinitionLocation_execCommand(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module execdef\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const src = `package main
import "os/exec"
func main() {
  exec.Command("true")
}
`
	ftPath := filepath.Join(root, "main.ft")
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		GoWorkspaceDir:     root,
		ForstFileDir:       root,
		SkipUnlessGoImport: "exec",
	})
	loc, ok := tc.GoQualifiedExportDefinitionLocation("exec", "Command")
	if !ok {
		t.Fatal("expected ok")
	}
	if loc.File == "" || loc.Line <= 0 {
		t.Fatalf("loc=%+v", loc)
	}
}

func TestGoReceiverMethodDefinitionLocation_cmdRun(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	const src = `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.Run()
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	if !tc.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}
	if tc.GoTypeForVariable("cmd") == nil {
		t.Fatal("expected Go type for cmd")
	}
	loc, ok := tc.GoReceiverMethodDefinitionLocation("cmd", "Run")
	if !ok {
		t.Fatal("expected ok")
	}
	if loc.File == "" || loc.Line <= 0 {
		t.Fatalf("loc=%+v", loc)
	}
}
