package lsp

import (
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
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

func TestLSPDebugger_ConvertDebugEventToHover(t *testing.T) {
	t.Parallel()
	ld := NewLSPDebugger(nil, "file:///h.ft")
	h := ld.ConvertDebugEventToHover(DebugEvent{
		Phase:     PhaseTypechecker,
		EventType: "t",
		Message:   "m",
		Line:      2,
		Function:  "Fn",
		TypeInfo:  &TypeInfo{ExpectedType: "int", ActualType: "str", InferredType: "inf"},
		Scope:     &ScopeInfo{FunctionName: "G", Variables: map[string]string{"v": "T"}},
		Error:     &ErrorInfo{Message: "e", Suggestions: []string{"fix"}},
	})
	if !strings.Contains(h.Contents.Value, "Fn") || !strings.Contains(h.Contents.Value, "Expected Type") {
		t.Fatalf("hover: %q", h.Contents.Value)
	}
	if h.Range == nil || h.Range.Start.Line != 1 {
		t.Fatalf("range start = %+v", h.Range)
	}
}

func TestLSPDebugger_ConvertDebugEventToListItem(t *testing.T) {
	t.Parallel()
	ld := NewLSPDebugger(nil, "")
	c := ld.ConvertDebugEventToCompletion(DebugEvent{
		Phase:     PhaseParser,
		EventType: "ignored_when_function_set",
		Message:   "body",
		Function:  "MyFn",
		TypeInfo:  &TypeInfo{ExpectedType: "want"},
		Error:     &ErrorInfo{Message: "err"},
	})
	if c.Label != "MyFn" {
		t.Fatalf("label = %q", c.Label)
	}
	if !strings.Contains(c.Documentation, "err") {
		t.Fatalf("doc = %q", c.Documentation)
	}
}

func TestLSPDebugger_ProcessDebugEvents_GetLSPOutput_notifications(t *testing.T) {
	t.Parallel()
	cd := NewCompilerDebugger(true)
	dbg := cd.GetDebugger(PhaseParser, "/tmp/proc.ft")
	dbg.LogType("infer", "type line", &TypeInfo{ExpectedType: "int", ActualType: "string", InferredType: "x"})
	dbg.LogEvent(EventFunctionParsed, "fn done", nil)

	ld := NewLSPDebugger(cd, "file:///tmp/proc.ft")
	if err := ld.ProcessDebugEvents(); err != nil {
		t.Fatal(err)
	}
	if len(ld.GetDiagnostics()) == 0 {
		t.Fatal("expected diagnostics")
	}
	if len(ld.GetHovers()) == 0 {
		t.Fatal("expected hovers from TypeInfo event")
	}
	if len(ld.GetCompletions()) == 0 {
		t.Fatal("expected completions from function_parsed")
	}

	raw, err := ld.GetLSPOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"uri"`) {
		t.Fatalf("json: %s", raw)
	}

	n := ld.CreateDiagnosticsNotification()
	if n.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("method = %q", n.Method)
	}

	hn := ld.CreateHoverNotification(LSPPosition{Line: 1, Character: 0})
	if hn.Method != "textDocument/hover" {
		t.Fatalf("method = %q", hn.Method)
	}
}

func TestCreateTypeErrorDiagnostic_and_CreateFunctionHover(t *testing.T) {
	t.Parallel()
	d := CreateTypeErrorDiagnostic("file:///t.ft", 3, "int", "string", "ctx")
	if d.Code != ErrorCodeTypeMismatch || len(d.Tags) == 0 {
		t.Fatalf("diag = %#v", d)
	}
	h := CreateFunctionHover("file:///t.ft", 2, "foo", []string{"a int"}, []string{"int"})
	if !strings.Contains(h.Contents.Value, "foo") {
		t.Fatalf("hover = %q", h.Contents.Value)
	}
}

func TestLSPDebugger_ProcessDebugEvents_GetAllOutputError(t *testing.T) {
	t.Parallel()
	ld := NewLSPDebugger(failingCompilerDebugger{}, "file:///x.ft")
	if err := ld.ProcessDebugEvents(); err == nil || !strings.Contains(err.Error(), "failed to get debug output") {
		t.Fatalf("err = %v", err)
	}
}

type failingCompilerDebugger struct{}

func (failingCompilerDebugger) GetDebugger(CompilerPhase, string) Debugger { return nil }

func (failingCompilerDebugger) GetAllOutput() (map[CompilerPhase][]byte, error) {
	return nil, errFakeDebugOutput
}

func (failingCompilerDebugger) PrintAllSummaries() {}

func (failingCompilerDebugger) LogWithStructuredDebug(*logrus.Logger, logrus.Level, CompilerPhase, string, string, string, map[string]interface{}) {
}

var errFakeDebugOutput = errors.New("output failed")
