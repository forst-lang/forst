package lsp

import (
	"fmt"
	"strings"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

func withAmbientHoverMarkdown(tc *typechecker.TypeChecker, nodes []ast.Node, tok *ast.Token) string {
	if tc == nil || tok == nil || tok.Type != ast.TokenWith {
		return ""
	}
	chain := collectWithChainContainingPosition(nodes, tok.Line, tok.Column)
	if len(chain) == 0 {
		return ""
	}
	keys, err := tc.EffectiveAmbientKeys(chain)
	if err != nil {
		return ""
	}
	if len(keys) == 0 {
		return "**Effective ambient:** (empty)"
	}
	return fmt.Sprintf("**Effective ambient:** %s", strings.Join(keys, ", "))
}

func collectWithChainContainingPosition(nodes []ast.Node, line, col int) []ast.WithNode {
	var chain []ast.WithNode
	walkCollectWithChain(nodes, line, col, &chain)
	return chain
}

func walkCollectWithChain(stmts []ast.Node, line, col int, chain *[]ast.WithNode) {
	for _, n := range stmts {
		walkNodeCollectWithChain(n, line, col, chain)
	}
}

func walkNodeCollectWithChain(n ast.Node, line, col int, chain *[]ast.WithNode) {
	switch node := n.(type) {
	case ast.WithNode:
		if node.Span.ContainsPosition(line, col) {
			*chain = append(*chain, node)
			walkCollectWithChain(node.Body, line, col, chain)
		}
	case ast.IfNode:
		walkCollectWithChain(node.Body, line, col, chain)
		for _, branch := range node.ElseIfs {
			walkCollectWithChain(branch.Body, line, col, chain)
		}
		if node.Else != nil {
			walkCollectWithChain(node.Else.Body, line, col, chain)
		}
	case ast.FunctionNode:
		walkCollectWithChain(node.Body, line, col, chain)
	case ast.ForNode:
		walkCollectWithChain(node.Body, line, col, chain)
	case ast.EnsureNode:
		if node.Block != nil {
			walkCollectWithChain(node.Block.Body, line, col, chain)
		}
	}
}
