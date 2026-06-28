package transformergo

import "forst/internal/ast"

// lastImplicitReturnIndex returns the body index of a trailing expression statement when
// the function body ends with an expression that is not an explicit return.
func lastImplicitReturnIndex(body []ast.Node) (int, bool) {
	for i := len(body) - 1; i >= 0; i-- {
		if _, ok := body[i].(ast.CommentNode); ok {
			continue
		}
		if _, ok := body[i].(ast.ReturnNode); ok {
			return -1, false
		}
		if _, ok := body[i].(ast.ExpressionNode); ok {
			return i, true
		}
		return -1, false
	}
	return -1, false
}
