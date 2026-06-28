package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

type unsupportedInferNode struct{}

func (unsupportedInferNode) Kind() ast.NodeKind { return ast.NodeKind("UnsupportedInferNode") }
func (unsupportedInferNode) String() string     { return "UnsupportedInferNode" }

func TestInferNodeType_breakAndContinueLoopGuards(t *testing.T) {
	tc := New(logrus.New(), false)

	_, err := tc.inferNodeType(&ast.BreakNode{})
	if err == nil || !strings.Contains(err.Error(), "break is not inside a loop") {
		t.Fatalf("expected break outside loop error, got %v", err)
	}

	_, err = tc.inferNodeType(&ast.ContinueNode{})
	if err == nil || !strings.Contains(err.Error(), "continue is not inside a loop") {
		t.Fatalf("expected continue outside loop error, got %v", err)
	}

	tc.loopDepth = 1
	if _, err := tc.inferNodeType(&ast.BreakNode{}); err != nil {
		t.Fatalf("break inside loop should pass: %v", err)
	}
	if _, err := tc.inferNodeType(&ast.ContinueNode{}); err != nil {
		t.Fatalf("continue inside loop should pass: %v", err)
	}
}

func TestInferNodeType_breakAndContinueLabeled(t *testing.T) {
	tc := New(logrus.New(), false)

	outerFor := &ast.ForNode{
		Label: &ast.Ident{ID: "outer"},
		Body: []ast.Node{
			&ast.ForNode{
				Body: []ast.Node{
					&ast.BreakNode{Label: &ast.Ident{ID: "outer"}},
					&ast.ContinueNode{Label: &ast.Ident{ID: "outer"}},
				},
			},
		},
	}
	if _, err := tc.inferNodeType(outerFor); err != nil {
		t.Fatalf("labeled break/continue in nested loop: %v", err)
	}

	_, err := tc.inferNodeType(&ast.BreakNode{Label: &ast.Ident{ID: "missing"}})
	if err == nil || !strings.Contains(err.Error(), `undefined label "missing"`) {
		t.Fatalf("expected undefined label error, got %v", err)
	}
}

func TestInferNodeType_deferAndGoRequireFunctionCall(t *testing.T) {
	tc := New(logrus.New(), false)

	_, err := tc.inferNodeType(&ast.DeferNode{Call: ast.IntLiteralNode{Value: 1}})
	if err == nil || !strings.Contains(err.Error(), "defer requires a function or method call") {
		t.Fatalf("expected defer non-call error, got %v", err)
	}

	_, err = tc.inferNodeType(&ast.GoStmtNode{Call: ast.IntLiteralNode{Value: 1}})
	if err == nil || !strings.Contains(err.Error(), "go requires a function or method call") {
		t.Fatalf("expected go non-call error, got %v", err)
	}
}

func TestInferNodeType_elseBlockPointerAndValue(t *testing.T) {
	tc := New(logrus.New(), false)

	elsePointer := &ast.ElseBlockNode{Body: []ast.Node{ast.CommentNode{Text: "pointer-path"}}}
	_, err := tc.inferNodeType(elsePointer)
	if err != nil {
		t.Fatalf("non-nil else block pointer should be accepted: %v", err)
	}

	_, err = tc.inferNodeType(ast.ElseBlockNode{
		Body: []ast.Node{ast.CommentNode{Text: "ok"}},
	})
	if err != nil {
		t.Fatalf("else block with comment body should pass: %v", err)
	}
}

func TestInferNodeType_unsupportedNodeReturnsError(t *testing.T) {
	tc := New(logrus.New(), false)

	_, err := tc.inferNodeType(unsupportedInferNode{})
	if err == nil {
		t.Fatal("expected unsupported node error")
	}
	if !strings.Contains(err.Error(), "unsupported node type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
