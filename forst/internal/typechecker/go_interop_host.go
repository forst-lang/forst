package typechecker

import (
	"go/types"

	"forst/internal/ast"
	"forst/internal/typechecker/gointerop"
)

type goInteropHost TypeChecker

func (tc *TypeChecker) goInteropHost() gointerop.Host {
	return (*goInteropHost)(tc)
}

func (tc *TypeChecker) goInteropDiag() gointerop.Diagnose {
	return func(span ast.SourceSpan, code, format string, args ...any) error {
		return diagnosticf(span, code, format, args...)
	}
}

func (h *goInteropHost) ForstTypeForGoType(g types.Type) (ast.TypeNode, bool) {
	return (*TypeChecker)(h).forstTypeForGoType(g)
}

func (h *goInteropHost) IsTypeCompatible(a, b ast.TypeNode) bool {
	return (*TypeChecker)(h).IsTypeCompatible(a, b)
}

func (h *goInteropHost) InferExpressionType(expr ast.ExpressionNode) ([]ast.TypeNode, error) {
	return (*TypeChecker)(h).inferExpressionType(expr)
}
