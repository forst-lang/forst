package lsp

import (
	"forst/internal/ast"
	"forst/internal/hoverdoc"
	"forst/internal/typechecker"
)

func lspLocationFromASTSpan(uri string, span ast.SourceSpan) LSPLocation {
	return LSPLocation{
		URI:   uri,
		Range: lspRangeFromASTSpan(span),
	}
}

func lspLocationPtrFromASTSpan(uri string, span ast.SourceSpan) *LSPLocation {
	if !span.IsSet() {
		return nil
	}
	loc := lspLocationFromASTSpan(uri, span)
	return &loc
}

func spanFromToken(tok *ast.Token) ast.SourceSpan {
	if tok == nil {
		return ast.SourceSpan{}
	}
	return ast.SpanFromToken(*tok)
}

// definingLocationForLabel resolves goto/break/continue label uses and label decls to the decl span.
func definingLocationForLabel(tc *typechecker.TypeChecker, uri string, tokens []ast.Token, tok *ast.Token) *LSPLocation {
	if tc == nil || tok == nil || tok.Type != ast.TokenIdentifier {
		return nil
	}
	b := tc.LabelBindingAtSpan(spanFromToken(tok))
	if b == nil && isLabelContext(tokens, tok) {
		for _, lb := range tc.LabelsInScopeAt(tok.Line) {
			if lb.Name == ast.Identifier(tok.Value) && lb.DeclSpan.IsSet() {
				return lspLocationPtrFromASTSpan(uri, lb.DeclSpan)
			}
		}
		return nil
	}
	if b == nil {
		return nil
	}
	return lspLocationPtrFromASTSpan(uri, b.DeclSpan)
}

func isLabelContext(tokens []ast.Token, tok *ast.Token) bool {
	idx := -1
	for i := range tokens {
		if tokens[i].Line == tok.Line && tokens[i].Column == tok.Column && tokens[i].Value == tok.Value {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	if idx+1 < len(tokens) && tokens[idx+1].Type == ast.TokenColon {
		return true
	}
	prev := idx - 1
	for prev >= 0 && tokens[prev].Type == ast.TokenComment {
		prev--
	}
	if prev < 0 {
		return false
	}
	switch tokens[prev].Type {
	case ast.TokenGoto, ast.TokenBreak, ast.TokenContinue:
		return true
	default:
		return false
	}
}

func labelReferences(uri string, tc *typechecker.TypeChecker, tokens []ast.Token, tok *ast.Token, includeDecl bool) []LSPLocation {
	if tc == nil || tok == nil {
		return nil
	}
	b := tc.LabelBindingAtSpan(spanFromToken(tok))
	if b == nil && isLabelContext(tokens, tok) {
		for _, lb := range tc.LabelsInScopeAt(tok.Line) {
			if lb.Name == ast.Identifier(tok.Value) {
				cp := lb
				b = &cp
				break
			}
		}
	}
	if b == nil {
		return nil
	}
	var locs []LSPLocation
	if includeDecl && b.DeclSpan.IsSet() {
		locs = append(locs, lspLocationFromASTSpan(uri, b.DeclSpan))
	}
	for _, u := range b.UseSpans {
		if u.IsSet() {
			locs = append(locs, lspLocationFromASTSpan(uri, u))
		}
	}
	return locs
}

func labelHoverMarkdown(tc *typechecker.TypeChecker, tokens []ast.Token, tok *ast.Token) string {
	if tc == nil || tok == nil {
		return ""
	}
	b := tc.LabelBindingAtSpan(spanFromToken(tok))
	if b == nil && isLabelContext(tokens, tok) {
		for _, lb := range tc.LabelsInScopeAt(tok.Line) {
			if lb.Name == ast.Identifier(tok.Value) {
				cp := lb
				b = &cp
				break
			}
		}
	}
	if b == nil {
		return ""
	}
	kind := "label"
	if b.IsFor {
		kind = "for label"
	}
	return hoverdoc.Section(kind) + "\n\n" + hoverdoc.ForstBlock(string(b.Name)+":")
}

// afterGotoBreakContinue reports whether the completion position is the label slot after goto/break/continue.
func afterGotoBreakContinue(tokens []ast.Token, tokIdx int) bool {
	if tokIdx < 0 || len(tokens) == 0 {
		return false
	}
	i := tokIdx
	if i >= len(tokens) {
		i = len(tokens) - 1
	}
	// Walk back over the identifier being completed (if any).
	if i >= 0 && tokens[i].Type == ast.TokenIdentifier {
		i--
	}
	for i >= 0 && tokens[i].Type == ast.TokenComment {
		i--
	}
	if i < 0 {
		return false
	}
	switch tokens[i].Type {
	case ast.TokenGoto, ast.TokenBreak, ast.TokenContinue:
		return true
	default:
		return false
	}
}

func labelCompletionItems(tc *typechecker.TypeChecker, line int, prefix string) []LSPCompletionItem {
	if tc == nil {
		return nil
	}
	var out []LSPCompletionItem
	seen := map[string]bool{}
	for _, b := range tc.LabelsInScopeAt(line) {
		name := string(b.Name)
		if seen[name] {
			continue
		}
		if prefix != "" && !hasIdentifierPrefix(name, prefix) {
			continue
		}
		seen[name] = true
		detail := "label"
		if b.IsFor {
			detail = "for label"
		}
		out = append(out, LSPCompletionItem{
			Label:    name,
			Kind:     LSPCompletionItemKindVariable,
			Detail:   detail,
			SortText: "0" + name,
		})
	}
	return out
}

func hasIdentifierPrefix(name, prefix string) bool {
	if prefix == "" {
		return true
	}
	if len(prefix) > len(name) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if name[i] != prefix[i] {
			return false
		}
	}
	return true
}
