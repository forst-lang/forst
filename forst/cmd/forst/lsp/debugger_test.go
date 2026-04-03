package lsp

import (
	"encoding/json"
	"testing"
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
