package lsp

import (
	"strings"
	"unicode/utf8"
)

type codeActionDiagnosticParam struct {
	Range   LSPRange `json:"range"`
	Message string   `json:"message"`
	Code    string   `json:"code"`
}

func usablesQuickFixActions(uri, content string, diags []codeActionDiagnosticParam) []LSPCodeAction {
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
	if d.Code == "usables-unknown-key" {
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
