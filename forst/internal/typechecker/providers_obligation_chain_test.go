package typechecker

import (
	"strings"
	"testing"
)

func TestBuildProvidersObligationChain_fromParse(t *testing.T) {
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
	tc, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected unsatisfied providers error")
	}
	diag, ok := err.(*Diagnostic)
	if !ok {
		t.Fatalf("got %T: %v", err, err)
	}
	if !strings.Contains(diag.Msg, "TestX") || !strings.Contains(diag.Msg, "needsLogger") {
		t.Fatalf("expected obligation chain in message: %q", diag.Msg)
	}
	chain := tc.BuildProvidersObligationChain("TestX", "Logger")
	if !strings.Contains(chain, "TestX") || !strings.Contains(chain, "Logger") {
		t.Fatalf("BuildProvidersObligationChain = %q", chain)
	}
}

func TestBuildCallSiteObligationChain(t *testing.T) {
	tc := New(nil, false)
	tc.SetForstPackage("main")
	chain := tc.buildCallSiteObligationChain("caller", "callee", []string{"Logger"})
	if !strings.Contains(chain, "caller") || !strings.Contains(chain, "callee") || !strings.Contains(chain, "Logger") {
		t.Fatalf("chain = %q", chain)
	}
}

func TestBuildWiringRootObligationChain(t *testing.T) {
	tc := New(nil, false)
	tc.SetForstPackage("main")
	chain := tc.buildWiringRootObligationChain("TestX", []ProviderSlot{
		{RootIdent: "Logger", Key: "Logger"},
	})
	if !strings.Contains(chain, "TestX") || !strings.Contains(chain, "Logger") {
		t.Fatalf("chain = %q", chain)
	}
}
