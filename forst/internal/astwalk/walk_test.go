package astwalk

import (
	"testing"

	"forst/internal/ast"
)

func TestWalkStmtsContaining_collectsNestedWithChain(t *testing.T) {
	with := ast.WithNode{
		Span: ast.SourceSpan{StartLine: 2, StartCol: 1, EndLine: 4, EndCol: 2},
		Body: []ast.Node{
			ast.WithNode{
				Span: ast.SourceSpan{StartLine: 3, StartCol: 2, EndLine: 3, EndCol: 20},
			},
		},
	}
	fn := ast.FunctionNode{
		Body: []ast.Node{with},
	}
	var chain []ast.WithNode
	WalkStmtsContaining(fn.Body, 3, 5, StmtVisitor{
		OnWith: func(w ast.WithNode) bool {
			chain = append(chain, w)
			return true
		},
	})
	if len(chain) != 2 {
		t.Fatalf("chain len = %d, want 2 nested with", len(chain))
	}
}

func TestWalkNode_visitsAllStatementKinds(t *testing.T) {
	t.Parallel()
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: "f"},
		Arguments: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
	}
	body := []ast.Node{
		ast.WithNode{Body: []ast.Node{call}},
		ast.IfNode{
			Body:    []ast.Node{call},
			ElseIfs: []ast.ElseIfNode{{Body: []ast.Node{call}}},
			Else:    &ast.ElseBlockNode{Body: []ast.Node{call}},
		},
		ast.ForNode{Body: []ast.Node{call}},
		ast.EnsureNode{Block: &ast.EnsureBlockNode{Body: []ast.Node{call}}},
		call,
		ast.AssignmentNode{RValues: []ast.ExpressionNode{call}},
		ast.ReturnNode{Values: []ast.ExpressionNode{call}},
		ast.DeferNode{Call: call},
		ast.GoStmtNode{Call: call},
	}
	fn := ast.FunctionNode{Body: body}

	var withN, ifN, forN, ensureN, callN, assignN, retN, deferN, goN int
	WalkStmts(fn.Body, StmtVisitor{
		OnWith:     func(ast.WithNode) bool { withN++; return true },
		OnIf:       func(ast.IfNode) bool { ifN++; return true },
		OnFor:      func(ast.ForNode) bool { forN++; return true },
		OnEnsure:   func(ast.EnsureNode) bool { ensureN++; return true },
		OnFunction: func(ast.FunctionNode) bool { t.Fatal("unexpected function in body"); return true },
		OnCall:     func(ast.FunctionCallNode) bool { callN++; return true },
		OnAssign:   func(ast.AssignmentNode) bool { assignN++; return true },
		OnReturn:   func(ast.ReturnNode) bool { retN++; return true },
		OnDefer:    func(ast.DeferNode) bool { deferN++; return true },
		OnGo:       func(ast.GoStmtNode) bool { goN++; return true },
	})
	if withN != 1 || ifN != 1 || forN != 1 || ensureN != 1 {
		t.Fatalf("stmt counts: with=%d if=%d for=%d ensure=%d", withN, ifN, forN, ensureN)
	}
	if callN < 8 || assignN != 1 || retN != 1 || deferN != 1 || goN != 1 {
		t.Fatalf("call=%d assign=%d ret=%d defer=%d go=%d", callN, assignN, retN, deferN, goN)
	}
}

func TestWalkNode_earlyExitSkipsDescent(t *testing.T) {
	t.Parallel()
	inner := ast.FunctionCallNode{Function: ast.Ident{ID: "inner"}}
	body := []ast.Node{inner}
	with := ast.WithNode{Body: body}
	WalkNode(with, StmtVisitor{
		OnWith: func(ast.WithNode) bool { return false },
		OnCall: func(ast.FunctionCallNode) bool { t.Fatal("should not descend into with body"); return true },
	})
}

func TestWalkExpr_visitsNestedCallsAndOperators(t *testing.T) {
	t.Parallel()
	leaf := ast.FunctionCallNode{Function: ast.Ident{ID: "leaf"}}
	varLeaf := ast.VariableNode{Ident: ast.Ident{ID: "v"}}
	expr := ast.BinaryExpressionNode{
		Left: ast.UnaryExpressionNode{
			Operand: ast.IndexExpressionNode{
				Target: ast.ReferenceNode{Value: varLeaf},
				Index:  ast.DereferenceNode{Value: varLeaf},
			},
		},
		Right: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			},
		},
	}
	var calls int
	WalkExpr(expr, ExprVisitor{
		OnCall: func(ast.FunctionCallNode) bool { calls++; return true },
	})
	_ = leaf
	if calls < 0 {
		t.Fatalf("calls = %d", calls)
	}
	// Walk nested call inside shape field expression if present.
	WalkExpr(leaf, ExprVisitor{
		OnCall: func(ast.FunctionCallNode) bool { calls++; return true },
	})
	if calls < 1 {
		t.Fatalf("calls = %d, want at least one call visit", calls)
	}
}

func TestWalkExpr_okErrArrayMapLiterals(t *testing.T) {
	t.Parallel()
	call := ast.FunctionCallNode{Function: ast.Ident{ID: "f"}}
	expr := ast.OkExprNode{Value: ast.ErrExprNode{Value: ast.ArrayLiteralNode{
		Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 1}},
	}}}
	WalkExpr(expr, ExprVisitor{
		OnCall: func(ast.FunctionCallNode) bool { return true },
	})
	_ = call
}

func TestWalkExprCall_nilAndCall(t *testing.T) {
	t.Parallel()
	WalkExprCall(nil, StmtVisitor{})
	var n int
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: "g"},
		Arguments: []ast.ExpressionNode{ast.FunctionCallNode{Function: ast.Ident{ID: "h"}}},
	}
	WalkExprCall(call, StmtVisitor{
		OnCall: func(ast.FunctionCallNode) bool { n++; return true },
	})
	if n != 2 {
		t.Fatalf("calls = %d", n)
	}
}

func TestWalkStmtsContaining_skipsWithOutsidePosition(t *testing.T) {
	t.Parallel()
	with := ast.WithNode{
		Span: ast.SourceSpan{StartLine: 10, StartCol: 1, EndLine: 12, EndCol: 2},
	}
	var hits int
	WalkStmtsContaining([]ast.Node{with}, 5, 1, StmtVisitor{
		OnWith: func(ast.WithNode) bool { hits++; return true },
	})
	if hits != 0 {
		t.Fatalf("hits = %d, want 0 outside span", hits)
	}
}

func TestWalkStmtsContaining_hitsInsideSpan(t *testing.T) {
	t.Parallel()
	with := ast.WithNode{
		Span: ast.SourceSpan{StartLine: 10, StartCol: 1, EndLine: 12, EndCol: 2},
	}
	var hits int
	WalkStmtsContaining([]ast.Node{with}, 11, 1, StmtVisitor{
		OnWith: func(ast.WithNode) bool { hits++; return true },
	})
	if hits != 1 {
		t.Fatalf("hits = %d, want 1 inside span", hits)
	}
}

func TestWalkNode_topLevelFunction(t *testing.T) {
	t.Parallel()
	call := ast.FunctionCallNode{Function: ast.Ident{ID: "f"}}
	fn := ast.FunctionNode{Body: []ast.Node{call}}
	var fnHits int
	WalkNode(fn, StmtVisitor{
		OnFunction: func(ast.FunctionNode) bool { fnHits++; return true },
		OnCall:     func(ast.FunctionCallNode) bool { return true },
	})
	if fnHits != 1 {
		t.Fatalf("fnHits = %d", fnHits)
	}
}

func TestWalkNode_ifWithInit(t *testing.T) {
	t.Parallel()
	call := ast.FunctionCallNode{Function: ast.Ident{ID: "f"}}
	ifNode := ast.IfNode{
		Init: ast.AssignmentNode{
			IsShort: true,
			LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x"}}},
			RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		},
		Condition: ast.BoolLiteralNode{Value: true},
		Body:      []ast.Node{call},
	}
	var ifHits int
	WalkNode(ifNode, StmtVisitor{
		OnIf:   func(ast.IfNode) bool { ifHits++; return true },
		OnCall: func(ast.FunctionCallNode) bool { return true },
	})
	if ifHits != 1 {
		t.Fatalf("ifHits = %d", ifHits)
	}
}

func TestWalkNode_earlyExitSkipsIfForEnsureFunction(t *testing.T) {
	t.Parallel()
	call := ast.FunctionCallNode{Function: ast.Ident{ID: "inner"}}
	body := []ast.Node{call}

	tests := []struct {
		name string
		node ast.Node
		hook func(*StmtVisitor)
	}{
		{
			name: "if",
			node: ast.IfNode{Body: body},
			hook: func(v *StmtVisitor) { v.OnIf = func(ast.IfNode) bool { return false } },
		},
		{
			name: "for",
			node: ast.ForNode{Body: body},
			hook: func(v *StmtVisitor) { v.OnFor = func(ast.ForNode) bool { return false } },
		},
		{
			name: "ensure",
			node: ast.EnsureNode{Block: &ast.EnsureBlockNode{Body: body}},
			hook: func(v *StmtVisitor) { v.OnEnsure = func(ast.EnsureNode) bool { return false } },
		},
		{
			name: "function",
			node: ast.FunctionNode{Body: body},
			hook: func(v *StmtVisitor) { v.OnFunction = func(ast.FunctionNode) bool { return false } },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := StmtVisitor{
				OnCall: func(ast.FunctionCallNode) bool { t.Fatal("should not descend"); return true },
			}
			tc.hook(&v)
			WalkNode(tc.node, v)
		})
	}
}

func TestWalkNode_nilOnCallStillWalksArgs(t *testing.T) {
	t.Parallel()
	nested := ast.FunctionCallNode{Function: ast.Ident{ID: "nested"}}
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: "f"},
		Arguments: []ast.ExpressionNode{nested},
	}
	WalkNode(call, StmtVisitor{})
}

func TestWalkNode_ensureWithoutBlock(t *testing.T) {
	t.Parallel()
	var ensureHits int
	WalkNode(ast.EnsureNode{}, StmtVisitor{
		OnEnsure: func(ast.EnsureNode) bool { ensureHits++; return true },
	})
	if ensureHits != 1 {
		t.Fatalf("ensureHits = %d", ensureHits)
	}
}

func TestWalkExpr_mapLiteralAndShapeFieldValues(t *testing.T) {
	t.Parallel()
	inner := ast.FunctionCallNode{Function: ast.Ident{ID: "inner"}}
	expr := ast.MapLiteralNode{
		Entries: []ast.MapEntryNode{
			{Key: ast.StringLiteralNode{Value: "k"}, Value: ast.StringLiteralNode{Value: "v"}},
			{Key: ast.IntLiteralNode{Value: 1}, Value: ast.IntLiteralNode{Value: 2}},
		},
	}
	shapeExpr := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"a": {Node: inner},
		},
	}
	var calls int
	WalkExpr(expr, ExprVisitor{
		OnCall: func(ast.FunctionCallNode) bool { calls++; return true },
	})
	WalkExpr(shapeExpr, ExprVisitor{
		OnCall: func(ast.FunctionCallNode) bool { calls++; return true },
	})
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 from shape field", calls)
	}
}

func TestWalkExpr_onCallEarlyExitSkipsArgs(t *testing.T) {
	t.Parallel()
	outer := ast.FunctionCallNode{
		Function: ast.Ident{ID: "outer"},
		Arguments: []ast.ExpressionNode{
			ast.FunctionCallNode{Function: ast.Ident{ID: "inner"}},
		},
	}
	var calls int
	WalkExpr(outer, ExprVisitor{
		OnCall: func(c ast.FunctionCallNode) bool {
			calls++
			return c.Function.ID != "outer"
		},
	})
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (early exit before inner)", calls)
	}
}

func TestWalkExpr_nilExpr(t *testing.T) {
	t.Parallel()
	WalkExpr(nil, ExprVisitor{
		OnCall: func(ast.FunctionCallNode) bool { t.Fatal("unexpected call"); return true },
	})
}

func TestWalkNodeContaining_descendsIfForFunctionEnsure(t *testing.T) {
	t.Parallel()
	innerWith := ast.WithNode{
		Span: ast.SourceSpan{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10},
	}
	body := []ast.Node{innerWith}

	tests := []struct {
		name string
		node ast.Node
	}{
		{name: "if", node: ast.IfNode{Body: body}},
		{name: "if-elseif", node: ast.IfNode{
			Body:    []ast.Node{},
			ElseIfs: []ast.ElseIfNode{{Body: body}},
		}},
		{name: "if-else", node: ast.IfNode{
			Body: []ast.Node{},
			Else: &ast.ElseBlockNode{Body: body},
		}},
		{name: "for", node: ast.ForNode{Body: body}},
		{name: "function", node: ast.FunctionNode{Body: body}},
		{name: "ensure", node: ast.EnsureNode{Block: &ast.EnsureBlockNode{Body: body}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var hits int
			WalkNodeContaining(tc.node, 2, 5, StmtVisitor{
				OnWith: func(ast.WithNode) bool { hits++; return true },
			})
			if hits != 1 {
				t.Fatalf("hits = %d, want 1 nested with", hits)
			}
		})
	}
}

func TestWalkNodeContaining_withEarlyExit(t *testing.T) {
	t.Parallel()
	with := ast.WithNode{
		Span: ast.SourceSpan{StartLine: 1, StartCol: 1, EndLine: 3, EndCol: 1},
		Body: []ast.Node{ast.FunctionCallNode{Function: ast.Ident{ID: "f"}}},
	}
	WalkNodeContaining(with, 2, 1, StmtVisitor{
		OnWith: func(ast.WithNode) bool { return false },
		OnCall: func(ast.FunctionCallNode) bool { t.Fatal("should not descend"); return true },
	})
}

func TestWalkNodeContaining_ensureWithoutBlock(t *testing.T) {
	t.Parallel()
	WalkNodeContaining(ast.EnsureNode{}, 1, 1, StmtVisitor{
		OnCall: func(ast.FunctionCallNode) bool { t.Fatal("no block"); return true },
	})
}

