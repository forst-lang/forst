package typechecker

import (
	"go/types"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestGoHoverMarkdown_fmtPrintln(t *testing.T) {
	dir := moduleRootFromWD(t)
	src := "package main\nimport \"fmt\"\nfunc main() {\n\tfmt.Println(\"x\")\n}\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	_ = tc.CheckTypes(nodes)

	md, ok := tc.GoHoverMarkdown("fmt", "Println")
	if !ok {
		t.Fatal("expected Go hover for fmt.Println")
	}
	if !strings.Contains(md, "Println") {
		t.Fatalf("expected signature to mention Println, got %q", md)
	}
	if tc.goPkgsByLocal != nil && tc.goPkgsByLocal["fmt"] != nil {
		if !strings.Contains(md, "func") {
			t.Fatalf("expected go/types func line in hover, got %q", md)
		}
	}
}

func TestGoHoverMarkdownForImportPath_fmt(t *testing.T) {
	dir := moduleRootFromWD(t)
	src := "package main\nimport \"fmt\"\nfunc main() {}\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	_ = tc.CheckTypes(nodes)

	md, ok := tc.GoHoverMarkdownForImportPath("fmt")
	if !ok {
		t.Fatal("expected hover for import path fmt")
	}
	if !strings.Contains(md, "fmt") {
		t.Fatalf("expected path or name fmt in hover, got %q", md)
	}
}

func TestGoHoverMarkdownForImportPath_unknownStillShows(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tc := New(log, false)
	md, ok := tc.GoHoverMarkdownForImportPath("example.com/nope/nope")
	if !ok {
		t.Fatal("expected generic hover for unknown path")
	}
	if !strings.Contains(md, "example.com/nope/nope") {
		t.Fatalf("got %q", md)
	}
}

func TestIsImportedLocalName(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := "package main\nimport \"fmt\"\nfunc main() {}\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	_ = tc.CheckTypes(nodes)

	if !tc.IsImportedLocalName("fmt") {
		t.Fatal("expected fmt as imported local name")
	}
	if tc.IsImportedLocalName("notimported") {
		t.Fatal("unexpected import name")
	}
}

func TestGoSignatureReturnsToForst_void(t *testing.T) {
	t.Parallel()
	sig := types.NewSignature(nil, nil, nil, false)
	out := goSignatureReturnsToForst(sig)
	if len(out) != 1 || out[0].Ident != ast.TypeVoid {
		t.Fatalf("want single Void, got %#v", out)
	}
}

func TestGoSignatureReturnsToForst_int(t *testing.T) {
	t.Parallel()
	results := types.NewTuple(types.NewParam(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignature(nil, nil, results, false)
	out := goSignatureReturnsToForst(sig)
	if len(out) != 1 || out[0].Ident != ast.TypeInt {
		t.Fatalf("got %#v", out)
	}
}

func TestGoSignatureReturnsToForst_unsupportedReturnYieldsNil(t *testing.T) {
	t.Parallel()
	// unsafe.Pointer is a basic kind that goTypeToForstType does not map.
	results := types.NewTuple(types.NewParam(0, nil, "", types.Typ[types.UnsafePointer]))
	sig := types.NewSignature(nil, nil, results, false)
	if out := goSignatureReturnsToForst(sig); out != nil {
		t.Fatalf("expected nil, got %#v", out)
	}
}

func TestGoHoverMarkdown_packageOnly(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := "package main\nimport \"fmt\"\nfunc main() {}\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	_ = tc.CheckTypes(nodes)

	md, ok := tc.GoHoverMarkdown("fmt", "")
	if !ok {
		t.Fatal("expected package-only hover")
	}
	if !strings.Contains(md, "fmt") || !strings.Contains(md, "import") {
		t.Fatalf("expected package hover, got %q", md)
	}
}
