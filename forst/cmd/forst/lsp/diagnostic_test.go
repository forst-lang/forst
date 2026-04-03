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

func TestLSPDebugger_convertSeverityToLSP(t *testing.T) {
	t.Parallel()
	ld := NewLSPDebugger(nil, "")
	cases := []struct {
		info *ErrorInfo
		want LSPDiagnosticSeverity
	}{
		{nil, LSPDiagnosticSeverityInformation},
		{&ErrorInfo{Severity: SeverityError}, LSPDiagnosticSeverityError},
		{&ErrorInfo{Severity: SeverityWarning}, LSPDiagnosticSeverityWarning},
		{&ErrorInfo{Severity: SeverityInfo}, LSPDiagnosticSeverityInformation},
		{&ErrorInfo{Severity: SeverityDebug}, LSPDiagnosticSeverityHint},
		{&ErrorInfo{Severity: "unknown"}, LSPDiagnosticSeverityInformation},
	}
	for _, tc := range cases {
		if got := ld.convertSeverityToLSP(tc.info); got != tc.want {
			t.Fatalf("%+v: got %d want %d", tc.info, got, tc.want)
		}
	}
}

func TestLSPDebugger_positionInRange(t *testing.T) {
	t.Parallel()
	ld := NewLSPDebugger(nil, "")
	r := LSPRange{
		Start: LSPPosition{Line: 1, Character: 2},
		End:   LSPPosition{Line: 3, Character: 10},
	}
	if !ld.positionInRange(LSPPosition{Line: 2, Character: 0}, r) {
		t.Fatal("middle line should be in range")
	}
	if ld.positionInRange(LSPPosition{Line: 1, Character: 1}, r) {
		t.Fatal("same line before start char should be out")
	}
	if !ld.positionInRange(LSPPosition{Line: 1, Character: 2}, r) {
		t.Fatal("start boundary should be in")
	}
	if !ld.positionInRange(LSPPosition{Line: 3, Character: 10}, r) {
		t.Fatal("end boundary should be in")
	}
	if ld.positionInRange(LSPPosition{Line: 3, Character: 11}, r) {
		t.Fatal("past end char should be out")
	}
	if ld.positionInRange(LSPPosition{Line: 0, Character: 5}, r) {
		t.Fatal("line before range should be out")
	}
	if ld.positionInRange(LSPPosition{Line: 4, Character: 0}, r) {
		t.Fatal("line after range should be out")
	}
}
