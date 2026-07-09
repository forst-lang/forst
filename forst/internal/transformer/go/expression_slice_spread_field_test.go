package transformergo

import (
	"bytes"
	goast "go/ast"
	"go/format"
	"go/token"
	"strings"
	"testing"

	forstast "forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testutil"
	"forst/internal/typechecker"
)

func formatGoExpr(t *testing.T, expr goast.Expr) string {
	t.Helper()
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), expr); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestTransformExpression_sliceSubslice_allForms(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	main := forstast.FunctionNode{
		Ident: forstast.Ident{ID: "main"},
		Body: []forstast.Node{
			forstast.AssignmentNode{
				LValues: []forstast.ExpressionNode{forstast.VariableNode{Ident: forstast.Ident{ID: "xs"}}},
				RValues: []forstast.ExpressionNode{
					forstast.ArrayLiteralNode{
						Value: []forstast.ExpressionNode{
							forstast.IntLiteralNode{Value: 1},
							forstast.IntLiteralNode{Value: 2},
							forstast.IntLiteralNode{Value: 3},
						},
						Type: forstast.TypeNode{Ident: forstast.TypeImplicit},
					},
				},
				IsShort: true,
			},
		},
	}
	if err := tc.CheckTypes([]forstast.Node{main}); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}

	xs := forstast.VariableNode{Ident: forstast.Ident{ID: "xs"}}
	cases := []struct {
		name string
		expr forstast.SliceExpressionNode
		want string
	}{
		{
			name: "low high",
			expr: forstast.SliceExpressionNode{
				Target: xs,
				Low:    forstast.IntLiteralNode{Value: 1},
				High:   forstast.IntLiteralNode{Value: 3},
			},
			want: "xs[1:3]",
		},
		{
			name: "low only",
			expr: forstast.SliceExpressionNode{
				Target: xs,
				Low:    forstast.IntLiteralNode{Value: 1},
			},
			want: "xs[1:]",
		},
		{
			name: "high only",
			expr: forstast.SliceExpressionNode{
				Target: xs,
				High:   forstast.IntLiteralNode{Value: 2},
			},
			want: "xs[:2]",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tr.transformExpression(tc.expr)
			if err != nil {
				t.Fatal(err)
			}
			slice, ok := out.(*goast.SliceExpr)
			if !ok {
				t.Fatalf("want *ast.SliceExpr, got %T", out)
			}
			got := formatGoExpr(t, slice)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTransformExpression_spreadArgument(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	args := []forstast.ExpressionNode{
		forstast.SpreadExpressionNode{
			Expr: forstast.VariableNode{Ident: forstast.Ident{ID: "argv"}},
		},
	}
	out, err := tr.transformFunctionCallArgs(forstast.Identifier("f"), args)
	if err != nil {
		t.Fatal(err)
	}
	if out.ellipsis == 0 {
		t.Fatal("expected ellipsis on spread call args")
	}
	if len(out.exprs) != 1 {
		t.Fatalf("exprs len = %d, want 1", len(out.exprs))
	}
	if formatGoExpr(t, out.exprs[0]) != "argv" {
		t.Fatalf("spread inner = %q", formatGoExpr(t, out.exprs[0]))
	}
}

func TestTransformExpression_goFieldAccess(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	expr := forstast.FieldAccessNode{
		Target: forstast.VariableNode{Ident: forstast.Ident{ID: "cmd"}},
		Field:  forstast.Ident{ID: "ProcessState"},
	}
	out, err := tr.transformExpression(expr)
	if err != nil {
		t.Fatal(err)
	}
	sel, ok := out.(*goast.SelectorExpr)
	if !ok {
		t.Fatalf("want *ast.SelectorExpr, got %T", out)
	}
	if sel.Sel.Name != "ProcessState" {
		t.Fatalf("field = %q", sel.Sel.Name)
	}
	if ident, ok := sel.X.(*goast.Ident); !ok || ident.Name != "cmd" {
		t.Fatalf("receiver = %#v", sel.X)
	}
}

func TestTransformExpression_goBoundDottedMethodCall_ProcessStateExitCode(t *testing.T) {
	t.Parallel()
	src := `package main
import "os/exec"
func f(cmd *exec.Cmd): Int {
	return cmd.ProcessState.ExitCode()
}
`
	code := MustCompileGo(t, src, testutil.CompileOpts{TypecheckOpts: testutil.TypecheckOpts{UseModuleRoot: true, SkipUnlessGoImport: "exec"}})
	if !strings.Contains(code, "ProcessState") || !strings.Contains(code, "ExitCode") {
		t.Fatalf("expected ProcessState.ExitCode lowering, got:\n%s", code)
	}

	log := setupTestLogger(nil)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := typechecker.New(log, false)
	tc.GoWorkspaceDir = testutil.ModuleRoot(t)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	if tc.GoTypeForVariable(forstast.Identifier("cmd")) == nil {
		t.Skip("cmd Go type not available")
	}
	tr := setupTransformer(tc, log)
	call := forstast.FunctionCallNode{
		Function: forstast.Ident{ID: "cmd.ProcessState.ExitCode"},
	}
	out, ok, err := tr.transformGoBoundDottedMethodCall(call)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected transformGoBoundDottedMethodCall to handle dotted method call")
	}
	callExpr, ok := out.(*goast.CallExpr)
	if !ok {
		t.Fatalf("want *ast.CallExpr, got %T", out)
	}
	sel, ok := callExpr.Fun.(*goast.SelectorExpr)
	if !ok || sel.Sel.Name != "ExitCode" {
		t.Fatalf("fun = %#v", callExpr.Fun)
	}
}

func TestMustCompileGo_execCommandSliceSpread(t *testing.T) {
	t.Parallel()
	src := `package main
import "os/exec"
func main() {
	argv := []String{"true", "extra"}
	exec.Command(argv[0], argv[1:]...)
}
`
	code := MustCompileGo(t, src, testutil.CompileOpts{TypecheckOpts: testutil.TypecheckOpts{UseModuleRoot: true, SkipUnlessGoImport: "exec"}})
	if !strings.Contains(code, "exec.Command") {
		t.Fatalf("missing exec.Command in output:\n%s", code)
	}
	if !strings.Contains(code, "argv[1:]") && !strings.Contains(code, "argv[1 : ]") {
		t.Fatalf("expected subslice in spread arg, got:\n%s", code)
	}
}
