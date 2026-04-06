package typechecker

import (
	"testing"

	"forst/internal/ast"
)

// TestIfMergePoint_xWidensToOuterTypeAfterIfChain verifies plan §3.2 / §0.3: after a completed
// if x is … chain, a use of x has the enclosing (pre-if) type, not only the branch refinement.
func TestIfMergePoint_xWidensToOuterTypeAfterIfChain(t *testing.T) {
	t.Parallel()
	myStr := ast.TypeIdent("MyStr")
	str := ast.TypeString
	typeDef := ast.TypeDefNode{
		Ident: myStr,
		Expr: &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &str,
			},
		},
	}

	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	explicitMyStr := ast.TypeNode{Ident: myStr}
	spanCondX := ast.SourceSpan{StartLine: 2, StartCol: 22, EndLine: 2, EndCol: 23}
	spanBodyX := ast.SourceSpan{StartLine: 2, StartCol: 40, EndLine: 2, EndCol: 41}
	spanAfterIfX := ast.SourceSpan{StartLine: 4, StartCol: 10, EndLine: 4, EndCol: 11}

	fn := ast.FunctionNode{
		Ident:       ast.Ident{ID: "f"},
		ReturnTypes: []ast.TypeNode{{Ident: myStr}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.VariableNode{{
					Ident:        ast.Ident{ID: "x", Span: spanDeclX},
					ExplicitType: explicitMyStr,
				}},
				RValues:       []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				ExplicitTypes: []*ast.TypeNode{&explicitMyStr},
				IsShort:       true,
			},
			ast.IfNode{
				Condition: ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanCondX}},
					Operator: ast.TokenIs,
					Right: ast.AssertionNode{
						BaseType: &str,
					},
				},
				Body: []ast.Node{
					ast.ReturnNode{
						Values: []ast.ExpressionNode{
							ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanBodyX}},
						},
					},
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{
					ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanAfterIfX}},
				},
			},
		},
	}

	tc := New(discardLogger(), false)
	if err := tc.CheckTypes([]ast.Node{typeDef, fn}); err != nil {
		t.Fatal(err)
	}

	ifn := fn.Body[1].(ast.IfNode)
	retInBranch := ifn.Body[0].(ast.ReturnNode)
	vInBranch, ok := retInBranch.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("branch return: expected variable")
	}
	typesInBranch, ok := tc.InferredTypesForVariableNode(vInBranch)
	if !ok || len(typesInBranch) != 1 || typesInBranch[0].Ident != ast.TypeString {
		t.Fatalf("inside branch want refined String, got %+v ok=%v", typesInBranch, ok)
	}

	retAfter := fn.Body[2].(ast.ReturnNode)
	vAfter, ok := retAfter.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("after if: expected variable")
	}
	typesAfter, ok := tc.InferredTypesForVariableNode(vAfter)
	if !ok || len(typesAfter) != 1 {
		t.Fatalf("after if: expected type for x, got ok=%v len=%d", ok, len(typesAfter))
	}
	if typesAfter[0].Ident != myStr {
		t.Fatalf("after if merge want outer type %s, got %s", myStr, typesAfter[0].Ident)
	}
}

func TestJoinAfterIfMerge_returnsOuter(t *testing.T) {
	t.Parallel()
	outer := ast.TypeNode{Ident: ast.TypeIdent("MyStr")}
	refined := []ast.TypeNode{{Ident: ast.TypeString}}
	got := JoinAfterIfMerge(outer, refined)
	if got.Ident != outer.Ident {
		t.Fatalf("JoinAfterIfMerge: got %s want %s", got.Ident, outer.Ident)
	}
}

func TestIfMergePoint_tableElseIfLadder(t *testing.T) {
	baseStr := ast.TypeString
	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	spanCond2 := ast.SourceSpan{StartLine: 1, StartCol: 40, EndLine: 1, EndCol: 41}
	spanBody2 := ast.SourceSpan{StartLine: 1, StartCol: 55, EndLine: 1, EndCol: 56}
	spanAfter := ast.SourceSpan{StartLine: 1, StartCol: 70, EndLine: 1, EndCol: 71}

	cases := []struct {
		name string
		fn   ast.FunctionNode
	}{
		{
			name: "else_if_then_merge",
			fn: ast.FunctionNode{
				Ident:       ast.Ident{ID: "f"},
				ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}},
				Body: []ast.Node{
					ast.AssignmentNode{
						LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "x", Span: spanDeclX}}},
						RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
						IsShort: true,
					},
					ast.IfNode{
						Condition: ast.BoolLiteralNode{Value: false},
						Body: []ast.Node{
							ast.ReturnNode{Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: "a"}}},
						},
						ElseIfs: []ast.ElseIfNode{{
							Condition: ast.BinaryExpressionNode{
								Left:     ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanCond2}},
								Operator: ast.TokenIs,
								Right:    ast.AssertionNode{BaseType: &baseStr},
							},
							Body: []ast.Node{
								ast.ReturnNode{Values: []ast.ExpressionNode{
									ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanBody2}},
								}},
							},
						}},
						Else: &ast.ElseBlockNode{
							Body: []ast.Node{
								ast.ReturnNode{Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: ""}}},
							},
						},
					},
					ast.ReturnNode{Values: []ast.ExpressionNode{
						ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanAfter}},
					}},
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			checker := New(discardLogger(), false)
			if err := checker.CheckTypes([]ast.Node{c.fn}); err != nil {
				t.Fatal(err)
			}
			ret := c.fn.Body[2].(ast.ReturnNode)
			vn := ret.Values[0].(ast.VariableNode)
			ty, ok := checker.InferredTypesForVariableNode(vn)
			if !ok || len(ty) != 1 || ty[0].Ident != ast.TypeString {
				t.Fatalf("post-chain x: want String (outer), got %+v ok=%v", ty, ok)
			}
		})
	}
}

// TestIfMergePoint_nestedInnerIfEmptyBody_xWidensInsideOuterIfBody verifies that merge after an
// inner if-chain runs before the continuation of an enclosing if body: x must widen to the
// outer (pre-outer-if) binding type for uses after the inner chain.
func TestIfMergePoint_nestedInnerIfEmptyBody_xWidensInsideOuterIfBody(t *testing.T) {
	t.Parallel()
	myStr := ast.TypeIdent("MyStr")
	str := ast.TypeString
	typeDef := ast.TypeDefNode{
		Ident: myStr,
		Expr: &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{BaseType: &str},
		},
	}
	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	explicitMyStr := ast.TypeNode{Ident: myStr}
	spanCondInner := ast.SourceSpan{StartLine: 2, StartCol: 22, EndLine: 2, EndCol: 23}
	spanAfterInner := ast.SourceSpan{StartLine: 4, StartCol: 12, EndLine: 4, EndCol: 13}
	spanFinal := ast.SourceSpan{StartLine: 6, StartCol: 10, EndLine: 6, EndCol: 11}

	innerIf := ast.IfNode{
		Condition: ast.BinaryExpressionNode{
			Left:     ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanCondInner}},
			Operator: ast.TokenIs,
			Right:    ast.AssertionNode{BaseType: &str},
		},
		Body: []ast.Node{}, // narrowing is still recorded; inner endIfChainApplyJoin still runs
	}

	fn := ast.FunctionNode{
		Ident:       ast.Ident{ID: "f"},
		ReturnTypes: []ast.TypeNode{{Ident: myStr}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.VariableNode{{
					Ident:        ast.Ident{ID: "x", Span: spanDeclX},
					ExplicitType: explicitMyStr,
				}},
				RValues:       []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				ExplicitTypes: []*ast.TypeNode{&explicitMyStr},
				IsShort:       true,
			},
			ast.IfNode{
				Condition: ast.BoolLiteralNode{Value: true},
				Body: []ast.Node{
					innerIf,
					ast.ReturnNode{
						Values: []ast.ExpressionNode{
							ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanAfterInner}},
						},
					},
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{
					ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanFinal}},
				},
			},
		},
	}

	tc := New(discardLogger(), false)
	if err := tc.CheckTypes([]ast.Node{typeDef, fn}); err != nil {
		t.Fatal(err)
	}

	outerIf := fn.Body[1].(ast.IfNode)
	retAfterInner := outerIf.Body[1].(ast.ReturnNode)
	vAfter, ok := retAfterInner.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatal("expected variable after inner if")
	}
	ty, ok := tc.InferredTypesForVariableNode(vAfter)
	if !ok || len(ty) != 1 || ty[0].Ident != myStr {
		t.Fatalf("after inner merge inside outer body want outer type %s, got %+v ok=%v", myStr, ty, ok)
	}
}

// TestIfMergePoint_inferredString_outerUnchangedAfterIfChain uses an inferred String binding (no
// alias) so post-merge type stays String — same widening policy, simpler surface.
func TestIfMergePoint_inferredString_outerUnchangedAfterIfChain(t *testing.T) {
	t.Parallel()
	str := ast.TypeString
	spanDecl := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	spanCond := ast.SourceSpan{StartLine: 2, StartCol: 22, EndLine: 2, EndCol: 23}
	spanInBranch := ast.SourceSpan{StartLine: 2, StartCol: 40, EndLine: 2, EndCol: 41}
	spanAfter := ast.SourceSpan{StartLine: 4, StartCol: 10, EndLine: 4, EndCol: 11}

	fn := ast.FunctionNode{
		Ident:       ast.Ident{ID: "f"},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "x", Span: spanDecl}}},
				RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				IsShort: true,
			},
			ast.IfNode{
				Condition: ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanCond}},
					Operator: ast.TokenIs,
					Right:    ast.AssertionNode{BaseType: &str},
				},
				Body: []ast.Node{
					ast.ReturnNode{
						Values: []ast.ExpressionNode{
							ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanInBranch}},
						},
					},
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{
					ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanAfter}},
				},
			},
		},
	}

	tc := New(discardLogger(), false)
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}

	ifn := fn.Body[1].(ast.IfNode)
	retInBranch := ifn.Body[0].(ast.ReturnNode)
	vInBranch := retInBranch.Values[0].(ast.VariableNode)
	branchTy, ok := tc.InferredTypesForVariableNode(vInBranch)
	if !ok || len(branchTy) != 1 || branchTy[0].Ident != ast.TypeString {
		t.Fatalf("branch want String, got %+v", branchTy)
	}

	retAfter := fn.Body[2].(ast.ReturnNode)
	vAfter := retAfter.Values[0].(ast.VariableNode)
	afterTy, ok := tc.InferredTypesForVariableNode(vAfter)
	if !ok || len(afterTy) != 1 || afterTy[0].Ident != ast.TypeString {
		t.Fatalf("after if want outer String, got %+v ok=%v", afterTy, ok)
	}
}
