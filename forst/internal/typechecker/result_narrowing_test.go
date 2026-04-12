package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

)

func TestIfResult_isOk_narrowsSuccessType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, Error) {
	return Ok(0)
}

func main() {
	x := f()
	if x is Ok() {
		y := x
		println(y)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	v := ast.VariableNode{Ident: ast.Ident{ID: "y"}}
	types, ok := tc.InferredTypesForVariableNode(v)
	if !ok || len(types) != 1 {
		t.Fatalf("y: ok=%v types=%v", ok, types)
	}
	if types[0].Ident != ast.TypeInt {
		t.Fatalf("expected y narrowed to Int, got %s", types[0].String())
	}
}

func TestIfResult_isOk_hoverShowsIntNotOkSuffix(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, Error) {
	return Ok(1)
}

func main() {
	x := f()
	if x is Ok() {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ifStmt := unwrapIfNode(t, mainFn.Body[1])
	printlnCall, ok := ifStmt.Body[0].(ast.FunctionCallNode)
	if !ok {
		t.Fatalf("expected println call, got %T", ifStmt.Body[0])
	}
	if len(printlnCall.Arguments) != 1 {
		t.Fatalf("println args: %d", len(printlnCall.Arguments))
	}
	vn, ok := printlnCall.Arguments[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable arg, got %T", printlnCall.Arguments[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("x: ok=%v types=%v", ok, types)
	}
	hover := tc.FormatVariableOccurrenceTypeForHover(vn, types)
	if hover != "Int" {
		t.Fatalf("hover: got %q want Int (no .Ok() suffix)", hover)
	}
}

func TestEnsureResult_isOkEmpty_hoverShowsIntNotOkSuffix(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func okInt(): Result(Int, Error) {
	return Ok(42)
}

func main() {
	x := okInt()
	ensure x is Ok()
	println(x)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	printlnCall := unwrapFunctionCall(t, mainFn.Body[2])
	if len(printlnCall.Arguments) != 1 {
		t.Fatalf("println args: %d", len(printlnCall.Arguments))
	}
	vn, ok := printlnCall.Arguments[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable arg, got %T", printlnCall.Arguments[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("x: ok=%v types=%v", ok, types)
	}
	hover := tc.FormatVariableOccurrenceTypeForHover(vn, types)
	if hover != "Int" {
		t.Fatalf("hover: got %q want Int (no .Ok() suffix after ensure)", hover)
	}
}

func TestIfResult_isErr_hoverShowsFailureTypeNotErrSuffix(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, String) {
	return Err("e")
}

func main() {
	x := f()
	if x is Err() {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ifStmt := unwrapIfNode(t, mainFn.Body[1])
	printlnCall, ok := ifStmt.Body[0].(ast.FunctionCallNode)
	if !ok {
		t.Fatalf("expected println call, got %T", ifStmt.Body[0])
	}
	vn, ok := printlnCall.Arguments[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable arg, got %T", printlnCall.Arguments[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("x: ok=%v types=%v", ok, types)
	}
	hover := tc.FormatVariableOccurrenceTypeForHover(vn, types)
	if hover != "String" {
		t.Fatalf("hover: got %q want String (no .Err() suffix)", hover)
	}
}

func unwrapIfNode(t *testing.T, n ast.Node) ast.IfNode {
	t.Helper()
	switch x := n.(type) {
	case ast.IfNode:
		return x
	case *ast.IfNode:
		return *x
	default:
		t.Fatalf("expected IfNode, got %T", n)
		panic("unreachable")
	}
}

func unwrapFunctionCall(t *testing.T, n ast.Node) ast.FunctionCallNode {
	t.Helper()
	switch x := n.(type) {
	case ast.FunctionCallNode:
		return x
	case *ast.FunctionCallNode:
		if x == nil {
			t.Fatal("nil *FunctionCallNode")
		}
		return *x
	default:
		t.Fatalf("expected FunctionCallNode, got %T", n)
		panic("unreachable")
	}
}

func findMainFunction(t *testing.T, nodes []ast.Node) ast.FunctionNode {
	t.Helper()
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok {
			continue
		}
		if fn.Ident.ID == "main" {
			return fn
		}
	}
	t.Fatal("main not found")
	return ast.FunctionNode{}
}

func TestIfResult_isOk_wrongSubject_errors(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func main() {
	x := 1
	if x is Ok() {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: Ok() requires Result subject")
	}
	if !strings.Contains(err.Error(), "Result") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureResult_isOk_validates(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, Error) {
	return Ok(0)
}

func main() {
	x := f()
	ensure x is Ok(0)
	println(x)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestIfResult_thenBranch_okNarrowed_elseBranch_xStillResult(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, Error) {
	return Ok(0)
}

func main() {
	x := f()
	if x is Ok() {
		println(x)
	} else {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ifStmt := unwrapIfNode(t, mainFn.Body[1])
	if ifStmt.Else == nil {
		t.Fatal("expected else branch")
	}
	thenPrint := unwrapFunctionCall(t, ifStmt.Body[0])
	elsePrint := unwrapFunctionCall(t, ifStmt.Else.Body[0])
	checkVarHover(t, tc, thenPrint, ast.TypeInt)
	vElse := elsePrint.Arguments[0].(ast.VariableNode)
	typesElse, ok := tc.InferredTypesForVariableNode(vElse)
	if !ok || len(typesElse) != 1 {
		t.Fatalf("else x: ok=%v types=%v", ok, typesElse)
	}
	if !typesElse[0].IsResultType() {
		t.Fatalf("else branch: want Result widened, got %s", typesElse[0].String())
	}
}

func TestIfResult_elseIf_isErr_narrowsFailureType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, String) {
	return Err("e")
}

func main() {
	x := f()
	if x is Ok() {
		println(x)
	} else if x is Err() {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ifStmt := unwrapIfNode(t, mainFn.Body[1])
	if len(ifStmt.ElseIfs) != 1 {
		t.Fatalf("ElseIfs: %d", len(ifStmt.ElseIfs))
	}
	eiPrint := unwrapFunctionCall(t, ifStmt.ElseIfs[0].Body[0])
	checkVarHover(t, tc, eiPrint, ast.TypeString)
}

func TestIfResult_isOk_withLiteralArg_narrowsSuccessType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, Error) {
	return Ok(0)
}

func main() {
	x := f()
	if x is Ok(42) {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ifStmt := unwrapIfNode(t, mainFn.Body[1])
	printlnCall := unwrapFunctionCall(t, ifStmt.Body[0])
	checkVarHover(t, tc, printlnCall, ast.TypeInt)
}

func TestIfResult_isOk_incompatibleLiteralArg_errors(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, Error) {
	return Ok(0)
}

func main() {
	x := f()
	if x is Ok("nope") {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: Ok literal incompatible with Int")
	}
	if !strings.Contains(err.Error(), "incompatible") && !strings.Contains(err.Error(), "Ok") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIfResult_isErr_wrongSubject_errors(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func main() {
	x := 1
	if x is Err() {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: Err() requires Result subject")
	}
	if !strings.Contains(err.Error(), "Result") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureResult_blockBody_seesOkNarrowing(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func okInt(): Result(Int, Error) {
	return Ok(42)
}

func main() {
	x := okInt()
	ensure x is Ok() {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ensureStmt := mainFn.Body[1].(ast.EnsureNode)
	if ensureStmt.Block == nil {
		t.Fatal("expected ensure block")
	}
	printlnCall := unwrapFunctionCall(t, ensureStmt.Block.Body[0])
	checkVarHover(t, tc, printlnCall, ast.TypeInt)
}

func TestEnsureResult_isErr_narrowsFailureInSuccessor(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func bad(): Result(Int, String) {
	return Err("e")
}

func main() {
	x := bad()
	ensure x is Err()
	println(x)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	printlnCall := unwrapFunctionCall(t, mainFn.Body[2])
	checkVarHover(t, tc, printlnCall, ast.TypeString)
}

// fieldPathResultFixture defines Wrap { r: Result(Int, Error) } and helpers; shape literals cannot use
// Ok(...) in-field (parseValue), so we bind x := okInt() then w := { r: x }.
const fieldPathResultSourcePrefix = `package main

type Wrap = {
	r: Result(Int, Error),
}

func okInt(): Result(Int, Error) {
	return Ok(42)
}
`

func TestIfResult_fieldPath_isOk_narrowsSuccessType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := fieldPathResultSourcePrefix + `
func main() {
	x := okInt()
	w := { r: x }
	if w.r is Ok() {
		println(w.r)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ifStmt := unwrapIfNode(t, mainFn.Body[2])
	printlnCall := unwrapFunctionCall(t, ifStmt.Body[0])
	vn := printlnCall.Arguments[0].(ast.VariableNode)
	if vn.Ident.ID != "w.r" {
		t.Fatalf("want println subject w.r, got %q", vn.Ident.ID)
	}
	checkVarHover(t, tc, printlnCall, ast.TypeInt)
}

func TestIfResult_fieldPath_isErr_narrowsFailureType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type WrapStr = {
	r: Result(Int, String),
}

func bad(): Result(Int, String) {
	return Err("e")
}

func main() {
	x := bad()
	w := { r: x }
	if w.r is Err() {
		println(w.r)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ifStmt := unwrapIfNode(t, mainFn.Body[2])
	printlnCall := unwrapFunctionCall(t, ifStmt.Body[0])
	vn := printlnCall.Arguments[0].(ast.VariableNode)
	if vn.Ident.ID != "w.r" {
		t.Fatalf("want println subject w.r, got %q", vn.Ident.ID)
	}
	checkVarHover(t, tc, printlnCall, ast.TypeString)
}

func TestIfResult_fieldPath_thenOk_elseBranch_fieldStaysResult(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := fieldPathResultSourcePrefix + `
func main() {
	x := okInt()
	w := { r: x }
	if w.r is Ok() {
		println(w.r)
	} else {
		println(w.r)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ifStmt := unwrapIfNode(t, mainFn.Body[2])
	if ifStmt.Else == nil {
		t.Fatal("expected else branch")
	}
	thenCall := unwrapFunctionCall(t, ifStmt.Body[0])
	checkVarHover(t, tc, thenCall, ast.TypeInt)

	elseCall := unwrapFunctionCall(t, ifStmt.Else.Body[0])
	vElse := elseCall.Arguments[0].(ast.VariableNode)
	if vElse.Ident.ID != "w.r" {
		t.Fatalf("else: want w.r, got %q", vElse.Ident.ID)
	}
	typesElse, ok := tc.InferredTypesForVariableNode(vElse)
	if !ok || len(typesElse) != 1 {
		t.Fatalf("else w.r: ok=%v types=%v", ok, typesElse)
	}
	if !typesElse[0].IsResultType() {
		t.Fatalf("else: want Result, got %s", typesElse[0].String())
	}
}

func TestEnsureResult_fieldPath_isOk_narrowsSuccessor(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := fieldPathResultSourcePrefix + `
func main() {
	x := okInt()
	w := { r: x }
	ensure w.r is Ok()
	println(w.r)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	printlnCall := unwrapFunctionCall(t, mainFn.Body[3])
	vn := printlnCall.Arguments[0].(ast.VariableNode)
	if vn.Ident.ID != "w.r" {
		t.Fatalf("want println subject w.r, got %q", vn.Ident.ID)
	}
	checkVarHover(t, tc, printlnCall, ast.TypeInt)
}

func TestEnsureResult_fieldPath_blockBody_seesOkNarrowing(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := fieldPathResultSourcePrefix + `
func main() {
	x := okInt()
	w := { r: x }
	ensure w.r is Ok() {
		println(w.r)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	ensureStmt := mainFn.Body[2].(ast.EnsureNode)
	if ensureStmt.Block == nil {
		t.Fatal("expected ensure block")
	}
	if ensureStmt.Variable.Ident.ID != "w.r" {
		t.Fatalf("ensure subject: got %q", ensureStmt.Variable.Ident.ID)
	}
	printlnCall := unwrapFunctionCall(t, ensureStmt.Block.Body[0])
	checkVarHover(t, tc, printlnCall, ast.TypeInt)
}

func TestEnsureResult_fieldPath_isErr_narrowsFailureSuccessor(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type WrapStr = {
	r: Result(Int, String),
}

func bad(): Result(Int, String) {
	return Err("e")
}

func main() {
	x := bad()
	w := { r: x }
	ensure w.r is Err()
	println(w.r)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	mainFn := findMainFunction(t, nodes)
	printlnCall := unwrapFunctionCall(t, mainFn.Body[3])
	vn := printlnCall.Arguments[0].(ast.VariableNode)
	if vn.Ident.ID != "w.r" {
		t.Fatalf("want println subject w.r, got %q", vn.Ident.ID)
	}
	checkVarHover(t, tc, printlnCall, ast.TypeString)
}

// checkVarHover asserts the first println argument is a variable with the expected builtin type (Ident + hover display).
func checkVarHover(t *testing.T, tc *TypeChecker, printlnCall ast.FunctionCallNode, wantIdent ast.TypeIdent) {
	t.Helper()
	if len(printlnCall.Arguments) != 1 {
		t.Fatalf("println args: %d", len(printlnCall.Arguments))
	}
	vn, ok := printlnCall.Arguments[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable arg, got %T", printlnCall.Arguments[0])
	}
	types, ok2 := tc.InferredTypesForVariableNode(vn)
	if !ok2 || len(types) != 1 {
		t.Fatalf("variable: ok=%v types=%v", ok2, types)
	}
	if types[0].Ident != wantIdent {
		t.Fatalf("want ident %v, got %s", wantIdent, types[0].String())
	}
	wantHover := tc.FormatTypeNodeDisplay(ast.TypeNode{Ident: wantIdent})
	hover := tc.FormatVariableOccurrenceTypeForHover(vn, types)
	if hover != wantHover {
		t.Fatalf("hover: got %q want %q", hover, wantHover)
	}
}
