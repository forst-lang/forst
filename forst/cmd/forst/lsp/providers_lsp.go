package lsp

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/astwalk"
	"forst/internal/hoverdoc"
	"forst/internal/typechecker"
)

func withScopeHoverMarkdown(tc *typechecker.TypeChecker, nodes []ast.Node, tok *ast.Token) string {
	if tc == nil || tok == nil || tok.Type != ast.TokenWith {
		return ""
	}
	chain := collectWithChainContainingPosition(nodes, tok.Line, tok.Column)
	if len(chain) == 0 {
		return ""
	}
	labels, err := tc.EffectiveScopeKeyLabels(chain)
	if err != nil {
		return ""
	}
	if len(labels) == 0 {
		return hoverdoc.Section("Effective scope") + "\n\n" + hoverdoc.ForstBlock("// empty")
	}
	parts := make([]string, len(labels))
	for i, l := range labels {
		if l.Shadowed {
			parts[i] = l.Key + " // shadows outer"
		} else {
			parts[i] = l.Key
		}
	}
	return hoverdoc.Section("Effective scope") + "\n\n" + hoverdoc.ForstBlock(strings.Join(parts, "\n"))
}

func collectWithChainContainingPosition(nodes []ast.Node, line, col int) []ast.WithNode {
	var chain []ast.WithNode
	astwalk.WalkStmtsContaining(nodes, line, col, astwalk.StmtVisitor{
		OnWith: func(w ast.WithNode) bool {
			chain = append(chain, w)
			return true
		},
	})
	return chain
}

type withWiringCursor struct {
	inWiringShape bool
	afterColon    bool
	fieldKey      string
}

// withWiringBraceRange returns token indices [open, close] for the wiring shape `{ ... }` after `with`.
func withWiringBraceRange(tokens []ast.Token, withIdx int) (open, closeIdx int, ok bool) {
	j := withIdx + 1
	for j < len(tokens) && tokens[j].Type == ast.TokenComment {
		j++
	}
	if j >= len(tokens) || tokens[j].Type != ast.TokenLBrace {
		return -1, -1, false
	}
	closeIdx = matchingRBrace(tokens, j)
	if closeIdx < 0 {
		return -1, -1, false
	}
	return j, closeIdx, true
}

func detectWithWiringCursor(tokens []ast.Token, pos LSPPosition, content string) (withWiringCursor, bool) {
	idx := tokenIndexAtLSPPosition(tokens, pos)
	if idx < 0 {
		return withWiringCursor{}, false
	}
	for wi := range tokens {
		if tokens[wi].Type != ast.TokenWith {
			continue
		}
		open, closeIdx, ok := withWiringBraceRange(tokens, wi)
		if !ok {
			continue
		}
		if idx < open || idx > closeIdx {
			continue
		}
		cur := withWiringCursor{inWiringShape: true}
		line := lineUpToCursor(content, pos)
		colon := strings.LastIndex(line, ":")
		if colon >= 0 {
			cur.afterColon = true
			before := strings.TrimSpace(line[:colon])
			cur.fieldKey = lastIdentifierToken(before)
		}
		return cur, true
	}
	return withWiringCursor{}, false
}

func lastIdentifierToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var last string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == '{' || r == ',' || r == '(' || r == ')' || unicode.IsSpace(r)
	}) {
		if part != "" {
			last = part
		}
	}
	return last
}

func completionItemsFromStrings(labels []string, prefix string, sortPrefix string) []LSPCompletionItem {
	var out []LSPCompletionItem
	for _, label := range labels {
		if prefix != "" && !strings.HasPrefix(label, prefix) {
			continue
		}
		out = append(out, LSPCompletionItem{
			Label:    label,
			Kind:     6, // Variable — contract roots / impl types
			SortText: sortPrefix + label,
		})
	}
	return out
}

func providersWiringCompletionItems(ctx *forstDocumentContext, pos LSPPosition, prefix string) []LSPCompletionItem {
	if ctx == nil || ctx.TC == nil {
		return nil
	}
	cur, ok := detectWithWiringCursor(ctx.Tokens, pos, ctx.Content)
	if !ok || !cur.inWiringShape {
		return nil
	}
	if cur.afterColon && cur.fieldKey != "" {
		impls := ctx.TC.ProviderImplTypeNamesForContract(cur.fieldKey)
		return completionItemsFromStrings(impls, prefix, "2")
	}
	keys := ctx.TC.KnownProviderRootKeys()
	return completionItemsFromStrings(keys, prefix, "2")
}

func detectUseCompletionLine(line string) (afterColon bool, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "use ") {
		return false, false
	}
	rest := strings.TrimPrefix(trimmed, "use ")
	if strings.Contains(rest, ":") {
		return true, true
	}
	return false, true
}

func effectiveProviderContractLabels(ctx *forstDocumentContext, pos LSPPosition) []string {
	if ctx == nil || ctx.TC == nil {
		return nil
	}
	line1 := pos.Line + 1
	col1 := pos.Character + 1
	chain := collectWithChainContainingPosition(ctx.Nodes, line1, col1)
	if len(chain) > 0 {
		labels, err := ctx.TC.EffectiveScopeKeyLabels(chain)
		if err == nil && len(labels) > 0 {
			out := make([]string, len(labels))
			for i, l := range labels {
				out[i] = l.Key
			}
			return out
		}
	}
	return ctx.TC.KnownProviderRootKeys()
}

func providersUseCompletionItems(ctx *forstDocumentContext, pos LSPPosition, prefix string) []LSPCompletionItem {
	if ctx == nil || ctx.TC == nil {
		return nil
	}
	line := lineUpToCursor(ctx.Content, pos)
	afterColon, ok := detectUseCompletionLine(line)
	if !ok {
		return nil
	}
	var labels []string
	if afterColon {
		labels = ctx.TC.KnownProviderRootKeys()
	} else {
		labels = effectiveProviderContractLabels(ctx, pos)
	}
	return completionItemsFromStrings(labels, prefix, "2")
}

type codeActionDiagnosticParam struct {
	Range   LSPRange `json:"range"`
	Message string   `json:"message"`
	Code    string   `json:"code"`
}

func providersQuickFixActions(uri, content string, diags []codeActionDiagnosticParam) []LSPCodeAction {
	var out []LSPCodeAction
	for _, d := range diags {
		if !isUnknownWiringKeyDiagnostic(d) {
			continue
		}
		edit := lineDeleteEdit(content, d.Range)
		out = append(out, LSPCodeAction{
			Title: "Remove unknown wiring key",
			Kind:  "quickfix",
			Edit: &LSPWorkspaceEdit{
				Changes: map[string][]LSPTextEdit{
					uri: []LSPTextEdit{edit},
				},
			},
		})
	}
	return out
}

func isUnknownWiringKeyDiagnostic(d codeActionDiagnosticParam) bool {
	if d.Code == "providers-unknown-key" {
		return true
	}
	return strings.Contains(d.Message, "unknown wiring key")
}

func lineDeleteEdit(content string, diagRange LSPRange) LSPTextEdit {
	lines := strings.Split(content, "\n")
	line := diagRange.Start.Line
	if line < 0 || line >= len(lines) {
		return LSPTextEdit{Range: diagRange, NewText: ""}
	}
	endLine := line + 1
	endChar := 0
	if endLine >= len(lines) {
		endLine = line
		endChar = utf8.RuneCountInString(lines[line])
	}
	return LSPTextEdit{
		Range: LSPRange{
			Start: LSPPosition{Line: line, Character: 0},
			End:   LSPPosition{Line: endLine, Character: endChar},
		},
		NewText: "",
	}
}
