package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestInferFunctionReturnType_testFunctionWithEnsureBoolConstraints(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main
import "testing"

type Sample = { active: Bool, enabled: Bool }

func makeSample(): Sample {
  return Sample { active: false, enabled: true }
}

func TestSampleFields(t *testing.T) {
  s := makeSample()
  ensure s.active is False()
  ensure s.enabled is True()
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	sig, ok := tc.Functions["TestSampleFields"]
	if !ok {
		t.Fatal("missing TestSampleFields signature")
	}
	if !IsVoidReturnTypes(sig.ReturnTypes) {
		t.Fatalf("TestSampleFields return types = %#v, want void", sig.ReturnTypes)
	}
}

func TestInferFunctionReturnType_emptyBodyNoReturnAnnotation(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "noop"},
		Body:  nil,
	}
	got, err := tc.inferFunctionReturnType(fn)
	if err != nil {
		t.Fatal(err)
	}
	if !IsVoidReturnTypes(got) {
		t.Fatalf("got %v", got)
	}
}

func TestInferFunctionReturnType_singleIntReturn(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main
func f(): Int {
	return 42
}
func main() {}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	sig := tc.Functions["f"]
	if len(sig.ReturnTypes) != 1 || sig.ReturnTypes[0].Ident != ast.TypeInt {
		t.Fatalf("return types = %#v", sig.ReturnTypes)
	}
}

func TestInferFunctionReturnType_nilReturnWithErrorType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main
func f(): Error {
	return nil
}
func main() {}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}

func TestInferFunctionReturnType_nilReturnNonNilableErrors(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main
func f(): Int {
	return nil
}
func main() {}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("got %v", err)
	}
}

func TestInferFunctionReturnType_resultFromExpression(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main
func inner(): Result(Int, Error) {
	return 1
}
func outer(): Result(Int, Error) {
	return inner()
}
func main() {}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}
