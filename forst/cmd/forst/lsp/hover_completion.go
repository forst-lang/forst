package lsp

import (
	"encoding/json"
	"fmt"
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

// handleHover handles the textDocument/hover method
func (s *LSPServer) handleHover(request LSPRequest) LSPServerResponse {
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

	hover := s.findHoverForPosition(params.TextDocument.URI, params.Position)
	if hover == nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  nil,
		}
	}

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  hover,
	}
}

// handleCompletion handles the textDocument/completion method
func (s *LSPServer) handleCompletion(request LSPRequest) LSPServerResponse {
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

	completions := s.getCompletionsForPosition(params.TextDocument.URI, params.Position)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"isIncomplete": false,
			"items":        completions,
		},
	}
}

// findHoverForPosition finds hover information for a given position.
func (s *LSPServer) findHoverForPosition(uri string, position LSPPosition) *LSPHover {
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

	return s.hoverMarkdownForContent(filePath, content, position)
}

func basicHoverMarkdown(text string) *LSPHover {
	return &LSPHover{
		Contents: LSPMarkedString{
			Language: "markdown",
			Value:    text,
		},
	}
}

func (s *LSPServer) hoverMarkdownForContent(filePath, content string, position LSPPosition) *LSPHover {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"function": "hoverMarkdownForContent",
				"panic":    r,
			}).Debug("hover panic recovered")
		}
	}()

	cd, ok := s.debugger.(*CompilerDebugger)
	if !ok {
		return nil
	}
	packageStore := cd.packageStore
	fileID := packageStore.RegisterFile(filePath, extractPackagePath(filePath))
	lex := lexer.New([]byte(content), string(fileID), s.log)
	tokens := lex.Lex()
	tok := tokenAtLSPPosition(tokens, position)
	if tok == nil {
		return nil
	}

	psr := parser.New(tokens, string(fileID), s.log)
	astNodes, err := psr.ParseFile()
	if err != nil {
		// Diagnostics carry parse errors; hover stays quiet (matches typical TS / LSP behavior).
		return nil
	}

	tc := typechecker.New(s.log, false)
	if err := tc.CheckTypes(astNodes); err != nil {
		// Still show quick info for the symbol when inferable; full error is in Problems.
		text := hoverTextForToken(tc, tokens, tok)
		if text == "" {
			return nil
		}
		return basicHoverMarkdown(text)
	}

	text := hoverTextForToken(tc, tokens, tok)
	if text == "" {
		return nil
	}
	return basicHoverMarkdown(text)
}

func tokenAtLSPPosition(tokens []ast.Token, pos LSPPosition) *ast.Token {
	line1 := int(pos.Line) + 1
	char1 := int(pos.Character) + 1 // 1-based column to match lexer

	var best *ast.Token
	for i := range tokens {
		t := &tokens[i]
		if t.Type == ast.TokenEOF || t.Line != line1 {
			continue
		}
		width := utf8.RuneCountInString(t.Value)
		if width < 1 {
			width = 1
		}
		endCol := t.Column + width - 1
		if char1 >= t.Column && char1 <= endCol {
			if best == nil || t.Column >= best.Column {
				best = t
			}
		}
	}
	return best
}

func hoverTextForToken(tc *typechecker.TypeChecker, tokens []ast.Token, tok *ast.Token) string {
	if tok.Type == ast.TokenIdentifier {
		id := ast.Identifier(tok.Value)
		if types, ok := tc.VariableTypes[id]; ok && len(types) > 0 {
			var parts []string
			for _, tn := range types {
				parts = append(parts, tn.String())
			}
			return fmt.Sprintf("```forst\n%s: %s\n```", tok.Value, strings.Join(parts, ", "))
		}
		if sig, ok := tc.Functions[id]; ok {
			doc := leadingCommentDocBeforeFunc(tokens, string(id))
			body := fmt.Sprintf("```forst\n%s\n```", sig.String())
			if doc == "" {
				return body
			}
			return doc + "\n\n" + body
		}
		if def, ok := tc.Defs[ast.TypeIdent(tok.Value)]; ok {
			return typeDefHoverMarkdown(def)
		}
		return ""
	}

	if literalHover(tok) {
		return ""
	}
	if kw := keywordHover(tok); kw != "" {
		return kw
	}
	return ""
}

func typeDefHoverMarkdown(def ast.Node) string {
	switch d := def.(type) {
	case ast.TypeDefNode:
		return fmt.Sprintf("```forst\ntype %s\n```", d.Ident)
	default:
		return ""
	}
}

// literalHover reports whether the token is a literal. We intentionally omit hover text for
// literals (aligned with TypeScript: no noisy hovers on `42`, `"hi"`, etc.).
func literalHover(tok *ast.Token) bool {
	switch tok.Type {
	case ast.TokenIntLiteral, ast.TokenFloatLiteral, ast.TokenStringLiteral,
		ast.TokenTrue, ast.TokenFalse, ast.TokenNil:
		return true
	default:
		return false
	}
}

// keywordHover returns a single-line quick info string (keyword in backticks), similar to
// built-in TypeScript keyword hovers — no tutorial paragraphs.
func keywordHover(tok *ast.Token) string {
	switch tok.Type {
	case ast.TokenFunc:
		return "`func`"
	case ast.TokenType:
		return "`type`"
	case ast.TokenReturn:
		return "`return`"
	case ast.TokenEnsure:
		return "`ensure`"
	case ast.TokenImport:
		return "`import`"
	case ast.TokenPackage:
		return "`package`"
	case ast.TokenInt:
		return "`Int`"
	case ast.TokenFloat:
		return "`Float`"
	case ast.TokenString:
		return "`String`"
	case ast.TokenBool:
		return "`Bool`"
	case ast.TokenVoid:
		return "`Void`"
	case ast.TokenArray:
		return "`Array`"
	case ast.TokenStruct:
		return "`struct`"
	default:
		return ""
	}
}

// leadingCommentDocBeforeFunc returns plain text from // or /* */ comments that appear in the
// token stream immediately before `func` `<name>` (TypeScript-style doc above the signature).
func leadingCommentDocBeforeFunc(tokens []ast.Token, funcName string) string {
	idx := findFuncKeywordIndex(tokens, funcName)
	if idx < 0 {
		return ""
	}
	parts := collectContiguousLeadingCommentLines(tokens, idx)
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func findFuncKeywordIndex(tokens []ast.Token, funcName string) int {
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Type != ast.TokenFunc {
			continue
		}
		if tokens[i+1].Type == ast.TokenIdentifier && tokens[i+1].Value == funcName {
			return i
		}
	}
	return -1
}

func collectContiguousLeadingCommentLines(tokens []ast.Token, funcKeywordIdx int) []string {
	var rev []string
	for j := funcKeywordIdx - 1; j >= 0 && tokens[j].Type == ast.TokenComment; j-- {
		line := stripCommentBody(tokens[j].Value)
		if line != "" {
			rev = append(rev, line)
		}
	}
	n := len(rev)
	out := make([]string, n)
	for i := range rev {
		out[i] = rev[n-1-i]
	}
	return out
}

func stripCommentBody(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "//") {
		return strings.TrimSpace(s[2:])
	}
	if strings.HasPrefix(s, "/*") {
		body := strings.TrimSuffix(strings.TrimPrefix(s, "/*"), "*/")
		return strings.TrimSpace(body)
	}
	return s
}

// getCompletionsForPosition gets completion items for a given position
func (s *LSPServer) getCompletionsForPosition(uri string, position LSPPosition) []LSPCompletionItem {
	// Convert URI to file path
	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}

	// Only process .ft files
	if !strings.HasSuffix(filePath, ".ft") {
		return nil
	}

	// Return Forst keywords as completion items
	var completions []LSPCompletionItem
	for keyword := range lexer.Keywords {
		completions = append(completions, LSPCompletionItem{
			Label:            keyword,
			Kind:             LSPCompletionItemKindKeyword,
			Detail:           "Forst keyword",
			Documentation:    keyword,
			InsertText:       keyword,
			InsertTextFormat: LSPInsertTextFormatPlainText,
		})
	}

	return completions
}

// handleWorkspaceSymbol handles the workspace/symbol method
func (s *LSPServer) handleWorkspaceSymbol(request LSPRequest) LSPServerResponse {
	// Parse query from params
	var params map[string]interface{}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	// For now, return empty array (no workspace symbols found)
	// TODO: Implement actual workspace symbol search
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  []interface{}{},
	}
}
