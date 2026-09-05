package lsp

import (
	"encoding/json"
)

// reportFixQuickFixActions builds quickfixes from Diagnostic.data.fixes (no message scraping).
func reportFixQuickFixActions(uri string, diags []codeActionDiagnosticParam) []LSPCodeAction {
	var out []LSPCodeAction
	for _, d := range diags {
		for _, fix := range fixesFromDiagnosticData(d.Data) {
			rng := d.Range
			if fix.Range != nil {
				rng = *fix.Range
			}
			title := fix.Title
			if title == "" {
				title = "Apply fix"
			}
			out = append(out, LSPCodeAction{
				Title: title,
				Kind:  "quickfix",
				Edit: &LSPWorkspaceEdit{
					Changes: map[string][]LSPTextEdit{
						uri: {{Range: rng, NewText: fix.NewText}},
					},
				},
			})
		}
	}
	return out
}

func fixesFromDiagnosticData(data any) []lspFixPayload {
	if data == nil {
		return nil
	}
	// data may already be map[string]any from our publisher, or re-encoded JSON from the client.
	raw, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	var wrap struct {
		Fixes []lspFixPayload `json:"fixes"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil
	}
	return wrap.Fixes
}
