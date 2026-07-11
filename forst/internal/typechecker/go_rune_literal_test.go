package typechecker

import (
	"errors"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func TestRuneLiteral_infersInt(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
  x := 'f'
  println(x)
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
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}

func TestGoQualifiedCall_FormatFloat_runeLiteral_typechecks(t *testing.T) {
	dir := moduleRootFromWD(t)
	src := `package main
import "strconv"
func main() {
  println(strconv.FormatFloat(1.0, 'f', 0, 64))
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
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["strconv"] == nil {
		t.Skip("strconv not loaded (go/packages or workspace)")
	}
}

func TestBuiltinCall_string_runeLiteral_typechecks(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
  s := string('A')
  println(s)
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestGoQualifiedCall_FormatFloat_stringLiteral_rejects(t *testing.T) {
	dir := moduleRootFromWD(t)
	src := `package main
import "strconv"
func main() {
  println(strconv.FormatFloat(1.0, "f", 0, 64))
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
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected type error for string literal where byte expected")
	}
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["strconv"] == nil {
		t.Skip("strconv not loaded (go/packages or workspace)")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag.Code != "go-call" {
		t.Fatalf("expected go-call diagnostic, got %v", err)
	}
}

func TestBuiltinCall_FormatFloat_runeLiteral_typechecks(t *testing.T) {
	t.Parallel()
	src := `package main
import "strconv"
func main() {
  println(strconv.FormatFloat(1.0, 'f', 0, 64))
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
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	builtin, ok := BuiltinFunctions["strconv.FormatFloat"]
	if !ok {
		t.Fatal("strconv.FormatFloat builtin missing")
	}
	if len(builtin.ParamTypes) != 4 {
		t.Fatalf("want 4 params, got %d", len(builtin.ParamTypes))
	}
}
