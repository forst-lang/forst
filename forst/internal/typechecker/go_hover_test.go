package typechecker

import (
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func typecheckSrc(tb testing.TB, src string, opts testutil.TypecheckOpts) *TypeChecker {
	tb.Helper()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "test.ft", log).Lex()
	nodes, err := parser.New(toks, "test.ft", log).ParseFile()
	if err != nil {
		tb.Fatal(err)
	}
	tc := New(log, false)
	applyTypecheckOpts(tb, tc, opts)
	if err := tc.CheckTypes(nodes); err != nil {
		tb.Fatalf("CheckTypes: %v", err)
	}
	return tc
}

func TestGoHoverMarkdown_fmtPrintln_includesGodoc(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	tc := typecheckSrc(t, `package main

import "fmt"

func main() {
	fmt.Println("hi")
}
`, testutil.TypecheckOpts{UseModuleRoot: true, SkipUnlessGoImport: "fmt"})
	md, ok := tc.GoHoverMarkdown("fmt", "Println")
	if !ok || md == "" {
		t.Fatal("expected non-empty hover for fmt.Println")
	}
	if !strings.Contains(md, "Println") {
		t.Fatalf("hover should mention Println: %q", md)
	}
	if !strings.Contains(md, "format") && !strings.Contains(md, "Format") {
		t.Fatalf("hover should include godoc about formatting: %q", md)
	}
}

func TestGoHoverMarkdownForGoTypeMethod_execCmdRun(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	tc := typecheckSrc(t, `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.Run()
}
`, testutil.TypecheckOpts{UseModuleRoot: true, SkipUnlessGoImport: "exec"})
	cmdType := tc.GoTypeForVariable(ast.Identifier("cmd"))
	if cmdType == nil {
		t.Fatal("expected Go type for cmd variable")
	}
	md, ok := tc.GoHoverMarkdownForGoTypeMethod(cmdType, "os/exec", "Run")
	if !ok || md == "" {
		t.Fatal("expected non-empty hover for (*exec.Cmd).Run")
	}
	if !strings.Contains(md, "Run") {
		t.Fatalf("hover should mention Run: %q", md)
	}
}

func TestGoHoverMarkdownSamePackageFunc_includesGodoc(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	root, importPath := testutil.WriteMixedGoForstModule(t, "memos")
	src := `package memos

func main() {
  x := Add(1, 2)
  println(x)
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "memos/main.ft", log).Lex()
	nodes, err := parser.New(toks, "memos/main.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.ConfigureForForstFile(root, filepath.Join(root, "memos"), nodes)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	if !tc.SamePackageGoLoaded() {
		t.Fatalf("expected same-package Go loaded for %q", importPath)
	}
	md, ok := tc.GoHoverMarkdownSamePackageFunc("StringFromBytes")
	if !ok || md == "" {
		t.Fatal("expected non-empty hover for StringFromBytes")
	}
	if !strings.Contains(md, "StringFromBytes") {
		t.Fatalf("hover should mention StringFromBytes: %q", md)
	}
	if !strings.Contains(md, "memo file") {
		t.Fatalf("hover should include godoc from fixture: %q", md)
	}
}

func TestGoHoverMarkdownForMethodOnExpression_includesGodoc(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	tc := typecheckSrc(t, `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.Run()
}
`, testutil.TypecheckOpts{UseModuleRoot: true, SkipUnlessGoImport: "exec"})
	recv := ast.VariableNode{Ident: ast.Ident{ID: "cmd"}}
	md, ok := tc.GoHoverMarkdownForMethodOnExpression(recv, "Run")
	if !ok || md == "" {
		t.Fatal("expected non-empty hover for (*exec.Cmd).Run via expression path")
	}
	if !strings.Contains(md, "Run") {
		t.Fatalf("hover should mention Run: %q", md)
	}
}

func TestGoHoverMarkdownForGoTypeMethod_namedReceiver(t *testing.T) {
	t.Parallel()
	moduleRoot := testutil.ModuleRoot(t)
	m, err := goload.LoadByPkgPath(moduleRoot, []string{"os"})
	if err != nil {
		t.Fatalf("LoadByPkgPath: %v", err)
	}
	gp := m["os"]
	if gp == nil || gp.Types == nil {
		t.Fatal("os package not loaded")
	}
	fileInfo := gp.Types.Scope().Lookup("FileInfo").Type()
	if fileInfo == nil {
		t.Fatal("os.FileInfo type not found")
	}
	tc := New(nil, false)
	tc.GoWorkspaceDir = moduleRoot
	md, ok := tc.GoHoverMarkdownForGoTypeMethod(fileInfo, "os", "Name")
	if !ok || md == "" {
		t.Fatal("expected non-empty hover")
	}
	if !strings.Contains(md, "Name") {
		t.Fatalf("hover should mention Name: %q", md)
	}
}

func TestGoHoverMarkdownDotImportedSymbol_stringsContains(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	tc := typecheckSrc(t, `package main

import . "strings"

func main() {
	println(Contains("a", "b"))
}
`, testutil.TypecheckOpts{UseModuleRoot: true, SkipUnlessGoImport: "strings"})
	if !tc.HasDotImportPackages() {
		t.Skip("strings dot-import not loaded (go/packages or workspace)")
	}
	md, ok := tc.GoHoverMarkdownDotImportedSymbol("Contains")
	if !ok || md == "" {
		t.Fatal("expected non-empty hover for dot-imported Contains")
	}
	if !strings.Contains(md, "Contains") || !strings.Contains(md, "```go") {
		t.Fatalf("hover should include Go signature: %q", md)
	}
}
