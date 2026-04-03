package lsp

import (
	"testing"
)

func TestLSPDebugger_ConvertDebugEventToDiagnostic_LineSeverityAndCode(t *testing.T) {
	t.Parallel()
	ld := NewLSPDebugger(nil, "file:///doc.ft")
	ev := DebugEvent{
		Phase:     PhaseParser,
		Line:      3,
		Message:   "bad token",
		EventType: EventParserError,
		Error: &ErrorInfo{
			Code:     "E1",
			Message:  "detail",
			Severity: SeverityError,
		},
	}
	d := ld.ConvertDebugEventToDiagnostic(ev)
	if d.Range.Start.Line != 2 {
		t.Fatalf("LSP start line = %d, want 2 (1-based line 3)", d.Range.Start.Line)
	}
	if d.Range.End.Line != 2 {
		t.Fatalf("LSP end line = %d, want 2", d.Range.End.Line)
	}
	if d.Severity != LSPDiagnosticSeverityError {
		t.Fatalf("severity = %d, want error", d.Severity)
	}
	if d.Code != "E1" {
		t.Fatalf("code = %q", d.Code)
	}
	if d.Source != string(PhaseParser) {
		t.Fatalf("source = %q", d.Source)
	}
	if d.Message != "bad token" {
		t.Fatalf("message = %q", d.Message)
	}
}

func TestLSPDebugger_ConvertDebugEventToDiagnostic_NilErrorIsInformational(t *testing.T) {
	t.Parallel()
	ld := NewLSPDebugger(nil, "")
	d := ld.ConvertDebugEventToDiagnostic(DebugEvent{
		Phase:   PhaseLexer,
		Message: "status",
	})
	if d.Severity != LSPDiagnosticSeverityInformation {
		t.Fatalf("got severity %d", d.Severity)
	}
}
