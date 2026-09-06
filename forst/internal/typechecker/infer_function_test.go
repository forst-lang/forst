package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/testutil"

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

func TestCheckTypes_voidEarlyReturn_typechecks(t *testing.T) {
	t.Parallel()
	src := `package main
func early(ok Bool) {
	if !ok {
		return
	}
	println("ok")
}
func main() {
	early(false)
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestCheckTypes_blankIdentAssignment_typechecks(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	_ = 1
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestCheckTypes_blankIdentRange_typechecks(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	xs := ["a", "b"]
	for _, x := range xs {
		println(x)
	}
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestCheckTypes_blankOnlyShortDecl_errors(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	_ := 1
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{})
	if err == nil {
		t.Fatal("expected error for blank-only short declaration")
	}
}

func TestCheckTypes_voidBareReturn_trailingValueExpr_errors(t *testing.T) {
	t.Parallel()
	src := `package main
func f() {
	if true {
		return
	}
	1
}
func main() {}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{})
	if err == nil {
		t.Fatal("expected error for non-void trailing expression after void return")
	}
}

func TestCheckTypes_voidBareReturn_trailingPrintln_typechecks(t *testing.T) {
	t.Parallel()
	src := `package main
func f() {
	if true {
		return
	}
	println("ok")
}
func main() {}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestCheckTypes_multiValueReturn_typechecks(t *testing.T) {
	t.Parallel()
	src := `package main
func boom(): (String, error) {
	return "", nil
}
func main() {
	s, err := boom()
	println(s)
	println(err)
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}
