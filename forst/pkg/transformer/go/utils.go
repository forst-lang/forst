package transformer_go

import (
	goast "go/ast"
)

// Helper function to parse expressions from strings
// This is a simplified version - in a real implementation,
// you would need a proper expression parser
func parseExpr(expr string) goast.Expr {
	// For simplicity, just return an identifier
	// In a real implementation, you would parse the expression
	return goast.NewIdent(expr)
}
