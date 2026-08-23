package lsp

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"forst/internal/importlocal"
)

func importReservedLocalQuickFixActions(uri, content string, diags []codeActionDiagnosticParam) []LSPCodeAction {
	var out []LSPCodeAction
	for _, d := range diags {
		kind, ok := quickFixKindForCode(d.Code)
		if !ok {
			continue
		}
		fix := extractImportFixLine(d.Message, kind)
		if fix == "" {
			continue
		}
		edit := replaceLineEdit(content, d.Range.Start.Line, fix)
		title := "Add import alias"
		if kind == importlocal.KindGo {
			title = "Add Go import alias"
		} else {
			title = "Add JS import alias"
		}
		if alias := aliasFromImportFix(fix, kind); alias != "" {
			if kind == importlocal.KindGo {
				title = fmt.Sprintf("Add alias %q to Go import", alias)
			} else {
				title = fmt.Sprintf("Add alias %q to JS import", alias)
			}
		}
		out = append(out, LSPCodeAction{
			Title: title,
			Kind:  "quickfix",
			Edit: &LSPWorkspaceEdit{
				Changes: map[string][]LSPTextEdit{
					uri: {edit},
				},
			},
		})
	}
	return out
}

func quickFixKindForCode(code string) (importlocal.Kind, bool) {
	switch code {
	case "go-import-reserved-local":
		return importlocal.KindGo, true
	case "js-import-reserved-local":
		return importlocal.KindBridge, true
	default:
		return 0, false
	}
}

func extractImportFixLine(msg string, kind importlocal.Kind) string {
	for _, line := range strings.Split(msg, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "import ") {
			continue
		}
		switch kind {
		case importlocal.KindBridge:
			if strings.HasSuffix(line, " js") {
				return line
			}
		case importlocal.KindGo:
			if strings.HasSuffix(line, " js") {
				continue
			}
			if strings.Contains(line, `"`) || strings.Contains(line, `'`) {
				return line
			}
		}
	}
	return ""
}

func aliasFromImportFix(fix string, kind importlocal.Kind) string {
	fix = strings.TrimSpace(fix)
	if !strings.HasPrefix(fix, "import ") {
		return ""
	}
	inner := strings.TrimPrefix(fix, "import ")
	if kind == importlocal.KindBridge {
		if !strings.HasSuffix(inner, " js") {
			return ""
		}
		inner = strings.TrimSuffix(inner, " js")
	}
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return ""
	}
	if strings.HasPrefix(inner, `"`) || strings.HasPrefix(inner, `'`) {
		return ""
	}
	if i := strings.IndexByte(inner, ' '); i >= 0 {
		return inner[:i]
	}
	return ""
}

func replaceLineEdit(content string, line0 int, newLine string) LSPTextEdit {
	lines := strings.Split(content, "\n")
	if line0 < 0 || line0 >= len(lines) {
		return LSPTextEdit{Range: LSPRange{}, NewText: newLine}
	}
	endLine := line0 + 1
	endChar := 0
	if endLine >= len(lines) {
		endLine = line0
		endChar = utf8.RuneCountInString(lines[line0])
	}
	return LSPTextEdit{
		Range: LSPRange{
			Start: LSPPosition{Line: line0, Character: 0},
			End:   LSPPosition{Line: endLine, Character: endChar},
		},
		NewText: newLine,
	}
}
