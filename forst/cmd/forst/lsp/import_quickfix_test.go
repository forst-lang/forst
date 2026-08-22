package lsp

import (
	"testing"

	"forst/internal/importlocal"
)

func TestImportReservedLocalQuickFixActions(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		message   string
		content   string
		wantTitle string
		wantFix   string
	}{
		{
			name:      "go import",
			code:      "go-import-reserved-local",
			message:   "Go import local name \"type\" is a Forst keyword and cannot be used without an alias\n  import typePkg \"fmt\"",
			content:   "package main\nimport type \"fmt\"\n",
			wantTitle: `Add alias "typePkg" to Go import`,
			wantFix:   `import typePkg "fmt"`,
		},
		{
			name:      "node import",
			code:      "node-import-reserved-local",
			message:   "node import local name \"type\" is a Forst keyword and cannot be used without an alias\n  import typePkg \"./legacy/type.ts\" node",
			content:   "package main\nimport \"./legacy/type.ts\" node\n",
			wantTitle: `Add alias "typePkg" to node import`,
			wantFix:   `import typePkg "./legacy/type.ts" node`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := []codeActionDiagnosticParam{{
				Code:    tt.code,
				Message: tt.message,
				Range:   LSPRange{Start: LSPPosition{Line: 1}, End: LSPPosition{Line: 1, Character: 20}},
			}}
			acts := importReservedLocalQuickFixActions("file:///main.ft", tt.content, diags)
			if len(acts) != 1 {
				t.Fatalf("expected 1 action, got %d", len(acts))
			}
			if acts[0].Title != tt.wantTitle {
				t.Fatalf("title = %q, want %q", acts[0].Title, tt.wantTitle)
			}
			edit := acts[0].Edit.Changes["file:///main.ft"][0]
			if edit.NewText != tt.wantFix {
				t.Fatalf("edit = %q, want %q", edit.NewText, tt.wantFix)
			}
		})
	}
}

func TestExtractImportFixLine(t *testing.T) {
	goMsg := "Go import local name \"type\" is a Forst keyword and cannot be used without an alias\n  import typePkg \"fmt\""
	if got := extractImportFixLine(goMsg, importlocal.KindGo); got != `import typePkg "fmt"` {
		t.Fatalf("go fix = %q", got)
	}
	nodeMsg := "node import local name \"type\" is a Forst keyword and cannot be used without an alias\n  import typePkg \"./legacy/type.ts\" node"
	if got := extractImportFixLine(nodeMsg, importlocal.KindNode); got != `import typePkg "./legacy/type.ts" node` {
		t.Fatalf("node fix = %q", got)
	}
}
