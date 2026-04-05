package typechecker

import "forst/internal/ast"

// variableOccurrenceKey identifies a specific source occurrence of a variable (identifier + span).
// Used so flow-sensitive types do not require changing the structural hasher (which would churn
// hash-based type names in generated Go).
type variableOccurrenceKey struct {
	ident ast.Identifier
	span  ast.SourceSpan
}
