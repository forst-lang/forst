package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) applyWriteInvalidation(writePath *AccessPath, span ast.SourceSpan) {
	if tc == nil || writePath == nil {
		return
	}
	if tc.capturingClosure {
		tc.pendingClosureWrites = append(tc.pendingClosureWrites, writePath)
		return
	}
	tc.invalidateOverlappingFacts(writePath, span)
	tc.recordBranchOrLoopWrite(writePath, span)
	tc.recordParamWriteDuringInfer(writePath)
}

func (tc *TypeChecker) writePathFromIndexAssign(idx ast.IndexExpressionNode) (*AccessPath, ast.SourceSpan) {
	baseID := dottedIdentFromExpr(idx.Target)
	if baseID == "" {
		return nil, ast.SourceSpan{}
	}
	vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(baseID)}}
	base := tc.AccessPathForVariable(&vn)
	if base == nil {
		return nil, ast.SourceSpan{}
	}
	// users[0].age — IndexExpression may nest FieldAccess as target of outer assign
	// handled via VariableNode dotted ids; here target[index] → target[*].
	star := tc.paths.Intern(AccessPath{
		Root:  base.Root,
		Steps: append(base.CloneSteps(), AccessStep{Kind: AccessIndexAny}),
	})
	span := ast.SourceSpan{}
	if vn2, ok := idx.Target.(ast.VariableNode); ok {
		span = vn2.Ident.Span
	}
	return star, span
}
