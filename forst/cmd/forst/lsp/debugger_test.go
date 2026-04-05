package lsp

import (
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestCompilerDebugger_GetDebugger_DisabledReturnsNil(t *testing.T) {
	t.Parallel()
	cd := NewCompilerDebugger(false)
	if d := cd.GetDebugger(PhaseLexer, "/tmp/a.ft"); d != nil {
		t.Fatalf("expected nil when debugger disabled, got %T", d)
	}
}

func TestCompilerDebugger_ResetStructuredOutputs_ClearsPhaseEvents(t *testing.T) {
	t.Parallel()
	cd := NewCompilerDebugger(true)
	dbg := cd.GetDebugger(PhaseParser, "/tmp/x.ft")
	dbg.LogError(EventParserError, "parse issue", &ErrorInfo{
		Code:     ErrorCodeInvalidSyntax,
		Message:  "oops",
		Severity: SeverityError,
	})
	out, err := cd.GetAllOutput()
	if err != nil {
		t.Fatal(err)
	}
	var before []DebugEvent
	if err := json.Unmarshal(out[PhaseParser], &before); err != nil {
		t.Fatal(err)
	}
	if len(before) == 0 {
		t.Fatal("expected events before reset")
	}

	cd.ResetStructuredOutputs()
	out2, err := cd.GetAllOutput()
	if err != nil {
		t.Fatal(err)
	}
	var after []DebugEvent
	if err := json.Unmarshal(out2[PhaseParser], &after); err != nil {
		t.Fatal(err)
	}
	if len(after) != 0 {
		t.Fatalf("expected empty parser output after reset, got %d events", len(after))
	}
}

func TestExtractFilename(t *testing.T) {
	t.Parallel()
	if got, want := extractFilename("a/b/c.ft"), "c.ft"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got, want := extractFilename("solo.ft"), "solo.ft"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got, want := extractFilename(""), ""; got != want {
		t.Fatalf("empty path: got %q want %q", got, want)
	}
}

func TestExtractPackagePath(t *testing.T) {
	t.Parallel()
	if got, want := extractPackagePath("pkg/sub/file.ft"), "pkg/sub"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got, want := extractPackagePath("only.ft"), "."; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStructuredDebugger_LogScope_LogAST_LogType_GetPhaseSummary(t *testing.T) {
	t.Parallel()
	ps := NewPackageStore()
	d := NewStructuredDebugger(PhaseParser, "/proj/a.ft", ps)
	d.LogScope(EventScopeEntered, "scope", &ScopeInfo{FunctionName: "F", Variables: map[string]string{"x": "Int"}})
	d.LogAST(EventNodeCreated, "ast", &ASTInfo{NodeType: "Func"})
	d.LogType("infer", "types", &TypeInfo{ExpectedType: "A", ActualType: "B", InferredType: "C"})
	sum := d.GetPhaseSummary()
	if sum["total_events"].(int) != 3 {
		t.Fatalf("total_events = %v", sum["total_events"])
	}
	if sum["file_id"] == "" {
		t.Fatal("expected file_id")
	}
}

func TestStructuredDebugger_PrintSummary_doesNotPanic(t *testing.T) {
	t.Parallel()
	d := NewStructuredDebugger(PhaseLexer, "/x/y.ft", NewPackageStore())
	d.LogEvent("e", "m", nil)
	d.PrintSummary()
}

func TestCompilerDebugger_PrintAllSummaries_LogWithStructuredDebug(t *testing.T) {
	t.Parallel()
	cd := NewCompilerDebugger(true)
	dbg := cd.GetDebugger(PhaseParser, "/tmp/p.ft")
	dbg.LogEvent("evt", "hello", map[string]interface{}{"k": 1})
	cd.PrintAllSummaries()

	log := logrus.New()
	cd.LogWithStructuredDebug(log, logrus.DebugLevel, PhaseParser, "/tmp/p.ft", "evt2", "msg2", map[string]interface{}{"a": 2})
	out, err := cd.GetAllOutput()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected phase output")
	}
}
