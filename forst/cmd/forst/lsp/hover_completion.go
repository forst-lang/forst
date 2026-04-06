package lsp

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/goload"
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
		Context  *struct {
			TriggerKind      int     `json:"triggerKind"`
			TriggerCharacter *string `json:"triggerCharacter"`
		} `json:"context"`
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

	var reqCtx *completionRequestContext
	if params.Context != nil {
		reqCtx = &completionRequestContext{TriggerKind: params.Context.TriggerKind}
		if params.Context.TriggerCharacter != nil {
			reqCtx.TriggerCharacter = *params.Context.TriggerCharacter
		}
	}
	completions, incomplete := s.getCompletionsForPosition(params.TextDocument.URI, params.Position, reqCtx)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"isIncomplete": incomplete,
			"items":        completions,
		},
	}
}

// findHoverForPosition finds hover information for a given position.
func (s *LSPServer) findHoverForPosition(uri string, position LSPPosition) *LSPHover {
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		return nil
	}
	tok := tokenAtLSPPosition(ctx.Tokens, position)
	if tok == nil {
		return nil
	}
	if ctx.ParseErr != nil {
		if text := lexicalHoverMarkdown(tok); text != "" {
			return basicHoverMarkdown(text)
		}
		return nil
	}
	if ctx.TC == nil {
		return nil
	}
	return s.hoverFromAnalyzedContext(ctx, tok)
}

// lexicalHoverMarkdown returns hover text without type information (parse failed or no TC).
func lexicalHoverMarkdown(tok *ast.Token) string {
	if tok == nil {
		return ""
	}
	if literalHover(tok) {
		return ""
	}
	if kw := keywordHover(tok); kw != "" {
		return kw
	}
	if tok.Type == ast.TokenIdentifier {
		return fmt.Sprintf("Lexical identifier `%s` (types unavailable until the file parses)", tok.Value)
	}
	return ""
}

func basicHoverMarkdown(text string) *LSPHover {
	return &LSPHover{
		Contents: LSPMarkedString{
			Language: "markdown",
			Value:    text,
		},
	}
}

func (s *LSPServer) hoverFromAnalyzedContext(ctx *forstDocumentContext, tok *ast.Token) *LSPHover {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"function": "hoverFromAnalyzedContext",
				"panic":    r,
			}).Debug("hover panic recovered")
		}
	}()

	tc := ctx.TC
	tokens := ctx.Tokens
	if ctx.CheckErr != nil {
		text := hoverTextForToken(tc, tokens, tok, ctx.PackageMerge)
		if text == "" {
			return nil
		}
		return basicHoverMarkdown(text)
	}

	text := hoverTextForToken(tc, tokens, tok, ctx.PackageMerge)
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

func tokensForFuncDocFromPackageMerge(merge *packageMergeInfo, funcName string) []ast.Token {
	if merge == nil {
		return nil
	}
	for _, u := range merge.MemberURIs {
		tks := merge.TokensByURI[u]
		if findFuncNameToken(tks, funcName) != nil {
			return tks
		}
	}
	return nil
}

func tokensForTypeGuardDocFromPackageMerge(merge *packageMergeInfo, guardName string) []ast.Token {
	if merge == nil {
		return nil
	}
	for _, u := range merge.MemberURIs {
		tks := merge.TokensByURI[u]
		if findTypeGuardNameToken(tks, guardName) != nil {
			return tks
		}
	}
	return nil
}

func leadingCommentDocBeforeTypeGuard(tokens []ast.Token, guardName string) string {
	idx := findTypeGuardIsKeywordIndex(tokens, guardName)
	if idx < 0 {
		return ""
	}
	parts := collectContiguousLeadingCommentLines(tokens, idx)
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// variableHoverMarkdownWithGuardDocs puts leading // or /* */ docs first (one block per predicate in
// the full chain, stacked vertically), then a ```forst``` line with the same predicate chain in the
// type using dotted calls (see FormatVariableOccurrenceTypeForHover).
func variableHoverMarkdownWithGuardDocs(tc *typechecker.TypeChecker, tokens []ast.Token, merge *packageMergeInfo, tok *ast.Token, types []ast.TypeNode) string {
	vn := ast.VariableNode{
		Ident: ast.Ident{ID: ast.Identifier(tok.Value), Span: ast.SpanFromToken(*tok)},
	}
	display := tc.FormatVariableOccurrenceTypeForHover(vn, types)
	body := fmt.Sprintf("```forst\n%s: %s\n```", tok.Value, display)

	chain := tc.PredicateChainForVariableHover(vn, types)
	var docBlocks []string
	for _, g := range chain {
		docTokens := tokens
		if merge != nil && leadingCommentDocBeforeTypeGuard(tokens, g) == "" {
			if alt := tokensForTypeGuardDocFromPackageMerge(merge, g); len(alt) > 0 {
				docTokens = alt
			}
		}
		doc := leadingCommentDocBeforeTypeGuard(docTokens, g)
		if doc == "" {
			continue
		}
		docBlocks = append(docBlocks, fmt.Sprintf("**%s**\n\n%s", g, doc))
	}
	if len(docBlocks) == 0 {
		return body
	}
	top := strings.Join(docBlocks, "\n\n")
	return top + "\n\n" + body
}

func hoverTextForToken(tc *typechecker.TypeChecker, tokens []ast.Token, tok *ast.Token, merge *packageMergeInfo) string {
	if tok.Type == ast.TokenStringLiteral {
		if s := goHoverFromImportString(tc, tokens, tok); s != "" {
			return s
		}
	}
	if tok.Type == ast.TokenIdentifier {
		if s := goHoverFromQualifiedGoIdentifier(tc, tokens, tok); s != "" {
			return s
		}
		id := ast.Identifier(tok.Value)
		// Prefer function and type definitions over variable types when the token names those things.
		if sig, ok := tc.Functions[id]; ok {
			docTokens := tokens
			if merge != nil && leadingCommentDocBeforeFunc(tokens, string(id)) == "" {
				if alt := tokensForFuncDocFromPackageMerge(merge, string(id)); len(alt) > 0 {
					docTokens = alt
				}
			}
			doc := leadingCommentDocBeforeFunc(docTokens, string(id))
			body := fmt.Sprintf("```forst\n%s\n```", tc.FormatFunctionSignatureDisplay(sig))
			if doc == "" {
				return body
			}
			return doc + "\n\n" + body
		}
		if def, ok := tc.Defs[ast.TypeIdent(tok.Value)]; ok {
			return typeDefHoverMarkdown(def)
		}
		vn := ast.VariableNode{
			Ident: ast.Ident{ID: id, Span: ast.SpanFromToken(*tok)},
		}
		if types, ok := tc.InferredTypesForVariableNode(vn); ok && len(types) > 0 {
			return variableHoverMarkdownWithGuardDocs(tc, tokens, merge, tok, types)
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

func tokenSliceIndex(tokens []ast.Token, tok *ast.Token) int {
	for i := range tokens {
		if &tokens[i] == tok {
			return i
		}
	}
	for i := range tokens {
		t := tokens[i]
		if t.Line == tok.Line && t.Column == tok.Column && t.Type == tok.Type && t.Value == tok.Value {
			return i
		}
	}
	return -1
}

func goImportLocalShadowedByForstVar(tc *typechecker.TypeChecker, pkgLocal string) bool {
	inf, ok := tc.InferredTypesForVariableIdentifier(ast.Identifier(pkgLocal))
	return ok && len(inf) > 0
}

// goHoverFromQualifiedGoIdentifier adds hover for import.pkg.sel (e.g. fmt.Println) when pkg is a Go import.
func goHoverFromQualifiedGoIdentifier(tc *typechecker.TypeChecker, tokens []ast.Token, tok *ast.Token) string {
	if tok.Type != ast.TokenIdentifier {
		return ""
	}
	i := tokenSliceIndex(tokens, tok)
	if i < 0 {
		return ""
	}
	// … pkg . NAME — cursor on exported name
	if i >= 2 && tokens[i-1].Type == ast.TokenDot && tokens[i-2].Type == ast.TokenIdentifier {
		pkgLocal := tokens[i-2].Value
		if goImportLocalShadowedByForstVar(tc, pkgLocal) || !tc.IsImportedLocalName(pkgLocal) {
			return ""
		}
		if md, ok := tc.GoHoverMarkdown(pkgLocal, tok.Value); ok {
			return md
		}
	}
	// PKG . name … — cursor on package identifier
	if i+2 < len(tokens) && tokens[i+1].Type == ast.TokenDot && tokens[i+2].Type == ast.TokenIdentifier {
		pkgLocal := tok.Value
		if goImportLocalShadowedByForstVar(tc, pkgLocal) || !tc.IsImportedLocalName(pkgLocal) {
			return ""
		}
		if md, ok := tc.GoHoverMarkdown(pkgLocal, ""); ok {
			return md
		}
	}
	return ""
}

func goHoverFromImportString(tc *typechecker.TypeChecker, tokens []ast.Token, tok *ast.Token) string {
	if tok.Type != ast.TokenStringLiteral {
		return ""
	}
	i := tokenSliceIndex(tokens, tok)
	if i < 0 {
		return ""
	}
	for j := i - 1; j >= 0; {
		switch tokens[j].Type {
		case ast.TokenComment:
			j--
		case ast.TokenImport:
			path := goload.ImportPathFromForst(tok.Value)
			if md, ok := tc.GoHoverMarkdownForImportPath(path); ok {
				return md
			}
			return ""
		case ast.TokenLParen, ast.TokenComma, ast.TokenRParen, ast.TokenStringLiteral:
			// StringLiteral: sibling path in `import ( "a", "b" )`; keep scanning for `import`.
			j--
		default:
			return ""
		}
	}
	return ""
}

func typeDefHoverMarkdown(def ast.Node) string {
	switch d := def.(type) {
	case ast.TypeDefNode:
		return fmt.Sprintf("```forst\ntype %s\n```", d.Ident)
	case ast.TypeGuardNode:
		return typeGuardHoverStub(&d)
	case *ast.TypeGuardNode:
		return typeGuardHoverStub(d)
	default:
		return ""
	}
}

func typeGuardHoverStub(g *ast.TypeGuardNode) string {
	if g == nil {
		return ""
	}
	// registerTypeGuard stores *TypeGuardNode in Defs; full signature formatting is optional follow-up.
	return fmt.Sprintf("```forst\nis ... %s\n```", string(g.Ident))
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

// handleWorkspaceSymbol handles workspace/symbol over open .ft buffers (didOpen/didChange sync only).
func (s *LSPServer) handleWorkspaceSymbol(request LSPRequest) LSPServerResponse {
	var params struct {
		Query string `json:"query"`
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

	q := strings.ToLower(strings.TrimSpace(params.Query))

	s.documentMu.RLock()
	seen := make(map[string]struct{}, len(s.openDocuments))
	uris := make([]string, 0, len(s.openDocuments))
	for u := range s.openDocuments {
		cu := canonicalFileURI(u)
		if _, ok := seen[cu]; ok {
			continue
		}
		seen[cu] = struct{}{}
		uris = append(uris, cu)
	}
	s.documentMu.RUnlock()

	var out []LspSymbolInformation
	for _, uri := range uris {
		if !isForstDocumentURI(uri) {
			continue
		}
		ctx, ok := s.analyzeForstDocument(uri)
		if !ok || ctx == nil || ctx.ParseErr != nil || ctx.Nodes == nil {
			continue
		}
		for _, sym := range symbolsFromParsedDocument(uri, ctx.Tokens, ctx.Nodes) {
			if q != "" && !strings.Contains(strings.ToLower(sym.Name), q) {
				continue
			}
			out = append(out, sym)
		}
	}

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  out,
	}
}
