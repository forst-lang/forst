package compileplan

import (
	"strings"
	"testing"

	"forst/internal/testutil"
	"forst/internal/typechecker"
)

func TestEmitGo_nilChecker_returnsEmpty(t *testing.T) {
	t.Parallel()
	code, err := EmitGo(nil, Plan{}, EmitExecutor)
	if err != nil {
		t.Fatalf("EmitGo: %v", err)
	}
	if code != "" {
		t.Fatalf("code = %q, want empty", code)
	}
}

func TestEmitGo_EmitLibShim_emitsPackageTypeDefs(t *testing.T) {
	t.Parallel()
	src := `package main

type Widget = { n: Int }

func main() {}
`
	tc, nodes := typechecker.MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	code, err := EmitGo(nil, Plan{Checker: tc, Nodes: nodes}, EmitLibShim)
	if err != nil {
		t.Fatalf("EmitGo: %v", err)
	}
	if !strings.Contains(code, "type Widget") {
		t.Fatalf("EmitLibShim should emit package type defs:\n%s", code)
	}
}

func TestEmitGo_EmitTest_omitsPackageTypeDefs(t *testing.T) {
	t.Parallel()
	src := `package main

type Widget = { n: Int }

func main() {}
`
	tc, nodes := typechecker.MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	code, err := EmitGo(nil, Plan{Checker: tc, Nodes: nodes}, EmitTest)
	if err != nil {
		t.Fatalf("EmitGo: %v", err)
	}
	if strings.Contains(code, "type Widget") {
		t.Fatalf("EmitTest should omit package type defs:\n%s", code)
	}
}

func TestEmitGo_EmbedInvoke_onlyWithCompanions(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {}
`
	tc, nodes := typechecker.MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	plan := Plan{Checker: tc, Nodes: nodes, EmbedInvoke: true}

	executorCode, err := EmitGo(nil, plan, EmitExecutor)
	if err != nil {
		t.Fatalf("EmitGo executor: %v", err)
	}
	if strings.Contains(executorCode, "ForstInvokeWaitForShutdown") {
		t.Fatalf("EmitExecutor should not embed invoke shutdown:\n%s", executorCode)
	}

	companionCode, err := EmitGo(nil, plan, EmitWithCompanions)
	if err != nil {
		t.Fatalf("EmitGo companions: %v", err)
	}
	if !strings.Contains(companionCode, "ForstInvokeWaitForShutdown") {
		t.Fatalf("EmitWithCompanions+EmbedInvoke should append shutdown hook:\n%s", companionCode)
	}
}
