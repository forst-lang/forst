package parser

import (
	"fmt"

	"forst/internal/ast"
	"forst/internal/lexer"

	"github.com/sirupsen/logrus"
)

// ParseExpressionSource parses a single expression from a source fragment (for LSP member completion).
func ParseExpressionSource(log *logrus.Logger, fileID, expr string) (ast.ExpressionNode, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}
	src := fmt.Sprintf("package __lsp__\nfunc __() { _ = %s }\n", expr)
	lex := lexer.New([]byte(src), fileID, log)
	tokens := lex.Lex()
	psr := New(tokens, fileID, log)
	var parseErr error
	var nodes []ast.Node
	func() {
		defer func() {
			if r := recover(); r != nil {
				if pe, ok := r.(*ParseError); ok {
					parseErr = pe
				} else {
					parseErr = fmt.Errorf("parser panic: %v", r)
				}
			}
		}()
		nodes, parseErr = psr.ParseFile()
	}()
	if parseErr != nil {
		return nil, parseErr
	}
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok {
			continue
		}
		for _, st := range fn.Body {
			asg, ok := st.(ast.AssignmentNode)
			if !ok || len(asg.RValues) != 1 {
				continue
			}
			return asg.RValues[0], nil
		}
	}
	return nil, fmt.Errorf("expression not found in parse wrapper")
}
