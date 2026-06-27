package typechecker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func moduleRootForUsablesTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

func parseAndCheck(t *testing.T, src string) (*TypeChecker, error) {
	t.Helper()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = moduleRootForUsablesTest(t)
	err = tc.CheckTypes(nodes)
	return tc, err
}

func usableRootNames(slots []UsableSlot) []string {
	out := make([]string, len(slots))
	for i, s := range slots {
		out[i] = string(s.RootIdent)
	}
	return out
}

func containsAll(haystack []string, needles ...string) bool {
	set := make(map[string]struct{}, len(haystack))
	for _, h := range haystack {
		set[h] = struct{}{}
	}
	for _, n := range needles {
		if _, ok := set[n]; !ok {
			return false
		}
	}
	return true
}

func TestUsables_N1_useSitesInFunctionUsables(t *testing.T) {
	src := `package main

type Token = { id: String }

type Logger = {
	info(msg String)
}

type Clock = {
	now(): Int
}

func expireToken(token Token) {
	use logger: Logger
	use clock: Clock
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	slots := tc.FunctionUsables["expireToken"]
	roots := usableRootNames(slots)
	if !containsAll(roots, "Logger", "Clock") {
		t.Fatalf("FunctionUsables(expireToken) = %v, want Logger and Clock", roots)
	}
}

func TestUsables_N3_transitiveCallPropagation(t *testing.T) {
	src := `package main

type Logger = {
	info(msg String)
}

func inner() {
	use logger: Logger
}

func outer() {
	inner()
}
`
	tc, err := parseAndCheck(t, src)
	roots := usableRootNames(tc.FunctionUsables["outer"])
	if !containsAll(roots, "Logger") {
		t.Fatalf("FunctionUsables(outer) = %v, want Logger from inner()", roots)
	}
	// Call without ambient is validated in TestUsables_diagnostic_unsatisfiedAtCallSite.
	_ = err
}

func TestUsables_N3_innerWithSubtractsLocallySatisfied(t *testing.T) {
	src := `package main

type Logger = {
	info(msg String)
}

type NopLogger = {}

func (NopLogger) info(msg String) {}

func inner() {
	use logger: Logger
}

func outer() {
	with { Logger: &NopLogger {} } {
		inner()
	}
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	slots := tc.FunctionUsables["outer"]
	if len(slots) != 0 {
		t.Fatalf("FunctionUsables(outer) = %v, want empty (Logger satisfied by inner with)", usableRootNames(slots))
	}
}

func TestUsables_N5_nestedWithMergeAndShadow(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }
type Clock = { now(): Int }

type NopLogger = {}
type FakeClock = { fixedMs: Int }

func (NopLogger) info(msg String) {}
func (c FakeClock) now(): Int { return c.fixedMs }

func expireToken() {
	use logger: Logger
	use clock: Clock
}

func TestNestedWith(t *testing.T) {
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 1 },
	} {
		with { Clock: &FakeClock { fixedMs: 2 } } {
			expireToken()
		}
	}
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	if len(tc.FunctionUsables["TestNestedWith"]) != 0 {
		t.Fatalf("TestNestedWith should have no boundary usables, got %v", usableRootNames(tc.FunctionUsables["TestNestedWith"]))
	}
}

func TestUsables_N8_aliasSharesSlot(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }
type AuditLogger = Logger

func f() {
	use x: AuditLogger
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	slots := tc.FunctionUsables["f"]
	if len(slots) != 1 {
		t.Fatalf("expected one slot, got %v", slots)
	}
	if slots[0].RootIdent != "Logger" {
		t.Fatalf("alias slot root = %q, want Logger", slots[0].RootIdent)
	}
}

func TestUsables_N10_verticalSliceExpireToken(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }
type Clock = { now(): Int }
type Token = { id: String, expiresAt: Int }

type NopLogger = {}
type FakeClock = { fixedMs: Int }

func (NopLogger) info(msg String) {}
func (c FakeClock) now(): Int { return c.fixedMs }

func expireToken(token Token) {
	use logger: Logger
	use clock: Clock
}

error Expired { tokenId: String }

func TestExpireToken(t *testing.T) {
	token := Token { id: "t1", expiresAt: 1000 }
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 2000 },
	} {
		expireToken(token)
	}
}
`
	_, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatalf("vertical slice should typecheck: %v", err)
	}
}

func TestUsables_diagnostic_unknownWiringKey(t *testing.T) {
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
	_, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected error for unknown wiring key")
	}
	diag, ok := err.(*Diagnostic)
	if !ok {
		t.Fatalf("expected Diagnostic, got %T: %v", err, err)
	}
	if diag.Code != "usables-unknown-key" {
		t.Fatalf("code = %q, want usables-unknown-key", diag.Code)
	}
}

func TestUsables_diagnostic_unsatisfiedAtCallSite(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }

func needsLogger() {
	use logger: Logger
}

func TestX(t *testing.T) {
	needsLogger()
}
`
	_, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected error for unsatisfied Logger")
	}
	diag, ok := err.(*Diagnostic)
	if !ok {
		t.Fatalf("expected Diagnostic, got %T: %v", err, err)
	}
	if diag.Code != "usables-unsatisfied" {
		t.Fatalf("code = %q, want usables-unsatisfied", diag.Code)
	}
	if !strings.Contains(diag.Msg, "Logger") {
		t.Fatalf("message should mention Logger: %q", diag.Msg)
	}
	if !strings.Contains(diag.Msg, "TestX → needsLogger") {
		t.Fatalf("message should include obligation chain: %q", diag.Msg)
	}
	if len(diag.Related) < 2 {
		t.Fatalf("expected related diagnostics for obligation chain, got %d", len(diag.Related))
	}
}

func TestUsables_diagnostic_unsatisfiedWiringRoot(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }

func needsLogger() {
	use logger: Logger
}

func TestX(t *testing.T) {
	with { Logger: &NopLogger {} } {
		needsLogger()
	}
	needsLogger()
}

type NopLogger = {}
func (NopLogger) info(msg String) {}
`
	_, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected error for second unsatisfied needsLogger() call")
	}
}

func TestUsables_warning_unusedWiringKey(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }
type Clock = { now(): Int }

type NopLogger = {}
type FakeClock = { fixedMs: Int }

func (NopLogger) info(msg String) {}
func (c FakeClock) now(): Int { return c.fixedMs }

func expireToken() {
	use logger: Logger
}

func TestX(t *testing.T) {
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 1 },
	} {
		expireToken()
	}
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	found := false
	for _, w := range tc.Warnings {
		if w.Code == "usables-unused-key" && strings.Contains(w.Msg, "Clock") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unused Clock warning, got warnings: %+v", tc.Warnings)
	}
}

func TestUsables_unusedKey_detectsCallInAssignment(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }
type Clock = { now(): Int }

type NopLogger = {}
type FakeClock = { fixedMs: Int }

func (NopLogger) info(msg String) {}
func (c FakeClock) now(): Int { return c.fixedMs }

func expireToken() {
	use logger: Logger
}

func TestAssignCall(t *testing.T) {
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 1 },
	} {
		_ := expireToken()
	}
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	for _, w := range tc.Warnings {
		if w.Code == "usables-unused-key" && strings.Contains(w.Msg, "Logger") {
			t.Fatalf("Logger should be required via assignment call, got warning: %s", w.Msg)
		}
	}
	foundClockUnused := false
	for _, w := range tc.Warnings {
		if w.Code == "usables-unused-key" && strings.Contains(w.Msg, "Clock") {
			foundClockUnused = true
			break
		}
	}
	if !foundClockUnused {
		t.Fatalf("expected unused Clock warning, got: %+v", tc.Warnings)
	}
}

func TestUsables_ADR041_rejectsAliasWiringKey(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }
type AuditLogger = Logger

type NopLogger = {}
func (NopLogger) info(msg String) {}

func TestX(t *testing.T) {
	with { AuditLogger: &NopLogger {} } {
		x := 1
	}
}
`
	_, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected error for alias wiring key")
	}
	diag, ok := err.(*Diagnostic)
	if !ok {
		t.Fatalf("expected Diagnostic, got %T", err)
	}
	if diag.Code != "usables-alias-key" {
		t.Fatalf("code = %q, want usables-alias-key", diag.Code)
	}
}

func TestUsables_ADR037_rejectsNilWiring(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }

func TestX(t *testing.T) {
	with { Logger: nil } {
		x := 1
	}
}
`
	_, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected error for nil wiring")
	}
	diag, ok := err.(*Diagnostic)
	if !ok {
		t.Fatalf("expected Diagnostic, got %T", err)
	}
	if diag.Code != "usables-nil-wiring" {
		t.Fatalf("code = %q, want usables-nil-wiring", diag.Code)
	}
}

func TestUsables_testingTExcludedFromUsables(t *testing.T) {
	src := `package main

import "testing"

func TestX(t *testing.T) {
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	if len(tc.FunctionUsables["TestX"]) != 0 {
		t.Fatalf("*testing.T param should not appear in Usables, got %v", tc.FunctionUsables["TestX"])
	}
}

func TestUsables_sidecarExport_errorsOnPublicWithUsables(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }

func PublicApi() {
	use logger: Logger
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected sidecar export error")
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "usables-sidecar-export" {
		t.Fatalf("got %v", err)
	}
}
