package lsp

import (
	"fmt"
	"strings"

	"forst/internal/ast"
	"forst/internal/astwalk"
	"forst/internal/typechecker"
)

func withScopeHoverMarkdown(tc *typechecker.TypeChecker, nodes []ast.Node, tok *ast.Token) string {
	if tc == nil || tok == nil || tok.Type != ast.TokenWith {
		return ""
	}
	chain := collectWithChainContainingPosition(nodes, tok.Line, tok.Column)
	if len(chain) == 0 {
		return ""
	}
	labels, err := tc.EffectiveScopeKeyLabels(chain)
	if err != nil {
		return ""
	}
	if len(labels) == 0 {
		return "**Effective scope:** (empty)"
	}
	parts := make([]string, len(labels))
	for i, l := range labels {
		if l.Shadowed {
			parts[i] = l.Key + " (shadows outer)"
		} else {
			parts[i] = l.Key
		}
	}
	return fmt.Sprintf("**Effective scope:** %s", strings.Join(parts, ", "))
}

func collectWithChainContainingPosition(nodes []ast.Node, line, col int) []ast.WithNode {
	var chain []ast.WithNode
	astwalk.WalkStmtsContaining(nodes, line, col, astwalk.StmtVisitor{
		OnWith: func(w ast.WithNode) bool {
			chain = append(chain, w)
			return true
		},
	})
	return chain
}
