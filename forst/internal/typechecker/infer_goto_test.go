package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func TestCheckFunctionLabels_forwardGoto(t *testing.T) {
	body := []ast.Node{
		&ast.GotoNode{Label: &ast.Ident{ID: "done"}},
		&ast.LabeledStmtNode{
			Label: &ast.Ident{ID: "done"},
			Stmt:  &ast.ReturnNode{},
		},
	}
	tc := New(logrus.New(), false)
	if err := tc.checkFunctionLabels(body); err != nil {
		t.Fatalf("forward goto: %v", err)
	}
}

func TestCheckFunctionLabels_undefinedGoto(t *testing.T) {
	tc := New(logrus.New(), false)
	err := tc.checkFunctionLabels([]ast.Node{
		&ast.GotoNode{Label: &ast.Ident{ID: "missing"}},
	})
	if err == nil || !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("expected undefined label error, got %v", err)
	}
}

func TestCheckFunctionLabels_unusedLabel(t *testing.T) {
	tc := New(logrus.New(), false)
	err := tc.checkFunctionLabels([]ast.Node{
		&ast.LabeledStmtNode{
			Label: &ast.Ident{ID: "unused"},
			Stmt:  &ast.ReturnNode{},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "not used") {
		t.Fatalf("expected unused label error, got %v", err)
	}
}

func TestCheckFunctionLabels_jumpIntoBlock(t *testing.T) {
	tc := New(logrus.New(), false)
	err := tc.checkFunctionLabels([]ast.Node{
		&ast.GotoNode{Label: &ast.Ident{ID: "inner"}},
		&ast.IfNode{
			Condition: ast.BoolLiteralNode{Value: true},
			Body: []ast.Node{
				&ast.LabeledStmtNode{
					Label: &ast.Ident{ID: "inner"},
					Stmt:  &ast.ReturnNode{},
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "jumps into block") {
		t.Fatalf("expected jump into block error, got %v", err)
	}
}

func TestCheckFunctionLabels_jumpOverDecl(t *testing.T) {
	tc := New(logrus.New(), false)
	err := tc.checkFunctionLabels([]ast.Node{
		&ast.GotoNode{Label: &ast.Ident{ID: "L"}},
		ast.AssignmentNode{
			LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "v"}}},
			RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 3}},
			IsShort: true,
		},
		&ast.LabeledStmtNode{
			Label: &ast.Ident{ID: "L"},
			Stmt:  &ast.ReturnNode{},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "jumps over declaration") {
		t.Fatalf("expected jump over declaration error, got %v", err)
	}
}

func TestCheckTypes_gotoCleanup(t *testing.T) {
	src := `package main

func main() {
	goto done
	println(1)
done:
	println(2)
}
`
	nodes := testutil.ParseSource(t, src, "goto_cleanup.ft", nil)
	tc := New(logrus.New(), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestCheckTypes_gotoBackwardLabeledIf(t *testing.T) {
	src := `package main

func main() {
	i := 0
retry:
	if i < 2 {
		i = i + 1
		goto retry
	}
	println(i)
}
`
	nodes := testutil.ParseSource(t, src, "goto_retry.ft", nil)
	tc := New(logrus.New(), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes labeled if: %v", err)
	}
}

func TestCheckTypes_gotoOverDeclRejected(t *testing.T) {
	src := `package main

func main() {
	goto L
	v := 3
L:
	println(v)
}
`
	nodes := testutil.ParseSource(t, src, "goto_over_decl.ft", nil)
	tc := New(logrus.New(), false)
	err := tc.CheckTypes(nodes)
	if err == nil || !strings.Contains(err.Error(), "jumps over declaration") {
		t.Fatalf("expected jump over declaration, got %v", err)
	}
	d, ok := err.(*Diagnostic)
	if !ok || !d.Span.IsSet() {
		t.Fatalf("expected Diagnostic with span, got %T %v", err, err)
	}
	if d.Span.StartLine < 2 {
		t.Fatalf("diagnostic should not be at line 1 only, got line %d", d.Span.StartLine)
	}
}

func TestCheckTypes_gotoPersistsLabelBindings(t *testing.T) {
	src := `package main

func main() {
	goto done
	println(1)
done:
	println(2)
}
`
	nodes := testutil.ParseSource(t, src, "goto_persist.ft", nil)
	tc := New(logrus.New(), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	if len(tc.LabelScopes) == 0 || len(tc.LabelScopes[0].Labels) == 0 {
		t.Fatal("expected persisted label bindings")
	}
	b := tc.LabelScopes[0].Labels[0]
	if b.Name != "done" {
		t.Fatalf("label name = %q", b.Name)
	}
	if !b.DeclSpan.IsSet() || len(b.UseSpans) == 0 {
		t.Fatalf("expected decl+use spans, decl=%v uses=%v", b.DeclSpan, b.UseSpans)
	}
	found := tc.LabelBindingAtSpan(b.UseSpans[0])
	if found == nil || found.Name != "done" {
		t.Fatalf("LabelBindingAtSpan use: %+v", found)
	}
}

func TestCheckTypes_closureCannotSeeOuterLoopLabel(t *testing.T) {
	src := `package main

func main() {
outer:
	for {
		_ = func() {
			break outer
		}
	}
}
`
	nodes := testutil.ParseSource(t, src, "closure_label.ft", nil)
	tc := New(logrus.New(), false)
	err := tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected error: closure must not see outer loop label")
	}
	if !strings.Contains(err.Error(), "label") && !strings.Contains(err.Error(), "break") {
		t.Fatalf("unexpected error: %v", err)
	}
}
