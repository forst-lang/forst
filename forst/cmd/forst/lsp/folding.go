package lsp

import (
	"unicode/utf8"

	"forst/internal/ast"
)

// foldingRangeFromTokenPair builds an LSP folding range for a `{` … `}` pair (token indices).
func foldingRangeFromTokenPair(tokens []ast.Token, lBraceIdx, rBraceIdx int) map[string]interface{} {
	l := &tokens[lBraceIdx]
	r := &tokens[rBraceIdx]
	rw := utf8.RuneCountInString(r.Value)
	if rw < 1 {
		rw = 1
	}
	startLine := l.Line - 1
	if startLine < 0 {
		startLine = 0
	}
	startChar := l.Column - 1
	if startChar < 0 {
		startChar = 0
	}
	endLine := r.Line - 1
	if endLine < 0 {
		endLine = 0
	}
	endChar := r.Column - 1 + rw
	return map[string]interface{}{
		"startLine":      startLine,
		"startCharacter": startChar,
		"endLine":        endLine,
		"endCharacter":   endChar,
		"kind":           "region",
	}
}

func tokenIndexInSlice(tokens []ast.Token, tok *ast.Token) int {
	if tok == nil {
		return -1
	}
	for i := range tokens {
		if &tokens[i] == tok {
			return i
		}
	}
	for i := range tokens {
		t := &tokens[i]
		if t.Line == tok.Line && t.Column == tok.Column && t.Type == tok.Type && t.Value == tok.Value {
			return i
		}
	}
	return -1
}

// foldingBracesForFunctionAfterNameToken returns token indices of `{` and `}` for the function body.
func foldingBracesForFunctionAfterNameToken(tokens []ast.Token, nameIdx int) (lBrace, rBrace int) {
	if nameIdx < 0 || nameIdx >= len(tokens) {
		return -1, -1
	}
	k := nameIdx + 1
	for k < len(tokens) && tokens[k].Type != ast.TokenLParen {
		k++
	}
	if k >= len(tokens) {
		return -1, -1
	}
	closeParen := skipBalancedParens(tokens, k)
	if closeParen < 0 {
		return -1, -1
	}
	for j := closeParen + 1; j < len(tokens); j++ {
		if tokens[j].Type == ast.TokenLBrace && braceDepthAtIndex(tokens, j) == 0 {
			rb := matchingRBrace(tokens, j)
			if rb >= 0 {
				return j, rb
			}
			return -1, -1
		}
	}
	return -1, -1
}

// foldingBracesForTypeDefAfterNameToken finds the shape/value `{` … `}` for a type definition when present.
func foldingBracesForTypeDefAfterNameToken(tokens []ast.Token, nameIdx int) (lBrace, rBrace int) {
	if nameIdx < 0 || nameIdx >= len(tokens) {
		return -1, -1
	}
	for k := nameIdx + 1; k < len(tokens); k++ {
		if tokens[k].Type != ast.TokenLBrace {
			continue
		}
		if braceDepthAtIndex(tokens, k) != 0 {
			continue
		}
		rb := matchingRBrace(tokens, k)
		if rb >= 0 {
			return k, rb
		}
	}
	return -1, -1
}

// foldingBracesAfterTypeGuardName finds the type guard body `{` … `}`.
func foldingBracesAfterTypeGuardName(tokens []ast.Token, nameIdx int) (lBrace, rBrace int) {
	if nameIdx < 0 || nameIdx >= len(tokens) {
		return -1, -1
	}
	k := nameIdx + 1
	if k < len(tokens) && tokens[k].Type == ast.TokenLParen {
		closeParen := skipBalancedParens(tokens, k)
		if closeParen < 0 {
			return -1, -1
		}
		k = closeParen + 1
	}
	for ; k < len(tokens); k++ {
		if tokens[k].Type == ast.TokenLBrace && braceDepthAtIndex(tokens, k) == 0 {
			rb := matchingRBrace(tokens, k)
			if rb >= 0 {
				return k, rb
			}
			return -1, -1
		}
	}
	return -1, -1
}

func (s *LSPServer) foldingRangesForURI(uri string) []interface{} {
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.Nodes == nil {
		return []interface{}{}
	}
	tokens := ctx.Tokens
	var out []interface{}
	for _, n := range ctx.Nodes {
		switch v := n.(type) {
		case ast.FunctionNode:
			name := string(v.Ident.ID)
			ft := findFuncNameToken(tokens, name)
			idx := tokenIndexInSlice(tokens, ft)
			if idx < 0 {
				continue
			}
			lb, rb := foldingBracesForFunctionAfterNameToken(tokens, idx)
			if lb >= 0 && rb >= 0 {
				out = append(out, foldingRangeFromTokenPair(tokens, lb, rb))
			}
		case ast.TypeDefNode:
			name := string(v.Ident)
			if len(name) >= 2 && name[0:2] == "T_" {
				continue
			}
			tt := findTypeNameToken(tokens, name)
			idx := tokenIndexInSlice(tokens, tt)
			if idx < 0 {
				continue
			}
			lb, rb := foldingBracesForTypeDefAfterNameToken(tokens, idx)
			if lb >= 0 && rb >= 0 {
				out = append(out, foldingRangeFromTokenPair(tokens, lb, rb))
			}
		case *ast.TypeGuardNode:
			name := string(v.Ident)
			gt := findTypeGuardNameToken(tokens, name)
			idx := tokenIndexInSlice(tokens, gt)
			if idx < 0 {
				continue
			}
			lb, rb := foldingBracesAfterTypeGuardName(tokens, idx)
			if lb >= 0 && rb >= 0 {
				out = append(out, foldingRangeFromTokenPair(tokens, lb, rb))
			}
		case ast.TypeGuardNode:
			name := string(v.Ident)
			gt := findTypeGuardNameToken(tokens, name)
			idx := tokenIndexInSlice(tokens, gt)
			if idx < 0 {
				continue
			}
			lb, rb := foldingBracesAfterTypeGuardName(tokens, idx)
			if lb >= 0 && rb >= 0 {
				out = append(out, foldingRangeFromTokenPair(tokens, lb, rb))
			}
		}
	}
	return out
}
