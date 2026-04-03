package lsp

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"
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

	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}
	if !strings.HasSuffix(filePath, ".ft") {
		return nil
	}

	s.documentMu.RLock()
	content, haveOpen := s.openDocuments[uri]
	s.documentMu.RUnlock()
	if !haveOpen || content == "" {
		b, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}
		content = string(b)
	}

	cd, ok := s.debugger.(*CompilerDebugger)
	if !ok {
		return nil
	}
	packageStore := cd.packageStore
	fileID := packageStore.RegisterFile(filePath, extractPackagePath(filePath))
	lex := lexer.New([]byte(content), string(fileID), s.log)
	tokens := lex.Lex()
	tok := tokenAtLSPPosition(tokens, position)
	if tok == nil || tok.Type != ast.TokenIdentifier {
		return nil
	}

	psr := parser.New(tokens, string(fileID), s.log)
	astNodes, err := psr.ParseFile()
	if err != nil {
		return nil
	}

	tc := typechecker.New(s.log, false)
	_ = tc.CheckTypes(astNodes)

	id := ast.Identifier(tok.Value)

	if _, ok := tc.Functions[id]; ok {
		if defTok := findFuncNameToken(tokens, string(id)); defTok != nil {
			return lspLocationPtrFromToken(uri, defTok)
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
		if defTok := findTypeNameToken(tokens, tok.Value); defTok != nil {
			return lspLocationPtrFromToken(uri, defTok)
		}
	case *ast.TypeGuardNode:
		if defTok := findTypeGuardNameToken(tokens, tok.Value); defTok != nil {
			return lspLocationPtrFromToken(uri, defTok)
		}
	}

	return nil
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
