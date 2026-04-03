package lsp

import (
	"encoding/json"
	"testing"
)

func TestLspLineIndex(t *testing.T) {
	t.Parallel()
	if got, want := lspLineIndex(0), 0; got != want {
		t.Errorf("lspLineIndex(0) = %d, want %d (unset line must not go negative)", got, want)
	}
	if got, want := lspLineIndex(1), 0; got != want {
		t.Errorf("lspLineIndex(1) = %d, want %d", got, want)
	}
	if got, want := lspLineIndex(2), 1; got != want {
		t.Errorf("lspLineIndex(2) = %d, want %d", got, want)
	}
}

func TestIsCompilerPhaseCompleteEvent(t *testing.T) {
	t.Parallel()
	if !isCompilerPhaseCompleteEvent(DebugEvent{EventType: EventParserComplete, Message: "Parsing completed"}) {
		t.Error("expected parser_complete to be filtered")
	}
	if isCompilerPhaseCompleteEvent(DebugEvent{EventType: EventParserError, Message: "bad"}) {
		t.Error("errors must not be filtered")
	}
}

func TestShouldSkipDebugEventAsDiagnostic_MessageFallback(t *testing.T) {
	t.Parallel()
	if !shouldSkipDebugEventAsDiagnostic(DebugEvent{Phase: PhaseParser, Message: "Parsing completed"}) {
		t.Error("expected message fallback when event_type missing")
	}
	if shouldSkipDebugEventAsDiagnostic(DebugEvent{Phase: PhaseParser, Message: "real error", Error: &ErrorInfo{Message: "x"}}) {
		t.Error("events with Error must not be skipped")
	}
}

func TestDebugEventsJSONRoundTripPreservesEventType(t *testing.T) {
	t.Parallel()
	events := []DebugEvent{
		{Phase: PhaseParser, EventType: EventParserComplete, Message: "Parsing completed"},
	}
	data, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	var back []DebugEvent
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != 1 || back[0].EventType != EventParserComplete {
		t.Fatalf("EventType = %q", back[0].EventType)
	}
	if !shouldSkipDebugEventAsDiagnostic(back[0]) {
		t.Fatal("filter should apply after JSON round-trip")
	}
}
