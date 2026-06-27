package astwalk

import "forst/internal/ast"

// StmtVisitor receives statement-tree nodes during WalkStmts / WalkNode.
// Return false to skip descending into the node's children.
type StmtVisitor struct {
	OnWith    func(ast.WithNode) bool
	OnIf      func(ast.IfNode) bool
	OnFor     func(ast.ForNode) bool
	OnEnsure  func(ast.EnsureNode) bool
	OnFunction func(ast.FunctionNode) bool
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
