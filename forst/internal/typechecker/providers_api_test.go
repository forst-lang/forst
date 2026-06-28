package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func withChainFromMain(t *testing.T, src string) (*TypeChecker, []ast.WithNode) {
	t.Helper()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = moduleRootForProvidersTest(t)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	var chain []ast.WithNode
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok {
			continue
		}
		for _, st := range fn.Body {
			if w, ok := st.(ast.WithNode); ok {
				chain = append(chain, w)
				for _, inner := range w.Body {
					if w2, ok := inner.(ast.WithNode); ok {
						chain = append(chain, w2)
					}
				}
			}
		}
	}
	return tc, chain
}

func TestProvidersAPI_EffectiveScopeKeys_nestedWithShadow(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }
type NopLogger = { info(msg String) }

func main() {
	with {
		Logger: NopLogger{}
	} {
		with {
			Logger: NopLogger{}
		} {
		}
	}
}
`
	tc, chain := withChainFromMain(t, src)
	if len(chain) < 2 {
		t.Fatalf("expected nested with chain, got %d", len(chain))
	}
	labels, err := tc.EffectiveScopeKeyLabels(chain)
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) == 0 {
		t.Fatal("expected scope labels")
	}
	foundShadow := false
	for _, l := range labels {
		if l.Key == "Logger" && l.Shadowed {
			foundShadow = true
		}
	}
	if !foundShadow {
		t.Fatalf("expected Logger shadowed in inner scope, got %#v", labels)
	}
	keys, err := tc.EffectiveScopeKeys(chain)
	if err != nil || len(keys) == 0 {
		t.Fatalf("keys = %v err = %v", keys, err)
	}
}

func TestProvidersAPI_KnownProviderRootKeys_andImplTypes(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }
type NopLogger = { info(msg String) }

func main() {
	with { Logger: NopLogger{} } {
	}
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatal(err)
	}
	roots := tc.KnownProviderRootKeys()
	if len(roots) == 0 {
		t.Fatal("expected known roots")
	}
	impls := tc.ProviderImplTypeNamesForContract("Logger")
	if len(impls) == 0 {
		t.Fatal("expected impl types for Logger")
	}
	found := false
	for _, name := range impls {
		if name == "NopLogger" {
			found = true
		}
	}
	if !found {
		t.Fatalf("impls = %v", impls)
	}
	if got := tc.ProviderImplTypeNamesForContract("Missing"); got != nil {
		t.Fatalf("unknown contract should return nil, got %v", got)
	}
}

func TestProvidersAPI_CallSiteObligationChain_dedupesRoots(t *testing.T) {
	chain := CallSiteObligationChain("main", "Handle", []ProviderSlot{
		{RootIdent: "Logger"},
		{RootIdent: "Logger"},
		{RootIdent: "Clock"},
	})
	if !strings.Contains(chain, "main → Handle → Logger → Clock") {
		t.Fatalf("got %q", chain)
	}
}

func TestProvidersObligationChain_BuildProvidersObligationChain_transitive(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }

func useLogger(l Logger) {
	l.info("x")
}

func Handle() {
	useLogger(Logger{})
}

func main() {
	Handle()
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatal(err)
	}
	chain := tc.BuildProvidersObligationChain("Handle", "Logger")
	if !strings.Contains(chain, "Handle") || !strings.Contains(chain, "Logger") {
		t.Fatalf("got %q", chain)
	}
}

func TestProvidersFinish_wiringRootUnsatisfiedDiagnostic(t *testing.T) {
	// Reuses N4 wiring-root pattern from providers_test.go.
	src := `package main

import "testing"

type Logger = { info(msg String) }
type Clock = { now(): Int }

func needsBoth() {
	use logger: Logger
	use clock: Clock
}

func TestRoot(t *testing.T) {
	with { Logger: &NopLogger {} } {
		needsBoth()
	}
}

type NopLogger = {}
func (NopLogger) info(msg String) {}
`
	_, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected providers error at wiring root")
	}
	if !strings.Contains(err.Error(), "required by:") || !strings.Contains(err.Error(), "Clock") {
		t.Fatalf("expected obligation chain mentioning Clock: %v", err)
	}
}
