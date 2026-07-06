package executor

import (
	"path/filepath"
	"strings"
	"testing"
)

const ifAfterAssignExecutorSrc = `package demo

func F(cells []String, a Int): String {
	x0 := cells[a]
	if x0 == "" {
		return ""
	}
	return x0
}
`

func TestCompileFunction_rebindsScopesAfterModuleCheck(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo.ft"), ifAfterAssignExecutorSrc)

	ex := testExecutor(t, root)
	compiled, err := ex.compileFunction("demo", "F")
	if err != nil {
		t.Fatalf("compileFunction(demo, F): %v", err)
	}
	if compiled == nil {
		t.Fatal("expected compiled function")
	}
	if compiled.FunctionName != "F" || compiled.PackageName != "demo" {
		t.Fatalf("unexpected metadata: %+v", compiled)
	}
	if !strings.Contains(compiled.GoCode, "func F(") {
		t.Fatalf("expected F in Go output, got:\n%s", compiled.GoCode)
	}
}
