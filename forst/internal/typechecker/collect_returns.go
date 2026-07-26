package typechecker

import "forst/internal/ast"

func collectReturnStatements(stmts []ast.Node) []ast.ReturnNode {
	var out []ast.ReturnNode
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case ast.ReturnNode:
			out = append(out, s)
		case *ast.ReturnNode:
			if s != nil {
				out = append(out, *s)
			}
		case ast.IfNode:
			out = append(out, collectReturnStatements(s.Body)...)
			for _, ei := range s.ElseIfs {
				out = append(out, collectReturnStatements(ei.Body)...)
			}
			if s.Else != nil {
				out = append(out, collectReturnStatements(s.Else.Body)...)
			}
		case *ast.IfNode:
			if s != nil {
				out = append(out, collectReturnStatementsFromIf(s)...)
			}
		case ast.SwitchNode:
			for _, clause := range s.Clauses {
				out = append(out, collectReturnStatements(clause.Body)...)
			}
		case *ast.SwitchNode:
			if s != nil {
				for _, clause := range s.Clauses {
					out = append(out, collectReturnStatements(clause.Body)...)
				}
			}
		case ast.ForNode:
			out = append(out, collectReturnStatements(s.Body)...)
		case *ast.ForNode:
			if s != nil {
				out = append(out, collectReturnStatements(s.Body)...)
			}
		case ast.WithNode:
			out = append(out, collectReturnStatements(s.Body)...)
		case *ast.WithNode:
			if s != nil {
				out = append(out, collectReturnStatements(s.Body)...)
			}
		case ast.EnsureNode:
			if s.Block != nil {
				out = append(out, collectReturnStatements(s.Block.Body)...)
			}
		case *ast.EnsureNode:
			if s != nil && s.Block != nil {
				out = append(out, collectReturnStatements(s.Block.Body)...)
			}
		}
	}
	return out
}

func collectReturnStatementsFromIf(n *ast.IfNode) []ast.ReturnNode {
	out := collectReturnStatements(n.Body)
	for _, ei := range n.ElseIfs {
		out = append(out, collectReturnStatements(ei.Body)...)
	}
	if n.Else != nil {
		out = append(out, collectReturnStatements(n.Else.Body)...)
	}
	return out
}
