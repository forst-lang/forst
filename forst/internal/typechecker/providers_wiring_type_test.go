package typechecker

import (
	"testing"
)

func TestProviders_N7_wiringTypeMismatch(t *testing.T) {
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
	_, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected wiring type error")
	}
	diag, ok := err.(*Diagnostic)
	if !ok {
		t.Fatalf("expected Diagnostic, got %T", err)
	}
	if diag.Code != "providers-wiring-type" {
		t.Fatalf("code = %q, want providers-wiring-type", diag.Code)
	}
}

func TestProviders_N7_wiringShapeMismatch(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }

func TestX(t *testing.T) {
	with 42 {
		x := 1
	}
}
`
	_, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected wiring shape error")
	}
	diag, ok := err.(*Diagnostic)
	if !ok {
		t.Fatalf("expected Diagnostic, got %T", err)
	}
	if diag.Code != "providers-wiring-shape" {
		t.Fatalf("code = %q, want providers-wiring-shape", diag.Code)
	}
}
