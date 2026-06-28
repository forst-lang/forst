package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func TestProvidersWiringCompletion_contractKeys(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("providers_compl")), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "w.ft")
	const src = `package main

import "testing"

type Logger = { info(msg String) }
type Clock = { now(): Int }

func TestWiringKeys(t *testing.T) {
	with {
		Log
	} {
	}
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.setOpenDocument(uri, src)

	// Cursor on "Log" prefix inside wiring shape
	pos := LSPPosition{Line: 9, Character: 3}
	items, _ := s.getCompletionsForPosition(uri, pos, nil)
	var hasLogger bool
	for _, it := range items {
		if it.Label == "Logger" {
			hasLogger = true
		}
	}
	if !hasLogger {
		t.Fatalf("expected Logger in completions, got %v", items)
	}
}

func TestProvidersWiringCompletion_implTypesAfterColon(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("providers_compl")), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "w.ft")
	const src = `package main

import "testing"

type Logger = { info(msg String) }
type NopLogger = {}
func (NopLogger) info(msg String) {}

func TestImpl(t *testing.T) {
	with {
		Logger: Nop
	} {
	}
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.setOpenDocument(uri, src)

	pos := LSPPosition{Line: 9, Character: 11}
	items, _ := s.getCompletionsForPosition(uri, pos, nil)
	var hasNop bool
	for _, it := range items {
		if it.Label == "NopLogger" || it.Label == "&NopLogger" {
			hasNop = true
		}
	}
	if !hasNop {
		t.Fatalf("expected NopLogger impl in completions, got %v", items)
	}
}

func TestProvidersUseCompletion_contractAfterColon(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("providers_use")), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "u.ft")
	const src = `package main

import "testing"

type Logger = { info(msg String) }
type Clock = { now(): Int }
type NopLogger = {}
func (NopLogger) info(msg String) {}

func TestUseCompletion(t *testing.T) {
	with { Logger: &NopLogger {} } {
		use logger: Log
	}
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.setOpenDocument(uri, src)

	pos := LSPPosition{Line: 12, Character: 14}
	items, _ := s.getCompletionsForPosition(uri, pos, nil)
	var hasLogger bool
	for _, it := range items {
		if it.Label == "Logger" {
			hasLogger = true
		}
	}
	if !hasLogger {
		t.Fatalf("expected Logger contract in use completion, got %v", items)
	}
}

func TestHandleCodeAction_unknownWiringKeyQuickFix(t *testing.T) {
	t.Parallel()
	src := "package main\n\nimport \"testing\"\n\ntype Logger = { info(msg String) }\ntype NopLogger = {}\nfunc (NopLogger) info(msg String) {}\n\nfunc TestX(t *testing.T) {\n\twith { BadKey: &NopLogger {} } {\n\t}\n}\n"
	uri := mustFileURI(t, filepath.Join(t.TempDir(), "bad.ft"))
	s := NewLSPServer("8080", logrus.New())
	s.setOpenDocument(uri, src)

	diagRange := LSPRange{
		Start: LSPPosition{Line: 8, Character: 0},
		End:   LSPPosition{Line: 8, Character: 30},
	}
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]string{"uri": uri},
		"context": map[string]any{
			"only": []string{"quickfix"},
			"diagnostics": []map[string]any{
				{
					"range":   diagRange,
					"message": "unknown wiring key \"BadKey\"",
					"code":    "providers-unknown-key",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := s.handleCodeAction(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  json.RawMessage(params),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	actions, ok := resp.Result.([]any)
	if !ok || len(actions) != 1 {
		t.Fatalf("expected 1 quickfix, got %T %#v", resp.Result, resp.Result)
	}
	act, ok := actions[0].(LSPCodeAction)
	if !ok || act.Kind != "quickfix" {
		t.Fatalf("action = %#v", actions[0])
	}
}
