package typechecker

import (
	"strings"
	"testing"

	"forst/internal/testutil"
)

func TestProvidersScope_nilWiringRejected(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }

func TestX(t *testing.T) {
	with { Logger: nil } {
		x := 1
	}
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	if err == nil {
		t.Fatal("expected error for nil wiring")
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "providers-nil-wiring" {
		t.Fatalf("expected providers-nil-wiring, got %T: %v", err, err)
	}
}

func TestProvidersScope_unknownWiringKeyRejected(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }

func TestX(t *testing.T) {
	with { NotAContract: &NopLogger {} } {
		x := 1
	}
}

type NopLogger = {}
func (NopLogger) info(msg String) {}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	if err == nil {
		t.Fatal("expected error for unknown wiring key")
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "providers-unknown-key" {
		t.Fatalf("expected providers-unknown-key, got %T: %v", err, err)
	}
}

func TestProvidersScope_withBlockSatisfiesLogger(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }

type NopLogger = {}
func (NopLogger) info(msg String) {}

func needsLogger() {
	use logger: Logger
}

func TestX(t *testing.T) {
	with { Logger: &NopLogger {} } {
		needsLogger()
	}
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
}

func TestProvidersScope_wiringTypeMismatch(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }
type Clock = { now(): Int }

type NopLogger = {}
func (NopLogger) info(msg String) {}

func TestX(t *testing.T) {
	with { Logger: &FakeClock { fixedMs: 1 } } {
		x := 1
	}
}

type FakeClock = { fixedMs: Int }
func (c FakeClock) now(): Int { return c.fixedMs }
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	if err == nil {
		t.Fatal("expected wiring type mismatch error")
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "providers-wiring-type" {
		t.Fatalf("expected providers-wiring-type, got %T: %v", err, err)
	}
	if !strings.Contains(diag.Msg, "Logger") {
		t.Fatalf("expected Logger in message: %q", diag.Msg)
	}
}
