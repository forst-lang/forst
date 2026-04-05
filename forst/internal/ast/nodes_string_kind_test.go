package ast

import (
	"strings"
	"testing"
)

func TestNodeKindAndString_representativeNodes(t *testing.T) {
	base := TypeIdent("User")
	tests := []struct {
		name     string
		node     Node
		wantKind NodeKind
		contains string // substring expected in String()
	}{
		{"IntLiteral", IntLiteralNode{Value: 7}, NodeKindIntLiteral, "7"},
		{"FloatLiteral", FloatLiteralNode{Value: 1.5}, NodeKindFloatLiteral, "1.5"},
		{"StringLiteral", StringLiteralNode{Value: "a"}, NodeKindStringLiteral, `"a"`},
		{"BoolLiteral", BoolLiteralNode{Value: true}, NodeKindBoolLiteral, "true"},
		{"NilLiteral", NilLiteralNode{}, NodeKindNilLiteral, "nil"},
		{"Variable", VariableNode{Ident: Ident{ID: "v"}}, NodeKindVariable, "v"},
		{"UnaryExpression", UnaryExpressionNode{Operator: TokenMinus, Operand: IntLiteralNode{Value: 1}}, NodeKindUnaryExpression, "MINUS"},
		{"BinaryExpression", BinaryExpressionNode{Left: IntLiteralNode{Value: 1}, Operator: TokenPlus, Right: IntLiteralNode{Value: 2}}, NodeKindBinaryExpression, "PLUS"},
		{"FunctionCall", FunctionCallNode{Function: Ident{ID: "f"}, Arguments: []ExpressionNode{IntLiteralNode{Value: 0}}}, NodeKindFunctionCall, "f"},
		{"Package", PackageNode{Ident: Ident{ID: "main"}}, NodeKindPackage, "main"},
		{"Import", ImportNode{Path: "fmt"}, NodeKindImport, "fmt"},
		{"Return", ReturnNode{Values: []ExpressionNode{IntLiteralNode{Value: 3}}}, NodeKindReturn, "Return"},
		{"Assignment", AssignmentNode{LValues: []VariableNode{{Ident: Ident{ID: "x"}}}, RValues: []ExpressionNode{IntLiteralNode{Value: 1}}}, NodeKindAssignment, "x"},
		{"TypeNode", TypeNode{Ident: TypeString}, NodeKindType, "String"},
		{"Assertion", AssertionNode{BaseType: &base, Constraints: []ConstraintNode{{Name: "Min", Args: []ConstraintArgumentNode{}}}}, NodeKindAssertion, "User"},
		{"Shape", ShapeNode{Fields: map[string]ShapeFieldNode{"a": {Type: &TypeNode{Ident: TypeInt}}}}, NodeKindShape, "a"},
		{"If", IfNode{Condition: BoolLiteralNode{Value: true}, Body: []Node{}}, NodeKindIf, "If"},
		{"If_with_init", IfNode{
			Init:      AssignmentNode{LValues: []VariableNode{{Ident: Ident{ID: "i"}}}, RValues: []ExpressionNode{IntLiteralNode{Value: 0}}},
			Condition: BoolLiteralNode{Value: true},
			Body:      []Node{},
		}, NodeKindIf, "If("},
		{"ElseIf", ElseIfNode{Condition: BoolLiteralNode{Value: false}, Body: []Node{}}, NodeKindElseIf, "ElseIf"},
		{"ElseBlock", ElseBlockNode{Body: []Node{IntLiteralNode{Value: 1}}}, NodeKindElseBlock, "Else"},
		{"EnsureBlock", EnsureBlockNode{Body: []Node{}}, NodeKindEnsureBlock, "EnsureBlock"},
		{"TypeGuard", TypeGuardNode{Ident: "IsOK", Subject: SimpleParamNode{Ident: Ident{ID: "x"}, Type: TypeNode{Ident: TypeInt}}, Body: []Node{}}, NodeKindTypeGuard, "IsOK"},
		{"Ensure", EnsureNode{Variable: VariableNode{Ident: Ident{ID: "x"}}, Assertion: AssertionNode{}}, NodeKindEnsure, "Ensure"},
		{"Reference", ReferenceNode{Value: VariableNode{Ident: Ident{ID: "p"}}}, NodeKindReference, "Ref"},
		{"Dereference", DereferenceNode{Value: VariableNode{Ident: Ident{ID: "p"}}}, NodeKindDereference, "Deref"},
		{"SimpleParam", SimpleParamNode{Ident: Ident{ID: "a"}, Type: TypeNode{Ident: TypeInt}}, NodeKindSimpleParam, "a"},
		{"TypeDef", TypeDefNode{Ident: "T", Expr: TypeDefShapeExpr{Shape: ShapeNode{Fields: map[string]ShapeFieldNode{}}}}, NodeKindTypeDef, "T"},
		{"For_classic", ForNode{Cond: BoolLiteralNode{Value: true}, Body: []Node{}}, NodeKindFor, "For"},
		{"For_range", ForNode{IsRange: true, RangeX: VariableNode{Ident: Ident{ID: "xs"}}, Body: []Node{}}, NodeKindFor, "range"},
		{"Break", BreakNode{}, NodeKindBreak, "Break"},
		{"Continue", ContinueNode{}, NodeKindContinue, "Continue"},
		{"Comment", CommentNode{Text: "// x"}, NodeKindComment, "Comment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.node.Kind() != tt.wantKind {
				t.Fatalf("Kind: got %q want %q", tt.node.Kind(), tt.wantKind)
			}
			s := tt.node.String()
			if tt.contains != "" && !strings.Contains(s, tt.contains) {
				t.Fatalf("String() = %q, want substring %q", s, tt.contains)
			}
		})
	}
}

func TestConstraintArgumentNode_String_branches(t *testing.T) {
	var v ValueNode = IntLiteralNode{Value: 1}
	arg := ConstraintArgumentNode{Value: &v}
	if !strings.Contains(arg.String(), "1") {
		t.Fatalf("value branch: %s", arg.String())
	}
	arg2 := ConstraintArgumentNode{Shape: &ShapeNode{Fields: map[string]ShapeFieldNode{"k": {Type: &TypeNode{Ident: TypeInt}}}}}
	if !strings.Contains(arg2.String(), "k") {
		t.Fatalf("shape branch: %s", arg2.String())
	}
	arg3 := ConstraintArgumentNode{Type: &TypeNode{Ident: TypeString}}
	if !strings.Contains(arg3.String(), "String") {
		t.Fatalf("type branch: %s", arg3.String())
	}
	empty := ConstraintArgumentNode{}
	if empty.String() != "?" {
		t.Fatalf("empty: %q", empty.String())
	}
}
