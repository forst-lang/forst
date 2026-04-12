package lsp

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/typechecker"
)

// completionRequestContext is optional LSP CompletionContext (triggerKind / triggerCharacter).
type completionRequestContext struct {
	TriggerKind      int
	TriggerCharacter string
}

// completionZone classifies where in the source completion was requested.
type completionZone int

const (
	zoneUnknown completionZone = iota
	zoneTopLevel
	zoneInsideBlock
	zoneMemberAfterDot
)

func identifierPrefixAt(content string, pos LSPPosition) string {
	lines := strings.Split(content, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	runes := []rune(line)
	col := pos.Character
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}
	start := col
	for start > 0 && (unicode.IsLetter(runes[start-1]) || unicode.IsDigit(runes[start-1]) || runes[start-1] == '_') {
		start--
	}
	return string(runes[start:col])
}

func tokenIndexAtLSPPosition(tokens []ast.Token, pos LSPPosition) int {
	line1 := pos.Line + 1
	char1 := pos.Character + 1

	var lastIdxOnLine = -1
	for i := range tokens {
		t := &tokens[i]
		if t.Type == ast.TokenEOF {
			continue
		}
		if t.Line < line1 {
			continue
		}
		if t.Line > line1 {
			break
		}
		lastIdxOnLine = i
		width := utf8.RuneCountInString(t.Value)
		if width < 1 {
			width = 1
		}
		endCol := t.Column + width - 1
		if char1 >= t.Column && char1 <= endCol {
			return i
		}
	}
	if lastIdxOnLine >= 0 {
		t := &tokens[lastIdxOnLine]
		w := utf8.RuneCountInString(t.Value)
		if w < 1 {
			w = 1
		}
		endCol := t.Column + w - 1
		if char1 > endCol {
			return lastIdxOnLine
		}
	}
	if len(tokens) == 0 {
		return -1
	}
	return len(tokens) - 1
}

func isCursorInCommentOrString(tokens []ast.Token, idx int) bool {
	if idx < 0 || idx >= len(tokens) {
		return false
	}
	switch tokens[idx].Type {
	case ast.TokenComment, ast.TokenStringLiteral:
		return true
	default:
		return false
	}
}

func lineUpToCursor(content string, pos LSPPosition) string {
	lines := strings.Split(content, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	runes := []rune(line)
	col := pos.Character
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}
	return string(runes[:col])
}

func detectMemberAfterDot(linePrefix string, req *completionRequestContext) bool {
	if req != nil && req.TriggerCharacter == "." {
		return true
	}
	s := strings.TrimRight(linePrefix, " \t")
	return strings.HasSuffix(s, ".")
}

func inferCompletionZone(tokens []ast.Token, pos LSPPosition, content string, req *completionRequestContext) completionZone {
	idx := tokenIndexAtLSPPosition(tokens, pos)
	if idx >= 0 && isCursorInCommentOrString(tokens, idx) {
		return zoneUnknown
	}
	linePrefix := lineUpToCursor(content, pos)
	if detectMemberAfterDot(linePrefix, req) {
		return zoneMemberAfterDot
	}
	if idx < 0 {
		return zoneUnknown
	}
	d := braceDepthAtIndex(tokens, idx)
	if d == 0 {
		return zoneTopLevel
	}
	return zoneInsideBlock
}

func keywordsForZone(z completionZone) []string {
	var out []string
	for kw := range lexer.Keywords {
		switch z {
		case zoneTopLevel:
			if kw == "return" || kw == "ensure" || kw == "break" || kw == "continue" {
				continue
			}
		case zoneInsideBlock:
			if kw == "package" || kw == "import" {
				continue
			}
		case zoneMemberAfterDot:
			return nil
		default:
			return nil
		}
		out = append(out, kw)
	}
	return out
}

func matchingRBrace(tokens []ast.Token, lBraceIdx int) int {
	depth := 0
	for i := lBraceIdx; i < len(tokens); i++ {
		switch tokens[i].Type {
		case ast.TokenLBrace:
			depth++
		case ast.TokenRBrace:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func skipBalancedParens(tokens []ast.Token, openParenIdx int) int {
	depth := 0
	for i := openParenIdx; i < len(tokens); i++ {
		switch tokens[i].Type {
		case ast.TokenLParen:
			depth++
		case ast.TokenRParen:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findFuncKeywordIndex(tokens []ast.Token, funcName string) int {
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Type == ast.TokenFunc && tokens[i+1].Type == ast.TokenIdentifier && tokens[i+1].Value == funcName {
			return i
		}
	}
	return -1
}

func findLBraceAfterFuncSignature(tokens []ast.Token, funcKeywordIdx int) int {
	nameIdx := funcKeywordIdx + 1
	if nameIdx >= len(tokens) {
		return -1
	}
	j := nameIdx + 1
	if j >= len(tokens) || tokens[j].Type != ast.TokenLParen {
		return -1
	}
	closeParen := skipBalancedParens(tokens, j)
	if closeParen < 0 {
		return -1
	}
	k := closeParen + 1
	for k < len(tokens) && tokens[k].Type == ast.TokenComment {
		k++
	}
	if k < len(tokens) && tokens[k].Type == ast.TokenColon {
		k++
		for k < len(tokens) && tokens[k].Type != ast.TokenLBrace {
			k++
		}
	}
	if k < len(tokens) && tokens[k].Type == ast.TokenLBrace {
		return k
	}
	return -1
}

func functionBodyBraces(tokens []ast.Token, fn ast.FunctionNode) (lBrace, rBrace int) {
	name := string(fn.Ident.ID)
	idx := findFuncKeywordIndex(tokens, name)
	if idx < 0 {
		return -1, -1
	}
	lb := findLBraceAfterFuncSignature(tokens, idx)
	if lb < 0 {
		return -1, -1
	}
	rb := matchingRBrace(tokens, lb)
	return lb, rb
}

// paramListParenRange returns the indices of `(` and `)` around a function's parameter list
// (same region as findParamIdentTokenForFunction). ok is false if the signature cannot be found.
func paramListParenRange(tokens []ast.Token, fn ast.FunctionNode) (open, closeIdx int, ok bool) {
	idx := findFuncKeywordIndex(tokens, string(fn.Ident.ID))
	if idx < 0 {
		return -1, -1, false
	}
	j := idx + 2
	if j >= len(tokens) || tokens[j].Type != ast.TokenLParen {
		return -1, -1, false
	}
	closeParen := skipBalancedParens(tokens, j)
	if closeParen < 0 {
		return -1, -1, false
	}
	return j, closeParen, true
}

func findFirstToken(tokens []ast.Token, from, limit int, typ ast.TokenIdent) int {
	if limit > len(tokens) {
		limit = len(tokens)
	}
	for i := from; i < limit; i++ {
		if tokens[i].Type == typ {
			return i
		}
	}
	return -1
}

// scanToOpeningThenBrace finds the `{` that opens an if/else-if then-branch: optional init
// (semicolon at paren depth 0), then condition, then `{`. Matches Forst `if cond {` (no parens)
// as well as `if (cond) {` and `if init; cond {`.
func scanToOpeningThenBrace(tokens []ast.Token, j int) (l, r int) {
	paren := 0
	for j < len(tokens) {
		if tokens[j].Type == ast.TokenLBrace && paren == 0 {
			l = j
			r = matchingRBrace(tokens, l)
			return l, r
		}
		if tokens[j].Type == ast.TokenSemicolon && paren == 0 {
			j++
			break
		}
		switch tokens[j].Type {
		case ast.TokenLParen:
			paren++
		case ast.TokenRParen:
			if paren > 0 {
				paren--
			}
		}
		j++
	}
	paren = 0
	for j < len(tokens) {
		if tokens[j].Type == ast.TokenLBrace && paren == 0 {
			l = j
			r = matchingRBrace(tokens, l)
			return l, r
		}
		switch tokens[j].Type {
		case ast.TokenLParen:
			paren++
		case ast.TokenRParen:
			if paren > 0 {
				paren--
			}
		}
		j++
	}
	return -1, -1
}

func ifThenBraces(tokens []ast.Token, ifKeywordIdx int) (l, r int) {
	if ifKeywordIdx < 0 || ifKeywordIdx >= len(tokens) || tokens[ifKeywordIdx].Type != ast.TokenIf {
		return -1, -1
	}
	j := ifKeywordIdx + 1
	for j < len(tokens) && tokens[j].Type == ast.TokenComment {
		j++
	}
	return scanToOpeningThenBrace(tokens, j)
}

func elseIfThenBraces(tokens []ast.Token, elseIfIdx int) (l, r int) {
	if elseIfIdx < 0 || elseIfIdx >= len(tokens) || tokens[elseIfIdx].Type != ast.TokenElseIf {
		return -1, -1
	}
	j := elseIfIdx + 1
	for j < len(tokens) && tokens[j].Type == ast.TokenComment {
		j++
	}
	paren := 0
	for j < len(tokens) {
		if tokens[j].Type == ast.TokenLBrace && paren == 0 {
			l = j
			r = matchingRBrace(tokens, l)
			return l, r
		}
		switch tokens[j].Type {
		case ast.TokenLParen:
			paren++
		case ast.TokenRParen:
			if paren > 0 {
				paren--
			}
		}
		j++
	}
	return -1, -1
}

func elseBlockBraces(tokens []ast.Token, elseIdx int) (l, r int) {
	if elseIdx < 0 || elseIdx >= len(tokens) || tokens[elseIdx].Type != ast.TokenElse {
		return -1, -1
	}
	j := elseIdx + 1
	for j < len(tokens) && tokens[j].Type == ast.TokenComment {
		j++
	}
	if j >= len(tokens) {
		return -1, -1
	}
	if tokens[j].Type == ast.TokenElseIf {
		return -1, -1
	}
	if tokens[j].Type != ast.TokenLBrace {
		return -1, -1
	}
	l = j
	r = matchingRBrace(tokens, l)
	return l, r
}

func ensureBlockBraces(tokens []ast.Token, ensureIdx int) (l, r int) {
	if ensureIdx < 0 || ensureIdx >= len(tokens) || tokens[ensureIdx].Type != ast.TokenEnsure {
		return -1, -1
	}
	j := ensureIdx + 1
	for j < len(tokens) && tokens[j].Type != ast.TokenLBrace {
		j++
	}
	if j >= len(tokens) || tokens[j].Type != ast.TokenLBrace {
		return -1, -1
	}
	l = j
	r = matchingRBrace(tokens, l)
	return l, r
}

// netBraceDepthBetween returns the brace nesting delta between token indices [from, to).
func netBraceDepthBetween(tokens []ast.Token, from, to int) int {
	d := 0
	if from < 0 {
		from = 0
	}
	if to > len(tokens) {
		to = len(tokens)
	}
	for i := from; i < to; i++ {
		switch tokens[i].Type {
		case ast.TokenLBrace:
			d++
		case ast.TokenRBrace:
			d--
		}
	}
	return d
}

func isStmtAtBlockLevel(tokens []ast.Token, blockLBrace, stmtKeywordIdx int) bool {
	if stmtKeywordIdx <= blockLBrace || stmtKeywordIdx >= len(tokens) {
		return false
	}
	return netBraceDepthBetween(tokens, blockLBrace+1, stmtKeywordIdx) == 0
}

func findNthBlockLevelKeyword(tokens []ast.Token, blockLBrace, bodyR int, typ ast.TokenIdent, n int) int {
	seen := 0
	for i := blockLBrace + 1; i < bodyR && i < len(tokens); i++ {
		if tokens[i].Type != typ {
			continue
		}
		if !isStmtAtBlockLevel(tokens, blockLBrace, i) {
			continue
		}
		if seen == n {
			return i
		}
		seen++
	}
	return -1
}

// deepestScopeInBlock finds the innermost scope node for a statement list (function / if body / else).
func deepestScopeInBlock(body []ast.Node, tokens []ast.Token, tokIdx, bodyL, bodyR int, tc *typechecker.TypeChecker) ast.Node {
	ifIdx := 0
	ensureIdx := 0
	for _, stmt := range body {
		switch st := stmt.(type) {
		case ast.IfNode:
			idx := findNthBlockLevelKeyword(tokens, bodyL, bodyR, ast.TokenIf, ifIdx)
			ifIdx++
			if idx < 0 {
				continue
			}
			if n := matchIfChainFrom(st, tokens, tokIdx, idx, bodyR, tc); n != nil {
				return n
			}
		case *ast.IfNode:
			idx := findNthBlockLevelKeyword(tokens, bodyL, bodyR, ast.TokenIf, ifIdx)
			ifIdx++
			if idx < 0 {
				continue
			}
			if n := matchIfChainFrom(*st, tokens, tokIdx, idx, bodyR, tc); n != nil {
				return n
			}
		case ast.EnsureNode:
			idx := findNthBlockLevelKeyword(tokens, bodyL, bodyR, ast.TokenEnsure, ensureIdx)
			ensureIdx++
			if idx < 0 || st.Block == nil {
				continue
			}
			lb, rb := ensureBlockBraces(tokens, idx)
			if lb > 0 && tokIdx > lb && tokIdx < rb {
				_ = tc.RestoreScope(st)
				_ = tc.RestoreScope(st.Block)
				return st.Block
			}
		}
	}
	return nil
}

func matchIfChainFrom(ifn ast.IfNode, tokens []ast.Token, tokIdx, ifIdx, limit int, tc *typechecker.TypeChecker) ast.Node {
	l, r := ifThenBraces(tokens, ifIdx)
	if l < 0 {
		return nil
	}
	if tokIdx > l && tokIdx < r {
		inner := deepestScopeInBlock(ifn.Body, tokens, tokIdx, l, r, tc)
		if inner != nil {
			return inner
		}
		if err := tc.RestoreScope(ifn); err == nil {
			return ifn
		}
		return nil
	}
	next := r + 1
	for _, ei := range ifn.ElseIfs {
		eiIdx := findFirstToken(tokens, next, limit, ast.TokenElseIf)
		if eiIdx < 0 {
			break
		}
		l, r = elseIfThenBraces(tokens, eiIdx)
		if l < 0 {
			break
		}
		if tokIdx > l && tokIdx < r {
			inner := deepestScopeInBlock(ei.Body, tokens, tokIdx, l, r, tc)
			if inner != nil {
				return inner
			}
			if err := tc.RestoreScope(ei); err == nil {
				return ei
			}
			return nil
		}
		next = r + 1
	}
	if ifn.Else != nil {
		elseIdx := findFirstToken(tokens, next, limit, ast.TokenElse)
		if elseIdx >= 0 {
			l, r = elseBlockBraces(tokens, elseIdx)
			if l > 0 && tokIdx > l && tokIdx < r {
				inner := deepestScopeInBlock(ifn.Else.Body, tokens, tokIdx, l, r, tc)
				if inner != nil {
					return inner
				}
				if err := tc.RestoreScope(ifn.Else); err == nil {
					return ifn.Else
				}
			}
		}
	}
	return nil
}

func deepestScopeInFunction(fn ast.FunctionNode, tokens []ast.Token, tokIdx int, bodyL, bodyR int, tc *typechecker.TypeChecker) ast.Node {
	_ = tc.RestoreScope(fn)
	ifIdx := 0
	ensureIdx := 0
	for _, stmt := range fn.Body {
		switch st := stmt.(type) {
		case ast.IfNode:
			idx := findNthBlockLevelKeyword(tokens, bodyL, bodyR, ast.TokenIf, ifIdx)
			ifIdx++
			if idx < 0 {
				continue
			}
			if n := matchIfChainFrom(st, tokens, tokIdx, idx, bodyR, tc); n != nil {
				return n
			}
		case *ast.IfNode:
			idx := findNthBlockLevelKeyword(tokens, bodyL, bodyR, ast.TokenIf, ifIdx)
			ifIdx++
			if idx < 0 {
				continue
			}
			if n := matchIfChainFrom(*st, tokens, tokIdx, idx, bodyR, tc); n != nil {
				return n
			}
		case ast.EnsureNode:
			idx := findNthBlockLevelKeyword(tokens, bodyL, bodyR, ast.TokenEnsure, ensureIdx)
			ensureIdx++
			if idx < 0 || st.Block == nil {
				continue
			}
			lb, rb := ensureBlockBraces(tokens, idx)
			if lb > 0 && tokIdx > lb && tokIdx < rb {
				_ = tc.RestoreScope(st)
				_ = tc.RestoreScope(st.Block)
				return st.Block
			}
		}
	}
	_ = tc.RestoreScope(fn)
	return fn
}

func findInnermostScopeNode(nodes []ast.Node, tokens []ast.Token, tokIdx int, tc *typechecker.TypeChecker) ast.Node {
	for _, n := range nodes {
		switch v := n.(type) {
		case ast.FunctionNode:
			lb, rb := functionBodyBraces(tokens, v)
			if lb < 0 {
				continue
			}
			if pOpen, pClose, ok := paramListParenRange(tokens, v); ok && tokIdx > pOpen && tokIdx < pClose {
				return v
			}
			if tokIdx <= lb || tokIdx >= rb {
				continue
			}
			return deepestScopeInFunction(v, tokens, tokIdx, lb, rb, tc)
		case *ast.FunctionNode:
			lb, rb := functionBodyBraces(tokens, *v)
			if lb < 0 {
				continue
			}
			if pOpen, pClose, ok := paramListParenRange(tokens, *v); ok && tokIdx > pOpen && tokIdx < pClose {
				return v
			}
			if tokIdx <= lb || tokIdx >= rb {
				continue
			}
			return deepestScopeInFunction(*v, tokens, tokIdx, lb, rb, tc)
		case ast.TypeGuardNode:
			if n := typeGuardScopeForPosition(v, tokens, tokIdx, tc); n != nil {
				return n
			}
		case *ast.TypeGuardNode:
			if n := typeGuardScopeForPosition(*v, tokens, tokIdx, tc); n != nil {
				return n
			}
		}
	}
	return nil
}

func typeGuardBodyBraces(tokens []ast.Token, guardName string) (l, r int) {
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Type != ast.TokenIs || braceDepthAtIndex(tokens, i) != 0 {
			continue
		}
		if i+1 >= len(tokens) || tokens[i+1].Type != ast.TokenLParen {
			continue
		}
		closeParen := skipBalancedParens(tokens, i+1)
		if closeParen < 0 {
			continue
		}
		k := closeParen + 1
		for k < len(tokens) && tokens[k].Type == ast.TokenComment {
			k++
		}
		if k >= len(tokens) || tokens[k].Type != ast.TokenIdentifier || tokens[k].Value != guardName {
			continue
		}
		k++
		if k < len(tokens) && tokens[k].Type == ast.TokenLParen {
			closeExtra := skipBalancedParens(tokens, k)
			if closeExtra < 0 {
				continue
			}
			k = closeExtra + 1
		}
		for k < len(tokens) && tokens[k].Type == ast.TokenComment {
			k++
		}
		if k >= len(tokens) || tokens[k].Type != ast.TokenLBrace {
			continue
		}
		lb := k
		rb := matchingRBrace(tokens, lb)
		if rb < 0 {
			continue
		}
		return lb, rb
	}
	return -1, -1
}

func typeGuardScopeForPosition(g ast.TypeGuardNode, tokens []ast.Token, tokIdx int, tc *typechecker.TypeChecker) ast.Node {
	name := string(g.Ident)
	lb, rb := typeGuardBodyBraces(tokens, name)
	if lb < 0 || tokIdx <= lb || tokIdx >= rb {
		return nil
	}
	if err := tc.RestoreScope(g); err != nil {
		return nil
	}
	return g
}

func lhsExpressionBeforeDot(lineToCursor string) string {
	s := strings.TrimRight(lineToCursor, " \t")
	if !strings.HasSuffix(s, ".") {
		return ""
	}
	s = strings.TrimSuffix(s, ".")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func memberCompletionsAfterDot(ctx *forstDocumentContext, pos LSPPosition, prefix string) []LSPCompletionItem {
	line := lineUpToCursor(ctx.Content, pos)
	lhs := lhsExpressionBeforeDot(line)
	if lhs == "" {
		return nil
	}
	vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(lhs)}}
	types, err := ctx.TC.InferExpressionTypeForCompletion(vn)
	if err != nil || len(types) == 0 {
		return nil
	}
	fields := ctx.TC.ListFieldNamesForType(types[0])
	pl := strings.ToLower(prefix)
	var out []LSPCompletionItem
	for _, f := range fields {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(f), pl) {
			continue
		}
		out = append(out, LSPCompletionItem{
			Label:            f,
			Kind:             LSPCompletionItemKindField,
			Detail:           "member",
			InsertText:       f,
			InsertTextFormat: LSPInsertTextFormatPlainText,
			FilterText:       f,
			SortText:         "1" + f,
		})
	}
	return out
}

func completionItemsFromKeywords(keywords []string, prefix string) []LSPCompletionItem {
	pl := strings.ToLower(prefix)
	var out []LSPCompletionItem
	for _, kw := range keywords {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(kw), pl) {
			continue
		}
		out = append(out, LSPCompletionItem{
			Label:            kw,
			Kind:             LSPCompletionItemKindKeyword,
			Detail:           "Forst keyword",
			Documentation:    kw,
			InsertText:       kw,
			InsertTextFormat: LSPInsertTextFormatPlainText,
			FilterText:       kw,
			SortText:         "0" + kw,
		})
	}
	return out
}

func topLevelSymbolCompletionItems(ctx *forstDocumentContext, prefix string) []LSPCompletionItem {
	tc := ctx.TC
	pl := strings.ToLower(prefix)
	var out []LSPCompletionItem
	for id, sig := range tc.Functions {
		name := string(id)
		if strings.HasPrefix(name, "T_") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), pl) {
			continue
		}
		out = append(out, LSPCompletionItem{
			Label:            name,
			Kind:             LSPCompletionItemKindFunction,
			Detail:           tc.FormatFunctionSignatureDisplay(sig),
			InsertText:       name,
			InsertTextFormat: LSPInsertTextFormatPlainText,
			FilterText:       name,
			SortText:         "1" + name,
		})
	}
	for tid, def := range tc.Defs {
		name := string(tid)
		if strings.HasPrefix(name, "T_") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), pl) {
			continue
		}
		switch def.(type) {
		case ast.TypeDefNode:
			out = append(out, LSPCompletionItem{
				Label:            name,
				Kind:             LSPCompletionItemKindStruct,
				Detail:           "type",
				InsertText:       name,
				InsertTextFormat: LSPInsertTextFormatPlainText,
				FilterText:       name,
				SortText:         "1" + name,
			})
		case ast.TypeGuardNode, *ast.TypeGuardNode:
			out = append(out, LSPCompletionItem{
				Label:            name,
				Kind:             LSPCompletionItemKindMethod,
				Detail:           "type guard",
				InsertText:       name,
				InsertTextFormat: LSPInsertTextFormatPlainText,
				FilterText:       name,
				SortText:         "1" + name,
			})
		default:
			// hash-only or other defs — skip
		}
	}
	return out
}

func localVariableCompletionItems(ctx *forstDocumentContext, prefix string) []LSPCompletionItem {
	ids := ctx.TC.VisibleVariableLikeSymbols()
	pl := strings.ToLower(prefix)
	var out []LSPCompletionItem
	for _, id := range ids {
		name := string(id)
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), pl) {
			continue
		}
		out = append(out, LSPCompletionItem{
			Label:            name,
			Kind:             LSPCompletionItemKindVariable,
			Detail:           "local",
			InsertText:       name,
			InsertTextFormat: LSPInsertTextFormatPlainText,
			FilterText:       name,
			SortText:         "2" + name,
		})
	}
	return out
}

func dedupeCompletionItems(items []LSPCompletionItem) []LSPCompletionItem {
	seen := make(map[string]bool)
	var out []LSPCompletionItem
	for _, it := range items {
		if seen[it.Label] {
			continue
		}
		seen[it.Label] = true
		out = append(out, it)
	}
	return out
}

// forstPackageNameFromContent returns the package identifier from the first `package` line, or "".
func forstPackageNameFromContent(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "package" {
			return fields[1]
		}
		break
	}
	return ""
}

func (s *LSPServer) countOtherOpenFtURIs(exceptURI string) int {
	exceptCanon := canonicalFileURI(exceptURI)
	s.documentMu.RLock()
	defer s.documentMu.RUnlock()
	seen := make(map[string]struct{}, len(s.openDocuments))
	n := 0
	for u := range s.openDocuments {
		cu := canonicalFileURI(u)
		if _, ok := seen[cu]; ok {
			continue
		}
		seen[cu] = struct{}{}
		if cu == exceptCanon {
			continue
		}
		if isForstDocumentURI(cu) {
			n++
		}
	}
	return n
}

func uriDisplayBasename(uri string) string {
	return filepath.Base(filePathFromDocumentURI(uri))
}

// crossBufferTopLevelCompletionItems adds top-level func/type/guard symbols from other open .ft buffers in the same package.
func (s *LSPServer) crossBufferTopLevelCompletionItems(currentURI, pkg, prefix string) []LSPCompletionItem {
	if pkg == "" {
		return nil
	}
	currentCanon := canonicalFileURI(currentURI)
	s.documentMu.RLock()
	seen := make(map[string]struct{}, len(s.openDocuments))
	uris := make([]string, 0, len(s.openDocuments))
	for u := range s.openDocuments {
		cu := canonicalFileURI(u)
		if _, ok := seen[cu]; ok {
			continue
		}
		seen[cu] = struct{}{}
		if cu == currentCanon {
			continue
		}
		if isForstDocumentURI(cu) {
			uris = append(uris, cu)
		}
	}
	s.documentMu.RUnlock()

	var out []LSPCompletionItem
	for _, ou := range uris {
		octx, ok := s.peerDocumentContextForCompletion(ou)
		if !ok || octx == nil {
			continue
		}
		if forstPackageNameFromContent(octx.Content) != pkg {
			continue
		}
		base := uriDisplayBasename(ou)
		for _, it := range topLevelSymbolCompletionItems(octx, prefix) {
			it.SortText = "4" + it.Label
			if it.Detail != "" {
				it.Detail = fmt.Sprintf("%s · %s", it.Detail, base)
			} else {
				it.Detail = base
			}
			out = append(out, it)
		}
	}
	return out
}

// getCompletionsForPosition returns merged keyword, semantic, local, member, and same-package open-buffer completions.
// The bool is LSP isIncomplete: true when other open .ft buffers exist (index is open-buffers-only, not full workspace).
func (s *LSPServer) getCompletionsForPosition(uri string, position LSPPosition, reqCtx *completionRequestContext) ([]LSPCompletionItem, bool) {
	filePath := filePathFromDocumentURI(uri)
	if !strings.HasSuffix(filePath, ".ft") {
		return nil, false
	}

	otherOpen := s.countOtherOpenFtURIs(uri) > 0

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		return nil, false
	}

	z := inferCompletionZone(ctx.Tokens, position, ctx.Content, reqCtx)
	if z == zoneUnknown {
		return nil, false
	}
	prefix := identifierPrefixAt(ctx.Content, position)

	if ctx.ParseErr != nil || ctx.TC == nil {
		items := completionItemsFromKeywords(keywordsForZone(z), prefix)
		pkg := forstPackageNameFromContent(ctx.Content)
		if pkg != "" && (z == zoneTopLevel || z == zoneInsideBlock) && ctx.PackageMerge == nil {
			items = append(items, s.crossBufferTopLevelCompletionItems(uri, pkg, prefix)...)
		}
		return dedupeCompletionItems(items), otherOpen
	}

	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, position)
	scopeNode := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC)
	if scopeNode != nil {
		_ = ctx.TC.RestoreScope(scopeNode)
	} else {
		_ = ctx.TC.RestoreScope(nil)
	}

	if z == zoneMemberAfterDot {
		return memberCompletionsAfterDot(ctx, position, prefix), false
	}

	var items []LSPCompletionItem
	items = append(items, completionItemsFromKeywords(keywordsForZone(z), prefix)...)
	items = append(items, topLevelSymbolCompletionItems(ctx, prefix)...)
	items = append(items, localVariableCompletionItems(ctx, prefix)...)
	pkg := forstPackageNameFromContent(ctx.Content)
	if (z == zoneTopLevel || z == zoneInsideBlock) && ctx.PackageMerge == nil {
		items = append(items, s.crossBufferTopLevelCompletionItems(uri, pkg, prefix)...)
	}
	return dedupeCompletionItems(items), otherOpen
}
