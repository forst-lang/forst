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
			return basicHoverMarkdown("Forst language element")
		}
		content = string(b)
	}

	hover := s.hoverMarkdownForContent(filePath, content, position)
	if hover == nil {
		return basicHoverMarkdown("Forst language element")
	}
	return hover
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
		return basicHoverMarkdown("Forst language element")
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
		return basicHoverMarkdown(fmt.Sprintf("**Syntax**\n\n```\n%s\n```", err.Error()))
	}

	tc := typechecker.New(s.log, false)
	if err := tc.CheckTypes(astNodes); err != nil {
		return basicHoverMarkdown(fmt.Sprintf("%s\n\n---\n*Type checking:* `%s`", hoverTextForToken(tc, tok), err.Error()))
	}

	text := hoverTextForToken(tc, tok)
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

func hoverTextForToken(tc *typechecker.TypeChecker, tok *ast.Token) string {
	if tok.Type == ast.TokenIdentifier {
		id := ast.Identifier(tok.Value)
		if types, ok := tc.VariableTypes[id]; ok && len(types) > 0 {
			var parts []string
			for _, tn := range types {
				parts = append(parts, tn.String())
			}
			return fmt.Sprintf("`%s` · **%s**", tok.Value, strings.Join(parts, ", "))
		}
		if sig, ok := tc.Functions[id]; ok {
			return fmt.Sprintf("**function** `%s`\n\n```forst\n%s\n```", tok.Value, sig.String())
		}
		if def, ok := tc.Defs[ast.TypeIdent(tok.Value)]; ok {
			return fmt.Sprintf("**type** `%s`\n\n%s", tok.Value, summarizeTypeDefNode(def))
		}
		return fmt.Sprintf("`%s` · _identifier_", tok.Value)
	}

	if lit := literalHover(tok); lit != "" {
		return lit
	}
	if kw := keywordHover(tok); kw != "" {
		return kw
	}
	if tok.Type == ast.TokenDot {
		return "**`.`** · member access"
	}
	return fmt.Sprintf("**%s** `%s`", tok.Type, tok.Value)
}

func summarizeTypeDefNode(n ast.Node) string {
	switch d := n.(type) {
	case ast.TypeDefNode:
		return fmt.Sprintf("`type %s`", d.Ident)
	default:
		return fmt.Sprintf("`%T`", n)
	}
}

func literalHover(tok *ast.Token) string {
	switch tok.Type {
	case ast.TokenIntLiteral:
		return fmt.Sprintf("**integer literal** `%s`", tok.Value)
	case ast.TokenFloatLiteral:
		return fmt.Sprintf("**float literal** `%s`", tok.Value)
	case ast.TokenStringLiteral:
		return "**string literal**"
	case ast.TokenTrue, ast.TokenFalse:
		return "**boolean literal**"
	case ast.TokenNil:
		return "**nil**"
	default:
		return ""
	}
}

func keywordHover(tok *ast.Token) string {
	switch tok.Type {
	case ast.TokenFunc:
		return "**func** · declare a function"
	case ast.TokenType:
		return "**type** · declare a type alias or shape"
	case ast.TokenReturn:
		return "**return** · return from a function"
	case ast.TokenEnsure:
		return "**ensure** · contract / assertion"
	case ast.TokenImport:
		return "**import** · import a package"
	case ast.TokenPackage:
		return "**package** · package clause"
	case ast.TokenInt:
		return "**Int** · built-in integer type"
	case ast.TokenFloat:
		return "**Float** · built-in float type"
	case ast.TokenString:
		return "**String** · built-in string type"
	case ast.TokenBool:
		return "**Bool** · built-in boolean type"
	case ast.TokenVoid:
		return "**Void** · empty return"
	case ast.TokenArray:
		return "**Array** · array type"
	case ast.TokenStruct:
		return "**struct** · structural type"
	default:
		return ""
	}
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
