package typechecker

import (
	"errors"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
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

func TestGoQualifiedCall_twoValueErrorFoldsToResult(t *testing.T) {
	dir := moduleRootFromWD(t)
	src := `package main
import "strconv"
func main() {
  x := strconv.Atoi("42")
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
		t.Fatal(err)
	}
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["strconv"] == nil {
		t.Skip("strconv not loaded (go/packages or workspace)")
	}
	var call ast.FunctionCallNode
	found := false
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok || fn.Ident.ID != "main" {
			continue
		}
		asg, ok := fn.Body[0].(ast.AssignmentNode)
		if !ok {
			t.Fatalf("expected assignment, got %T", fn.Body[0])
		}
		var ok2 bool
		call, ok2 = asg.RValues[0].(ast.FunctionCallNode)
		if !ok2 {
			t.Fatalf("expected function call, got %T", asg.RValues[0])
		}
		found = true
		break
	}
	if !found {
		t.Fatal("main not found")
	}
	ts, err := tc.LookupInferredType(call, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 1 || !ts[0].IsResultType() {
		t.Fatalf("want single Result, got %v", ts)
	}
	if ts[0].TypeParams[0].Ident != ast.TypeInt || ts[0].TypeParams[1].Ident != ast.TypeError {
		t.Fatalf("want Result(Int, Error), got %s", ts[0].String())
	}
}

func TestGoDotImport_unqualifiedCall(t *testing.T) {
	dir := moduleRootFromWD(t)
	src := `package main

import . "strings"

func main() {
	x := Contains("ab", "a")
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
		t.Fatal(err)
	}
	if !tc.HasDotImportPackages() {
		t.Skip("strings dot-import not loaded (go/packages or workspace)")
	}
}

func TestGoQualifiedCall_twoValueAssignmentUnfoldsToPair(t *testing.T) {
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
		t.Fatal(err)
	}
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["strconv"] == nil {
		t.Skip("strconv not loaded (go/packages or workspace)")
	}
	var asg ast.AssignmentNode
	found := false
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok || fn.Ident.ID != "main" {
			continue
		}
		var ok2 bool
		asg, ok2 = fn.Body[0].(ast.AssignmentNode)
		if !ok2 {
			t.Fatalf("expected assignment, got %T", fn.Body[0])
		}
		found = true
		break
	}
	if !found {
		t.Fatal("main not found")
	}
	if len(asg.LValues) != 2 {
		t.Fatalf("want 2 LValues, got %d", len(asg.LValues))
	}
	v0, ok := asg.LValues[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("want variable, got %T", asg.LValues[0])
	}
	v1, ok := asg.LValues[1].(ast.VariableNode)
	if !ok {
		t.Fatalf("want variable, got %T", asg.LValues[1])
	}
	tv0, err := tc.LookupInferredType(v0, true)
	if err != nil {
		t.Fatal(err)
	}
	tv1, err := tc.LookupInferredType(v1, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(tv0) != 1 || tv0[0].Ident != ast.TypeInt {
		t.Fatalf("want v: Int, got %v", tv0)
	}
	if len(tv1) != 1 || tv1[0].Ident != ast.TypeError {
		t.Fatalf("want err: Error, got %v", tv1)
	}
}

func TestCheckGoSignature_foldsThreeIntErrorToResultTuple(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tc := New(log, false)
	errIface := types.Universe.Lookup("error").Type()
	r0 := types.NewVar(0, nil, "", types.Typ[types.Int])
	r1 := types.NewVar(0, nil, "", types.Typ[types.Int])
	r2 := types.NewVar(0, nil, "", errIface)
	results := types.NewTuple(r0, r1, r2)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), results, false)
	e := ast.FunctionCallNode{CallSpan: ast.SourceSpan{}}
	got, err := tc.checkGoSignature(sig, "test.F", e, [][]ast.TypeNode{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].IsResultType() {
		t.Fatalf("want single Result, got %v", got)
	}
	succ := got[0].TypeParams[0]
	fail := got[0].TypeParams[1]
	if !succ.IsTupleType() || len(succ.TypeParams) != 2 {
		t.Fatalf("want Tuple(Int,Int), got %s", succ.String())
	}
	if succ.TypeParams[0].Ident != ast.TypeInt || succ.TypeParams[1].Ident != ast.TypeInt {
		t.Fatalf("Tuple elems: %v", succ.TypeParams)
	}
	if fail.Ident != ast.TypeError {
		t.Fatalf("want Error failure, got %s", fail.String())
	}
}

func TestCheckGoSignature_foldsPairIntErrorNoTuple(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tc := New(log, false)
	errIface := types.Universe.Lookup("error").Type()
	r0 := types.NewVar(0, nil, "", types.Typ[types.Int])
	r1 := types.NewVar(0, nil, "", errIface)
	results := types.NewTuple(r0, r1)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), results, false)
	e := ast.FunctionCallNode{CallSpan: ast.SourceSpan{}}
	got, err := tc.checkGoSignature(sig, "test.F", e, [][]ast.TypeNode{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].IsResultType() {
		t.Fatalf("want single Result, got %v", got)
	}
	succ := got[0].TypeParams[0]
	if succ.IsTupleType() {
		t.Fatalf("pair should not use Tuple for success, got %s", succ.String())
	}
	if succ.Ident != ast.TypeInt {
		t.Fatalf("want Int success, got %s", succ.String())
	}
}
