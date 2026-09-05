package gointerop

import (
	"forst/internal/ast"
	"go/types"
)

// AssignabilityHost supplies type compatibility checks for Go FFI parameters.
type AssignabilityHost interface {
	ForstTypeForGoType(g types.Type) (ast.TypeNode, bool)
	IsTypeCompatible(a, b ast.TypeNode) bool
	// GoTypeForForstType reconstructs a go/types value for a Forst type at the FFI boundary.
	GoTypeForForstType(f ast.TypeNode) types.Type
}

// Host supplies typechecker callbacks needed for Go FFI call checking.
type Host interface {
	AssignabilityHost
	InferExpressionType(expr ast.ExpressionNode) ([]ast.TypeNode, error)
	// GoTypeForExpression returns the tracked go/types type for an expression, if known.
	GoTypeForExpression(expr ast.ExpressionNode) types.Type
}

// Diagnose formats a Go-interop diagnostic. Callers typically wrap this as typechecker.Diagnostic.
type Diagnose func(span ast.SourceSpan, code, format string, args ...any) error
