package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestCollectResultErrSlotUsed_ensureAndIfDiscriminators(t *testing.T) {
	t.Parallel()
	varName := "x"
	bodyWithEnsure := []ast.Node{
		ast.EnsureNode{Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}}},
	}
	if collectResultErrSlotUsed(bodyWithEnsure, varName) != true {
		t.Fatal("expected ensure on x to use error slot")
	}
	bodyWithIfOk := []ast.Node{
		ast.IfNode{
			Condition: ast.BinaryExpressionNode{
				Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
				Operator: ast.TokenIs,
				Right: ast.AssertionNode{
					Constraints: []ast.ConstraintNode{{Name: "Ok"}},
				},
			},
		},
	}
	if collectResultErrSlotUsed(bodyWithIfOk, varName) != true {
		t.Fatal("expected if x is Ok() to use error slot")
	}
	bodyUnused := []ast.Node{
		ast.FunctionCallNode{
			Function:  ast.Ident{ID: "println"},
			Arguments: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		},
	}
	if collectResultErrSlotUsed(bodyUnused, varName) {
		t.Fatal("expected no error slot use when x is never referenced")
	}
}

func TestTransformResultSplitAssignment_unusedErrAndSuccess_goBuilds(t *testing.T) {
	t.Parallel()
	src := `package main

func f(): Result(Int, Error) {
	return 1
}

func main() {
	x := f()
	println(1)
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, "_, _ = f()") && !strings.Contains(out, "_, _ := f()") {
		t.Fatalf("expected blank bindings when Result local is unused, got:\n%s", out)
	}
	assertGoBuildsInTempModule(t, out)
}

func TestTransformResultSplitAssignment_ifIsOk_keepsErrSlot(t *testing.T) {
	t.Parallel()
	src := `package main

func f(): Result(Int, Error) {
	return 1
}

func main() {
	x := f()
	if x is Ok() {
		println(x)
	}
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, "xErr") {
		t.Fatalf("expected xErr when if x is Ok() checks error slot, got:\n%s", out)
	}
	assertGoBuildsInTempModule(t, out)
}
