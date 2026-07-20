package typechecker

import (
	"strings"
	"testing"

	"forst/internal/nodeinterop"
	"forst/internal/testutil"
)

func TestNodeExportParamTypes_nilResolverErrors(t *testing.T) {
	tc := New(nil, false)
	_, err := tc.NodeExportParamTypes("legacy/payment.ts", "create")
	if err == nil || !strings.Contains(err.Error(), "node index resolver not initialized") {
		t.Fatalf("got %v", err)
	}
}

func TestNodeCallTarget_nilTypeChecker(t *testing.T) {
	var tc *TypeChecker
	_, ok := tc.NodeCallTarget("payment", "create")
	if ok {
		t.Fatal("expected false for nil typechecker")
	}
}

func TestNodeCallTarget_resolvesSyncExport(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)

	src := `package main
import node "./legacy/payment"

func main() {
	payment.create(10.0, "usd")
}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	target, ok := tc.NodeCallTarget("payment", "create")
	if !ok {
		t.Fatal("expected NodeCallTarget to resolve")
	}
	if target.ModuleID != "legacy/payment.ts" || target.ExportName != "create" {
		t.Fatalf("target = %+v", target)
	}
	if target.Kind != nodeinterop.ExportKindAsyncFunction {
		t.Fatalf("kind = %q", target.Kind)
	}
}

func TestNodeExportParamTypes_returnsIndexSignature(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)

	src := `package main
import node "./legacy/payment"

func main() {
	payment.create(10.0, "usd")
}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	params, err := tc.NodeExportParamTypes("legacy/payment.ts", "create")
	if err != nil {
		t.Fatal(err)
	}
	if len(params) != 2 {
		t.Fatalf("params = %v", params)
	}
}

func TestNodeExportParamTypes_missingExportErrors(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)

	src := `package main
import node "./legacy/payment"

func main() {}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	_, err := tc.NodeExportParamTypes("legacy/payment.ts", "missing")
	if err == nil {
		t.Fatal("expected error for missing export")
	}
}

func TestNodeModuleForLocal_resolvesAlias(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)

	src := `package main
import node "./legacy/payment"

func main() {}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	moduleID, ok := tc.NodeModuleForLocal("payment")
	if !ok || moduleID != "legacy/payment.ts" {
		t.Fatalf("moduleID = %q ok = %v", moduleID, ok)
	}
}
