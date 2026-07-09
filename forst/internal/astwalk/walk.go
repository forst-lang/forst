package astwalk

import "forst/internal/ast"

// StmtVisitor receives statement-tree nodes during WalkStmts / WalkNode.
// Return false to skip descending into the node's children.
type StmtVisitor struct {
	OnWith     func(ast.WithNode) bool
	OnIf       func(ast.IfNode) bool
	OnFor      func(ast.ForNode) bool
	OnEnsure   func(ast.EnsureNode) bool
	OnFunction func(ast.FunctionNode) bool
	OnCall     func(ast.FunctionCallNode) bool
	OnAssign   func(ast.AssignmentNode) bool
	OnReturn   func(ast.ReturnNode) bool
	OnDefer    func(ast.DeferNode) bool
	OnGo       func(ast.GoStmtNode) bool
}

// ExprVisitor receives expression nodes during WalkExpr.
type ExprVisitor struct {
	OnCall func(ast.FunctionCallNode) bool
}

// WalkStmts walks each top-level node and descends through statement bodies.
func WalkStmts(stmts []ast.Node, v StmtVisitor) {
	for _, n := range stmts {
		WalkNode(n, v)
	}
}

// WalkNode descends through statement bodies when visitor hooks allow.
func WalkNode(n ast.Node, v StmtVisitor) {
	switch node := n.(type) {
	case ast.WithNode:
		if v.OnWith != nil && !v.OnWith(node) {
			return
		}
		WalkStmts(node.Body, v)
	case ast.IfNode:
		if v.OnIf != nil && !v.OnIf(node) {
			return
		}
		WalkStmts(node.Body, v)
		for _, branch := range node.ElseIfs {
			WalkStmts(branch.Body, v)
		}
		if node.Else != nil {
			WalkStmts(node.Else.Body, v)
		}
	case ast.FunctionNode:
		if v.OnFunction != nil && !v.OnFunction(node) {
			return
		}
		WalkStmts(node.Body, v)
	case ast.ForNode:
		if v.OnFor != nil && !v.OnFor(node) {
			return
		}
		WalkStmts(node.Body, v)
	case ast.EnsureNode:
		if v.OnEnsure != nil && !v.OnEnsure(node) {
			return
		}
		if node.Block != nil {
			WalkStmts(node.Block.Body, v)
		}
	case ast.FunctionCallNode:
		if v.OnCall != nil {
			v.OnCall(node)
		}
		for _, arg := range node.Arguments {
			WalkExpr(arg, ExprVisitor{OnCall: v.OnCall})
		}
	case ast.AssignmentNode:
		if v.OnAssign != nil {
			v.OnAssign(node)
		}
		for _, rv := range node.RValues {
			WalkExpr(rv, ExprVisitor{OnCall: v.OnCall})
		}
	case ast.ReturnNode:
		if v.OnReturn != nil {
			v.OnReturn(node)
		}
		for _, val := range node.Values {
			WalkExpr(val, ExprVisitor{OnCall: v.OnCall})
		}
	case ast.DeferNode:
		if v.OnDefer != nil {
			v.OnDefer(node)
		}
		WalkExpr(node.Call, ExprVisitor{OnCall: v.OnCall})
	case ast.GoStmtNode:
		if v.OnGo != nil {
			v.OnGo(node)
		}
		WalkExpr(node.Call, ExprVisitor{OnCall: v.OnCall})
	}
}

func WalkExprCall(expr ast.ExpressionNode, v StmtVisitor) {
	if expr == nil {
		return
	}
	if call, ok := expr.(ast.FunctionCallNode); ok {
		if v.OnCall != nil {
			v.OnCall(call)
		}
		for _, arg := range call.Arguments {
			WalkExpr(arg, ExprVisitor{OnCall: v.OnCall})
		}
	}
}

// WalkExpr walks an expression tree for nested calls.
func WalkExpr(expr ast.ExpressionNode, v ExprVisitor) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case ast.FunctionCallNode:
		if v.OnCall != nil {
			if !v.OnCall(e) {
				return
			}
		}
		for _, arg := range e.Arguments {
			WalkExpr(arg, v)
		}
	case ast.BinaryExpressionNode:
		WalkExpr(e.Left, v)
		WalkExpr(e.Right, v)
	case ast.UnaryExpressionNode:
		WalkExpr(e.Operand, v)
	case ast.IndexExpressionNode:
		WalkExpr(e.Target, v)
		WalkExpr(e.Index, v)
	case ast.SliceExpressionNode:
		WalkExpr(e.Target, v)
		if e.Low != nil {
			WalkExpr(e.Low, v)
		}
		if e.High != nil {
			WalkExpr(e.High, v)
		}
	case ast.SpreadExpressionNode:
		WalkExpr(e.Expr, v)
	case ast.FieldAccessNode:
		WalkExpr(e.Target, v)
	case ast.ReferenceNode:
		if inner, ok := e.Value.(ast.ExpressionNode); ok {
			WalkExpr(inner, v)
		}
	case ast.DereferenceNode:
		if inner, ok := e.Value.(ast.ExpressionNode); ok {
			WalkExpr(inner, v)
		}
	case ast.ShapeNode:
		for _, field := range e.Fields {
			if fv, ok := field.ValueExpression(); ok {
				WalkExpr(fv, v)
			}
		}
	case ast.ArrayLiteralNode:
		for _, el := range e.Value {
			WalkExpr(el, v)
		}
	case ast.MapLiteralNode:
		for _, entry := range e.Entries {
			if kv, ok := entry.Key.(ast.ExpressionNode); ok {
				WalkExpr(kv, v)
			}
			if vv, ok := entry.Value.(ast.ExpressionNode); ok {
				WalkExpr(vv, v)
			}
		}
	case ast.OkExprNode:
		WalkExpr(e.Value, v)
	case ast.ErrExprNode:
		WalkExpr(e.Value, v)
	}
}

// WalkStmtsContaining walks statement trees and only descends into nodes whose span contains (line, col).
func WalkStmtsContaining(stmts []ast.Node, line, col int, v StmtVisitor) {
	for _, n := range stmts {
		WalkNodeContaining(n, line, col, v)
	}
}

// WalkNodeContaining descends only when node.Span.ContainsPosition(line, col).
func WalkNodeContaining(n ast.Node, line, col int, v StmtVisitor) {
	switch node := n.(type) {
	case ast.WithNode:
		if !node.Span.ContainsPosition(line, col) {
			return
		}
		if v.OnWith != nil && !v.OnWith(node) {
			return
		}
		WalkStmtsContaining(node.Body, line, col, v)
	case ast.IfNode:
		WalkStmtsContaining(node.Body, line, col, v)
		for _, branch := range node.ElseIfs {
			WalkStmtsContaining(branch.Body, line, col, v)
		}
		if node.Else != nil {
			WalkStmtsContaining(node.Else.Body, line, col, v)
		}
	case ast.FunctionNode:
		WalkStmtsContaining(node.Body, line, col, v)
	case ast.ForNode:
		WalkStmtsContaining(node.Body, line, col, v)
	case ast.EnsureNode:
		if node.Block != nil {
			WalkStmtsContaining(node.Block.Body, line, col, v)
		}
	}
}
