package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) writePathFromAssignTarget(lv ast.ExpressionNode) (*AccessPath, ast.SourceSpan) {
	switch l := lv.(type) {
	case ast.VariableNode:
		return tc.writePathFromVariableTarget(&l, l.Ident.Span)
	case *ast.VariableNode:
		if l == nil {
			return nil, ast.SourceSpan{}
		}
		return tc.writePathFromVariableTarget(l, l.Ident.Span)
	case ast.FieldAccessNode:
		return tc.writePathFromFieldTarget(l, l.Field.Span)
	case *ast.FieldAccessNode:
		if l == nil {
			return nil, ast.SourceSpan{}
		}
		return tc.writePathFromFieldTarget(*l, l.Field.Span)
	case ast.IndexExpressionNode:
		return tc.writePathFromIndexAssign(l)
	case *ast.IndexExpressionNode:
		if l == nil {
			return nil, ast.SourceSpan{}
		}
		return tc.writePathFromIndexAssign(*l)
	case ast.DereferenceNode:
		return tc.writePathFromDerefTarget(l)
	default:
		return nil, ast.SourceSpan{}
	}
}

func (tc *TypeChecker) writePathFromVariableTarget(l *ast.VariableNode, span ast.SourceSpan) (*AccessPath, ast.SourceSpan) {
	return tc.AccessPathForVariable(l), span
}

func (tc *TypeChecker) writePathFromFieldTarget(l ast.FieldAccessNode, span ast.SourceSpan) (*AccessPath, ast.SourceSpan) {
	id := dottedIdentFromExpr(l)
	vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(id), Span: l.Field.Span}}
	return tc.AccessPathForVariable(&vn), span
}

func (tc *TypeChecker) writePathFromDerefTarget(l ast.DereferenceNode) (*AccessPath, ast.SourceSpan) {
	inner := ""
	switch v := l.Value.(type) {
	case ast.VariableNode:
		inner = string(v.Ident.ID)
	case *ast.VariableNode:
		if v != nil {
			inner = string(v.Ident.ID)
		}
	}
	if inner == "" {
		return nil, ast.SourceSpan{}
	}
	vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(inner)}}
	base := tc.AccessPathForVariable(&vn)
	if base == nil {
		return nil, ast.SourceSpan{}
	}
	deref := tc.paths.Intern(AccessPath{Root: base.Root, Steps: append(base.CloneSteps(), AccessStep{Kind: AccessDeref})})
	return deref, ast.SourceSpan{}
}
