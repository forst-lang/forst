package parser //nolint:revive // package name matches internal/parser; overlaps with text/template parser etc.

import (
	"testing"

	"forst/internal/ast"
)

func TestParseExpression_spreadInCallArgument(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	f(argv[1:]...)
}
`
	logger := ast.SetupTestLogger(nil)
	p := NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findFunction(t, nodes, "main")
	call, ok := fn.Body[0].(ast.FunctionCallNode)
	if !ok {
		t.Fatalf("want FunctionCallNode, got %T", fn.Body[0])
	}
	if len(call.Arguments) != 1 {
		t.Fatalf("args len = %d, want 1", len(call.Arguments))
	}
	spread, ok := call.Arguments[0].(ast.SpreadExpressionNode)
	if !ok {
		t.Fatalf("arg: want SpreadExpressionNode, got %T", call.Arguments[0])
	}
	slice, ok := spread.Expr.(ast.SliceExpressionNode)
	if !ok {
		t.Fatalf("spread expr: want SliceExpressionNode, got %T", spread.Expr)
	}
	if _, ok := slice.Target.(ast.VariableNode); !ok {
		t.Fatalf("slice target: want VariableNode, got %T", slice.Target)
	}
}

func TestParseExpression_fieldAccessOnCallResult(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	_ = getCmd().ProcessState
}
`
	logger := ast.SetupTestLogger(nil)
	p := NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findFunction(t, nodes, "main")
	assign := fn.Body[0].(ast.AssignmentNode)
	fa, ok := assign.RValues[0].(ast.FieldAccessNode)
	if !ok {
		t.Fatalf("rhs: want FieldAccessNode, got %T", assign.RValues[0])
	}
	if fa.Field.ID != "ProcessState" {
		t.Fatalf("field id = %q", fa.Field.ID)
	}
	if _, ok := fa.Target.(ast.FunctionCallNode); !ok {
		t.Fatalf("target: want FunctionCallNode, got %T", fa.Target)
	}
}

func TestParseExpression_qualifiedFieldPathOnLocal(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	_ = cmd.ProcessState
}
`
	logger := ast.SetupTestLogger(nil)
	p := NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findFunction(t, nodes, "main")
	assign := fn.Body[0].(ast.AssignmentNode)
	vn, ok := assign.RValues[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("rhs: want VariableNode cmd.ProcessState, got %T", assign.RValues[0])
	}
	if vn.Ident.ID != "cmd.ProcessState" {
		t.Fatalf("ident id = %q", vn.Ident.ID)
	}
}

func TestParseExpression_chainedFieldAccess(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	_ = cmd.ProcessState.ExitCode()
}
`
	logger := ast.SetupTestLogger(nil)
	p := NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findFunction(t, nodes, "main")
	assign := fn.Body[0].(ast.AssignmentNode)
	call, ok := assign.RValues[0].(ast.FunctionCallNode)
	if !ok {
		t.Fatalf("rhs: want FunctionCallNode, got %T", assign.RValues[0])
	}
	if call.Function.ID != "cmd.ProcessState.ExitCode" {
		t.Fatalf("function id = %q", call.Function.ID)
	}
}

func TestParseExpression_subslicePreservesTarget(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	_ = argv[1:]
}
`
	logger := ast.SetupTestLogger(nil)
	p := NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findFunction(t, nodes, "main")
	assign := fn.Body[0].(ast.AssignmentNode)
	slice, ok := assign.RValues[0].(ast.SliceExpressionNode)
	if !ok {
		t.Fatalf("rhs: want SliceExpressionNode, got %T", assign.RValues[0])
	}
	vn, ok := slice.Target.(ast.VariableNode)
	if !ok || vn.GetIdent() != "argv" {
		t.Fatalf("target: want VariableNode argv, got %#v", slice.Target)
	}
}

func TestParseExpression_spreadAfterSubslice(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	exec.Command(argv[0], argv[1:]...)
}
`
	logger := ast.SetupTestLogger(nil)
	p := NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findFunction(t, nodes, "main")
	call, ok := fn.Body[0].(ast.FunctionCallNode)
	if !ok {
		t.Fatalf("want FunctionCallNode, got %T", fn.Body[0])
	}
	if len(call.Arguments) != 2 {
		t.Fatalf("args len = %d, want 2", len(call.Arguments))
	}
	spread, ok := call.Arguments[1].(ast.SpreadExpressionNode)
	if !ok {
		t.Fatalf("arg[1]: want SpreadExpressionNode, got %T", call.Arguments[1])
	}
	if _, ok := spread.Expr.(ast.SliceExpressionNode); !ok {
		t.Fatalf("spread expr: want SliceExpressionNode, got %T", spread.Expr)
	}
}

func TestParseExpression_rejectsSpreadOutsideCall(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	_ = argv...
}
`
	logger := ast.SetupTestLogger(nil)
	p := NewTestParser(src, logger)
	defer func() {
		if recover() == nil {
			t.Fatal("expected parse panic for spread outside call")
		}
	}()
	_, _ = p.ParseFile()
}
