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

func TestInferredTypesForVariableIdentifier_errInEnsureBody(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	src := `package main

import "fmt"

func checkConditions(): Error {
	return nil
}

func main() {
	err := checkConditions()
	ensure !err {
		fmt.Println(err.Error())
	}
}
`
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	ty, ok := tc.InferredTypesForVariableIdentifier("err")
	if !ok || len(ty) != 1 {
		t.Fatalf("err: ok=%v types=%v", ok, ty)
	}
	if ty[0].Ident != ast.TypeError {
		t.Fatalf("want TYPE_ERROR, got %#v", ty[0])
	}
}

func TestGoHoverMarkdownForForstReceiverMethod_errorError(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	md, ok := tc.GoHoverMarkdownForForstReceiverMethod(ast.TypeNode{Ident: ast.TypeError}, "Error")
	if !ok {
		t.Fatal("expected hover for Error.Error")
	}
	if !strings.Contains(md, "func (error).Error()") && !strings.Contains(md, "Error() string") {
		t.Fatalf("expected go/types error.Error signature, got %q", md)
	}
	if !strings.Contains(md, "predeclared") {
		t.Fatalf("expected predeclared error mention, got %q", md)
	}
	if !strings.Contains(md, "pkg.go.dev/builtin") {
		t.Fatalf("expected link to package builtin docs, got %q", md)
	}
	// From $GOROOT/src/builtin/builtin.go (via go/doc)
	if !strings.Contains(strings.ToLower(md), "conventional") {
		t.Fatalf("expected builtin doc excerpt for error interface, got %q", md)
	}
}

func TestGoHoverMarkdownForForstReceiverMethod_wrongReceiver(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	if _, ok := tc.GoHoverMarkdownForForstReceiverMethod(ast.TypeNode{Ident: ast.TypeString}, "Error"); ok {
		t.Fatal("String should not get error.Error hover")
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
