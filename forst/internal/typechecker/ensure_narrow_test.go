package typechecker

import (
	"io"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestApplyEnsureSuccessorNarrowing_minConstraintRegistersSymbol(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.log.SetOutput(io.Discard)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("x"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)
	baseStr := ast.TypeString
	sp := ast.SourceSpan{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 2}
	ensure := ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "x", Span: sp}},
		Assertion: ast.AssertionNode{
			BaseType: &baseStr,
			Constraints: []ast.ConstraintNode{
				{Name: "Min", Args: []ast.ConstraintArgumentNode{{Value: ptrVal(ast.IntLiteralNode{Value: 1})}}},
			},
		},
	}
	tc.applyEnsureSuccessorNarrowing(ensure)
	sym, ok := tc.CurrentScope().LookupVariable(ast.Identifier("x"))
	if !ok || sym.NarrowingPredicateDisplay != "Min(1)" {
		t.Fatalf("symbol: ok=%v display=%q guards=%v", ok, sym.NarrowingPredicateDisplay, sym.NarrowingTypeGuards)
	}
	if _, err := tc.inferExpressionType(ensure.Variable); err != nil {
		t.Fatal(err)
	}
	k := variableOccurrenceKey{ident: ast.Identifier("x"), span: sp}
	if tc.variableOccurrenceNarrowingPredicateDisplay[k] != "Min(1)" {
		t.Fatalf("occurrence display: %#v", tc.variableOccurrenceNarrowingPredicateDisplay)
	}
}

func TestEnsureSuccessorNarrowing_blockBodySeesRefinedType(t *testing.T) {
	baseStr := ast.TypeString
	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	spanEnsureX := ast.SourceSpan{StartLine: 1, StartCol: 30, EndLine: 1, EndCol: 31}
	spanBodyX := ast.SourceSpan{StartLine: 2, StartCol: 10, EndLine: 2, EndCol: 11}

	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "f"},
		// Functions that contain ensure are inferred as (T, Error).
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeError}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanDeclX}}},
				RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				IsShort: true,
			},
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanEnsureX}},
				Assertion: ast.AssertionNode{
					BaseType: &baseStr,
				},
				Block: &ast.EnsureBlockNode{
					Body: []ast.Node{
						ast.ReturnNode{
							Values: []ast.ExpressionNode{
								ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanBodyX}},
								ast.NilLiteralNode{},
							},
						},
					},
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{
					ast.StringLiteralNode{Value: ""},
					ast.NilLiteralNode{},
				},
			},
		},
	}

	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}

	ensureStmt := fn.Body[1].(ast.EnsureNode)
	ret := ensureStmt.Block.Body[0].(ast.ReturnNode)
	vn, ok := ret.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable in return, got %T", ret.Values[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("expected narrowed type for x in ensure block, got ok=%v len=%d", ok, len(types))
	}
	if types[0].Ident != ast.TypeString {
		t.Fatalf("expected String inside block, got %s", types[0].Ident)
	}
}

func TestEnsureSuccessorNarrowing_followingStatementsSeeRefinedTypeWithoutBlock(t *testing.T) {
	baseStr := ast.TypeString
	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	// Match spans from TestEnsureSuccessorNarrowing_blockBodySeesRefinedType for the ensure subject.
	spanEnsureX := ast.SourceSpan{StartLine: 1, StartCol: 30, EndLine: 1, EndCol: 31}
	spanReturnX := ast.SourceSpan{StartLine: 1, StartCol: 40, EndLine: 1, EndCol: 41}

	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "f"},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeError}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanDeclX}}},
				RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				IsShort: true,
			},
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanEnsureX}},
				Assertion: ast.AssertionNode{
					BaseType: &baseStr,
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{
					ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanReturnX}},
					ast.NilLiteralNode{},
				},
			},
		},
	}

	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}

	ret := fn.Body[2].(ast.ReturnNode)
	vn, ok := ret.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable in return, got %T", ret.Values[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("expected narrowed type for x after ensure, got ok=%v len=%d", ok, len(types))
	}
	if types[0].Ident != ast.TypeString {
		t.Fatalf("expected String after ensure, got %s", types[0].Ident)
	}
}

// TestEnsure_successiveSubjects_predicateDisplayReflectsPriorEnsuresOnly verifies each ensure
// subject is stamped before this line's successor narrowing: first subject has no prior guards;
// second subject reflects only the first ensure, not both.
func TestEnsure_successiveSubjects_predicateDisplayReflectsPriorEnsuresOnly(t *testing.T) {
	t.Parallel()
	baseStr := ast.TypeString
	spanX1 := ast.SourceSpan{StartLine: 10, StartCol: 10, EndLine: 10, EndCol: 11}
	spanX2 := ast.SourceSpan{StartLine: 11, StartCol: 10, EndLine: 11, EndCol: 11}

	tg := ast.TypeGuardNode{
		Ident: "G",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeString},
		},
		Body: []ast.Node{
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanX1}},
				Assertion: ast.AssertionNode{
					BaseType: &baseStr,
					Constraints: []ast.ConstraintNode{
						{Name: "Min", Args: []ast.ConstraintArgumentNode{{Value: ptrVal(ast.IntLiteralNode{Value: 1})}}},
					},
				},
			},
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanX2}},
				Assertion: ast.AssertionNode{
					BaseType: &baseStr,
					Constraints: []ast.ConstraintNode{
						{Name: "Max", Args: []ast.ConstraintArgumentNode{{Value: ptrVal(ast.IntLiteralNode{Value: 10})}}},
					},
				},
			},
		},
	}

	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{tg}); err != nil {
		t.Fatal(err)
	}

	v1 := ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanX1}}
	types1, ok := tc.InferredTypesForVariableNode(v1)
	if !ok || len(types1) != 1 {
		t.Fatalf("subject 1: ok=%v types=%v", ok, types1)
	}
	if got := tc.NarrowingPredicateDisplayForVariableOccurrence(v1); got != "" {
		t.Fatalf("subject 1 display: got %q want empty (no prior ensures)", got)
	}
	if hover1 := tc.FormatVariableOccurrenceTypeForHover(v1, types1); hover1 != "String" {
		t.Fatalf("subject 1 hover: got %q want String", hover1)
	}

	v2 := ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanX2}}
	types2, ok := tc.InferredTypesForVariableNode(v2)
	if !ok || len(types2) != 1 {
		t.Fatalf("subject 2: ok=%v types=%v", ok, types2)
	}
	if got := tc.NarrowingPredicateDisplayForVariableOccurrence(v2); got != "Min(1)" {
		t.Fatalf("subject 2 display: got %q want Min(1)", got)
	}
	if hover := tc.FormatVariableOccurrenceTypeForHover(v2, types2); hover != "String.Min(1)" {
		t.Fatalf("subject 2 hover: got %q want String.Min(1)", hover)
	}
}

// TestEnsure_followingStatementUsesMergedPredicateDisplayForHover checks `x` after two ensures in a function.
func TestEnsure_followingStatementUsesMergedPredicateDisplayForHover(t *testing.T) {
	t.Parallel()
	baseStr := ast.TypeString
	spanDecl := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	spanX1 := ast.SourceSpan{StartLine: 2, StartCol: 10, EndLine: 2, EndCol: 11}
	spanX2 := ast.SourceSpan{StartLine: 3, StartCol: 10, EndLine: 3, EndCol: 11}
	spanRet := ast.SourceSpan{StartLine: 4, StartCol: 10, EndLine: 4, EndCol: 11}

	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "f"},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeError}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanDecl}}},
				RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				IsShort: true,
			},
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanX1}},
				Assertion: ast.AssertionNode{
					BaseType: &baseStr,
					Constraints: []ast.ConstraintNode{
						{Name: "Min", Args: []ast.ConstraintArgumentNode{{Value: ptrVal(ast.IntLiteralNode{Value: 1})}}},
					},
				},
			},
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanX2}},
				Assertion: ast.AssertionNode{
					BaseType: &baseStr,
					Constraints: []ast.ConstraintNode{
						{Name: "Max", Args: []ast.ConstraintArgumentNode{{Value: ptrVal(ast.IntLiteralNode{Value: 10})}}},
					},
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{
					ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanRet}},
					ast.NilLiteralNode{},
				},
			},
		},
	}

	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}

	vRet := ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanRet}}
	types, ok := tc.InferredTypesForVariableNode(vRet)
	if !ok || len(types) != 1 {
		t.Fatalf("return x: ok=%v types=%v", ok, types)
	}
	if got := tc.NarrowingPredicateDisplayForVariableOccurrence(vRet); got != "Min(1).Max(10)" {
		t.Fatalf("return x display: got %q want Min(1).Max(10)", got)
	}
	if hover := tc.FormatVariableOccurrenceTypeForHover(vRet, types); hover != "String.Min(1).Max(10)" {
		t.Fatalf("return x hover: got %q", hover)
	}
}

// TestEnsure_fieldPathSubjectSuccessorNarrowingPredicateHover checks that after `ensure g.cells is Min(9)`,
// a later `ensure g.cells is Max(9)` sees hover `Array(String).Min(9)` on the second line's subject.
func TestEnsure_fieldPathSubjectSuccessorNarrowingPredicateHover(t *testing.T) {
	t.Parallel()
	spanCells1 := ast.SourceSpan{StartLine: 10, StartCol: 10, EndLine: 10, EndCol: 17}
	spanCells2 := ast.SourceSpan{StartLine: 11, StartCol: 10, EndLine: 11, EndCol: 17}

	gameStateDef := ast.TypeDefNode{
		Ident: "GameState",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"cells": {
						Type: &ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}},
					},
				},
			},
		},
	}

	tg := ast.TypeGuardNode{
		Ident: "ValidBoard",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "g"},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("GameState")},
		},
		Body: []ast.Node{
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "g.cells", Span: spanCells1}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{Name: "Min", Args: []ast.ConstraintArgumentNode{{Value: ptrVal(ast.IntLiteralNode{Value: 9, Type: ast.TypeNode{Ident: ast.TypeInt}})}}},
					},
				},
			},
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "g.cells", Span: spanCells2}},
				Assertion: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{Name: "Max", Args: []ast.ConstraintArgumentNode{{Value: ptrVal(ast.IntLiteralNode{Value: 9, Type: ast.TypeNode{Ident: ast.TypeInt}})}}},
					},
				},
			},
		},
	}

	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{gameStateDef, tg}); err != nil {
		t.Fatal(err)
	}

	v2 := ast.VariableNode{Ident: ast.Ident{ID: "g.cells", Span: spanCells2}}
	types2, ok := tc.InferredTypesForVariableNode(v2)
	if !ok || len(types2) != 1 {
		t.Fatalf("subject 2: ok=%v types=%v", ok, types2)
	}
	if got := tc.NarrowingPredicateDisplayForVariableOccurrence(v2); got != "Min(9)" {
		t.Fatalf("subject 2 display: got %q want Min(9)", got)
	}
	if hover := tc.FormatVariableOccurrenceTypeForHover(v2, types2); hover != "Array(String).Min(9)" {
		t.Fatalf("subject 2 hover: got %q want Array(String).Min(9)", hover)
	}
}

// TestEnsure_compoundParamFieldPath_typeGuardPredicateSurvivesDifferentSpan checks that after
// ensure req.state is ValidBoard(), hover formatting for req.state still sees ValidBoard() when the
// occurrence span differs from the ensure subject (e.g. a later if line).
func TestEnsure_compoundParamFieldPath_typeGuardPredicateSurvivesDifferentSpan(t *testing.T) {
	t.Parallel()
	const src = `package main

type GameState = {
	cells: []String,
	status: String,
}

is (g GameState) ValidBoard() {
	ensure g.cells is Min(9)
	ensure g.cells is Max(9)
}

type MoveRequest = {
	state: GameState,
	row:   Int,
	col:   Int,
}

func ApplyMove(req MoveRequest): Result(Int, Error) {
	ensure req.state is ValidBoard()
	if req.state.status != "playing" {
		return 0
	}
	return 1
}
`
	log := logrus.New()
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(logrus.New(), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	info, ok := tc.compoundNarrowingByIdentifier[ast.Identifier("req.state")]
	if !ok || info.disp == "" {
		t.Fatalf("expected compound narrowing for req.state, got ok=%v disp=%q", ok, info.disp)
	}
	if !strings.Contains(info.disp, "ValidBoard") {
		t.Fatalf("compound narrowing display should include ValidBoard; got %q", info.disp)
	}
	// Different span than ensure subject (line 21 vs line 22): predicate must still resolve.
	vIf := ast.VariableNode{Ident: ast.Ident{
		ID: ast.Identifier("req.state"),
		Span: ast.SourceSpan{StartLine: 22, StartCol: 5, EndLine: 22, EndCol: 14},
	}}
	types, have := tc.InferredTypesForVariableNode(vIf)
	if !have || len(types) != 1 {
		t.Fatalf("types for if-line req.state: have=%v types=%v", have, types)
	}
	if got := tc.FormatVariableOccurrenceTypeForHover(vIf, types); !strings.Contains(got, "ValidBoard") {
		t.Fatalf("hover format: got %q", got)
	}
}
