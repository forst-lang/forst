package lsp

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/parser"
)

// receiverExpressionSourceBeforeDot returns the source text of the receiver expression immediately
// before the dot at dotIdx (tokens[dotIdx] must be TokenDot).
func receiverExpressionSourceBeforeDot(tokens []ast.Token, dotIdx int) (string, bool) {
	if dotIdx <= 0 || dotIdx >= len(tokens) || tokens[dotIdx].Type != ast.TokenDot {
		return "", false
	}
	end := dotIdx - 1
	for end >= 0 && tokens[end].Type == ast.TokenComment {
		end--
	}
	if end < 0 {
		return "", false
	}
	start := end
	paren, brack, brace := 0, 0, 0
	for i := end; i >= 0; i-- {
		tok := tokens[i]
		switch tok.Type {
		case ast.TokenRParen:
			paren++
		case ast.TokenLParen:
			paren--
			if paren < 0 {
				start = i
				goto done
			}
		case ast.TokenRBracket:
			brack++
		case ast.TokenLBracket:
			brack--
			if brack < 0 {
				start = i
				goto done
			}
		case ast.TokenRBrace:
			brace++
		case ast.TokenLBrace:
			brace--
			if brace < 0 {
				start = i
				goto done
			}
		case ast.TokenIdentifier, ast.TokenIntLiteral, ast.TokenFloatLiteral, ast.TokenStringLiteral,
			ast.TokenTrue, ast.TokenFalse, ast.TokenNil:
			if paren == 0 && brack == 0 && brace == 0 {
				start = i
			}
		case ast.TokenDot:
			if paren == 0 && brack == 0 && brace == 0 {
				start = i
			}
		case ast.TokenStar:
			if paren == 0 && brack == 0 && brace == 0 {
				start = i
			}
		default:
			if paren == 0 && brack == 0 && brace == 0 {
				start = i + 1
				goto done
			}
		}
	}
done:
	if start > end {
		return "", false
	}
	var b strings.Builder
	for i := start; i <= end; i++ {
		if tokens[i].Type == ast.TokenComment || tokens[i].Type == ast.TokenEOF {
			continue
		}
		b.WriteString(tokens[i].Value)
	}
	src := strings.TrimSpace(b.String())
	if src == "" {
		return "", false
	}
	return src, true
}

// parseReceiverExpressionBeforeDot parses the receiver expression to the left of tokens[dotIdx].
func parseReceiverExpressionBeforeDot(tokens []ast.Token, dotIdx int, fileID string) (ast.ExpressionNode, bool) {
	src, ok := receiverExpressionSourceBeforeDot(tokens, dotIdx)
	if !ok {
		return nil, false
	}
	expr, err := parser.ParseExpressionSource(nil, fileID, src)
	if err != nil {
		return nil, false
	}
	return expr, true
}

// memberAccessDotIndex returns the token index of the dot immediately before the cursor when
// completing or hovering a member access (expr.method).
func memberAccessDotIndex(tokens []ast.Token, pos LSPPosition) int {
	idx := tokenIndexAtLSPPosition(tokens, pos)
	if idx < 0 {
		return -1
	}
	line1 := pos.Line + 1
	char1 := pos.Character + 1
	if tokens[idx].Type == ast.TokenDot {
		if tokens[idx].Line != line1 {
			return -1
		}
		w := max(utf8.RuneCountInString(tokens[idx].Value), 1)
		if char1 >= tokens[idx].Column && char1 <= tokens[idx].Column+w {
			return idx
		}
	}
	if tokens[idx].Type == ast.TokenIdentifier {
		if idx >= 2 && tokens[idx-1].Type == ast.TokenDot && tokens[idx-1].Line == line1 {
			return idx - 1
		}
	}
	return -1
}

// isIdentStart reports whether r can start a Forst identifier (for simple recv fallback).
func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
