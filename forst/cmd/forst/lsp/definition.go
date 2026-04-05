package lsp

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// handleDefinition handles textDocument/definition: navigate to the defining token for functions,
// user-defined types, and type guards in the same file (token + typechecker resolution).
func (s *LSPServer) handleDefinition(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position LSPPosition `json:"position"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	loc := s.findDefinitionForPosition(params.TextDocument.URI, params.Position)
	if loc == nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  nil,
		}
	}

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  loc,
	}
}

func (s *LSPServer) findDefinitionForPosition(uri string, position LSPPosition) (loc *LSPLocation) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"function": "findDefinitionForPosition",
				"panic":    r,
			}).Debug("definition lookup panic recovered")
			loc = nil
		}
	}()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		return nil
	}
	tokens := ctx.Tokens
	tok := tokenAtLSPPosition(tokens, position)
	if tok == nil || tok.Type != ast.TokenIdentifier {
		return nil
	}
	if ctx.PackageMerge != nil {
		if loc := s.definingTopLevelLocationForPackage(ctx.TC, uri, tokens, tok, ctx.PackageMerge); loc != nil {
			return loc
		}
	} else {
		if defTok := definingTokenForNavigableSymbol(ctx.TC, tokens, tok); defTok != nil {
			return lspLocationPtrFromToken(uri, defTok)
		}
	}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, position)
	if tokIdx < 0 {
		return nil
	}
	if defTok := definingTokenForLocalBinding(ctx, tokIdx, tok); defTok != nil {
		return lspLocationPtrFromToken(uri, defTok)
	}
	return nil
}

// definingTopLevelLocationForPackage resolves go-to-definition for top-level functions, types, and
// type guards when the definition may live in another open file in the same package.
func (s *LSPServer) definingTopLevelLocationForPackage(tc *typechecker.TypeChecker, curURI string, curTokens []ast.Token, tok *ast.Token, merge *packageMergeInfo) *LSPLocation {
	if tok == nil || tok.Type != ast.TokenIdentifier || merge == nil {
		return nil
	}
	if defTok := definingTokenForNavigableSymbol(tc, curTokens, tok); defTok != nil {
		return lspLocationPtrFromToken(curURI, defTok)
	}
	id := ast.Identifier(tok.Value)
	if _, ok := tc.Functions[id]; ok {
		for _, u := range merge.MemberURIs {
			tks := merge.TokensByURI[u]
			if defTok := findFuncNameToken(tks, string(id)); defTok != nil {
				return lspLocationPtrFromToken(u, defTok)
			}
		}
		return nil
	}
	if strings.HasPrefix(tok.Value, "T_") {
		return nil
	}
	def, ok := tc.Defs[ast.TypeIdent(tok.Value)]
	if !ok {
		return nil
	}
	switch def.(type) {
	case ast.TypeDefNode:
		for _, u := range merge.MemberURIs {
			tks := merge.TokensByURI[u]
			if defTok := findTypeNameToken(tks, tok.Value); defTok != nil {
				return lspLocationPtrFromToken(u, defTok)
			}
		}
	case ast.TypeGuardNode, *ast.TypeGuardNode:
		for _, u := range merge.MemberURIs {
			tks := merge.TokensByURI[u]
			if defTok := findTypeGuardNameToken(tks, tok.Value); defTok != nil {
				return lspLocationPtrFromToken(u, defTok)
			}
		}
	default:
		return nil
	}
	return nil
}

// definingTokenForNavigableSymbol returns the defining identifier token for a top-level function,
// user type, or type guard when the cursor token refers to that name (same rules as go-to-definition).
func definingTokenForNavigableSymbol(tc *typechecker.TypeChecker, tokens []ast.Token, tok *ast.Token) *ast.Token {
	if tok == nil || tok.Type != ast.TokenIdentifier {
		return nil
	}
	id := ast.Identifier(tok.Value)
	if _, ok := tc.Functions[id]; ok {
		return findFuncNameToken(tokens, string(id))
	}
	if strings.HasPrefix(tok.Value, "T_") {
		return nil
	}
	def, ok := tc.Defs[ast.TypeIdent(tok.Value)]
	if !ok {
		return nil
	}
	switch def.(type) {
	case ast.TypeDefNode:
		return findTypeNameToken(tokens, tok.Value)
	case *ast.TypeGuardNode:
		return findTypeGuardNameToken(tokens, tok.Value)
	default:
		return nil
	}
}

func lspLocationPtrFromToken(uri string, t *ast.Token) *LSPLocation {
	loc := lspLocationFromToken(uri, t)
	return &loc
}

func lspLocationFromToken(uri string, t *ast.Token) LSPLocation {
	width := utf8.RuneCountInString(t.Value)
	if width < 1 {
		width = 1
	}
	line0 := t.Line - 1
	if line0 < 0 {
		line0 = 0
	}
	col0 := t.Column - 1
	if col0 < 0 {
		col0 = 0
	}
	return LSPLocation{
		URI: uri,
		Range: LSPRange{
			Start: LSPPosition{Line: line0, Character: col0},
			End:   LSPPosition{Line: line0, Character: col0 + width},
		},
	}
}

// braceDepthAtIndex returns the brace nesting depth just before tokens[i] (0 = top-level).
func braceDepthAtIndex(tokens []ast.Token, i int) int {
	d := 0
	for j := 0; j < i && j < len(tokens); j++ {
		switch tokens[j].Type {
		case ast.TokenLBrace:
			d++
		case ast.TokenRBrace:
			if d > 0 {
				d--
			}
		}
	}
	return d
}

func findFuncNameToken(tokens []ast.Token, name string) *ast.Token {
	for i := 0; i+1 < len(tokens); i++ {
		if braceDepthAtIndex(tokens, i) != 0 {
			continue
		}
		if tokens[i].Type == ast.TokenFunc && tokens[i+1].Type == ast.TokenIdentifier && tokens[i+1].Value == name {
			return &tokens[i+1]
		}
	}
	return nil
}

func findTypeNameToken(tokens []ast.Token, name string) *ast.Token {
	for i := 0; i+1 < len(tokens); i++ {
		if braceDepthAtIndex(tokens, i) != 0 {
			continue
		}
		if tokens[i].Type == ast.TokenType && tokens[i+1].Type == ast.TokenIdentifier && tokens[i+1].Value == name {
			return &tokens[i+1]
		}
	}
	return nil
}

// findTypeGuardNameToken finds the guard name identifier in a top-level `is (…) Name` declaration.
// Brace depth ensures `ensure x is …` inside functions is not mistaken for a type guard.
func findTypeGuardNameToken(tokens []ast.Token, name string) *ast.Token {
	for i := 0; i+1 < len(tokens); i++ {
		if braceDepthAtIndex(tokens, i) != 0 {
			continue
		}
		if tokens[i].Type != ast.TokenIs || tokens[i+1].Type != ast.TokenLParen {
			continue
		}
		j := i + 1
		pdepth := 0
		for k := j; k < len(tokens); k++ {
			switch tokens[k].Type {
			case ast.TokenLParen:
				pdepth++
			case ast.TokenRParen:
				pdepth--
				if pdepth == 0 {
					if k+1 < len(tokens) && tokens[k+1].Type == ast.TokenIdentifier && tokens[k+1].Value == name {
						return &tokens[k+1]
					}
					break
				}
			}
		}
	}
	return nil
}
