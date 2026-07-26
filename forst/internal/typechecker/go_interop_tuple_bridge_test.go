package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestGoInterop_bridgeIdiomBuildsResult(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "strconv"
func atoi(s: String): Result(Int, Error) {
  n, err := strconv.Atoi(s)
  ensure !err or err
  return n
}
func main() {
  x := atoi("42")
  ensure x is Ok()
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
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
}

func TestGoInterop_okNarrowingOnTupleRejected(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "strconv"
func main() {
  x := strconv.Atoi("42")
  ensure x is Ok()
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
		t.Fatal("expected type error for ensure x is Ok() on Tuple")
	}
}

func TestGoInterop_noTupleToResultConversion(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "strconv"
func main() {
  var x: Result(Int, Error)
  x = strconv.Atoi("42")
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
		t.Fatal("expected type error assigning Tuple to Result")
	}
}

func TestGoInterop_twoValueAssignmentStillFlatArity(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "strconv"
func main() {
  v, err := strconv.Atoi("42")
  println(v)
  println(err)
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
		t.Fatalf("typecheck: %v", err)
	}
	var asg ast.AssignmentNode
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok || fn.Ident.ID != "main" {
			continue
		}
		asg = fn.Body[0].(ast.AssignmentNode)
		break
	}
	v0 := asg.LValues[0].(ast.VariableNode)
	v1 := asg.LValues[1].(ast.VariableNode)
	tv0, _ := tc.LookupInferredType(v0, true)
	tv1, _ := tc.LookupInferredType(v1, true)
	if len(tv0) != 1 || tv0[0].Ident != ast.TypeInt {
		t.Fatalf("want v: Int, got %v", tv0)
	}
	if len(tv1) != 1 || tv1[0].Ident != ast.TypeError {
		t.Fatalf("want err: Error, got %v", tv1)
	}
}

func TestGoInterop_tupleIndexedAccess(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "strconv"
func main() {
  x := strconv.Atoi("42")
  n := x.0
  e := x.1
  println(n)
  println(e)
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
		t.Fatalf("typecheck: %v", err)
	}
}

func TestGoInterop_multiValueReturnIntoResultRejected(t *testing.T) {
	t.Parallel()
	src := `package main
func f(): Result(Int, Error) {
  return 1, 2
}
func main() {}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected error for multi-value return into Result-declared function")
	}
}
