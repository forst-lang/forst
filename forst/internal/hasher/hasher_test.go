package hasher

import (
	"bytes"
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestStructuralHasher_Consistency(t *testing.T) {
	hasher := New()

	// Test that same structures produce same hashes
	t.Run("same structures produce same hashes", func(t *testing.T) {
		// Create two identical variable nodes
		node1 := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
		node2 := ast.VariableNode{Ident: ast.Ident{ID: "x"}}

		hash1, err := hasher.HashNode(node1)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		hash2, err := hasher.HashNode(node2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if hash1 != hash2 {
			t.Errorf("Expected identical nodes to produce same hash, got %v and %v", hash1, hash2)
		}
	})

	t.Run("different structures produce different hashes", func(t *testing.T) {
		// Create two different variable nodes
		node1 := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
		node2 := ast.VariableNode{Ident: ast.Ident{ID: "y"}}

		hash1, err := hasher.HashNode(node1)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		hash2, err := hasher.HashNode(node2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if hash1 == hash2 {
			t.Errorf("Expected different nodes to produce different hashes")
		}
	})

	t.Run("EnsureNode subject variable affects hash", func(t *testing.T) {
		// Regression: EnsureNode must hash Variable; otherwise structurally identical
		// assertions (e.g. Min(1)) on different subjects collide in scopeStack.scopes.
		assertion := ast.AssertionNode{
			Constraints: []ast.ConstraintNode{
				{Name: "Min", Args: []ast.ConstraintArgumentNode{}},
			},
		}
		e1 := ast.EnsureNode{
			Variable:  ast.VariableNode{Ident: ast.Ident{ID: "a"}},
			Assertion: assertion,
		}
		e2 := ast.EnsureNode{
			Variable:  ast.VariableNode{Ident: ast.Ident{ID: "b"}},
			Assertion: assertion,
		}
		h1, err := hasher.HashNode(e1)
		if err != nil {
			t.Fatalf("hash e1: %v", err)
		}
		h2, err := hasher.HashNode(e2)
		if err != nil {
			t.Fatalf("hash e2: %v", err)
		}
		if h1 == h2 {
			t.Fatalf("EnsureNodes with different subjects must not share a hash")
		}
	})

	t.Run("map field order doesn't affect hash", func(t *testing.T) {
		// Create two maps with same fields in different order
		map1 := ast.MapLiteralNode{
			Entries: []ast.MapEntryNode{
				{Key: ast.StringLiteralNode{Value: "a"}, Value: ast.IntLiteralNode{Value: 1}},
				{Key: ast.StringLiteralNode{Value: "b"}, Value: ast.IntLiteralNode{Value: 2}},
			},
		}
		map2 := ast.MapLiteralNode{
			Entries: []ast.MapEntryNode{
				{Key: ast.StringLiteralNode{Value: "b"}, Value: ast.IntLiteralNode{Value: 2}},
				{Key: ast.StringLiteralNode{Value: "a"}, Value: ast.IntLiteralNode{Value: 1}},
			},
		}

		hash1, err := hasher.HashNode(map1)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		hash2, err := hasher.HashNode(map2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if hash1 != hash2 {
			t.Errorf("Expected maps with same fields in different order to produce same hash")
		}
	})

	t.Run("pointer vs value types produce same hash", func(t *testing.T) {
		// Create same structure as pointer and value
		valueNode := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
		ptrNode := &valueNode

		hash1, err := hasher.HashNode(valueNode)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		hash2, err := hasher.HashNode(ptrNode)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if hash1 != hash2 {
			t.Errorf("Expected pointer and value types to produce same hash")
		}
	})

	t.Run("nil values handled correctly", func(t *testing.T) {
		// Test nil pointer
		var nilNode *ast.VariableNode
		hash1, err := hasher.HashNode(nilNode)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Test node with nil fields
		node := ast.EnsureNode{
			Variable:  ast.VariableNode{Ident: ast.Ident{ID: "x"}},
			Assertion: ast.AssertionNode{},
			Error:     nil,
			Block:     nil,
		}
		hash2, err := hasher.HashNode(node)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Both should be valid hashes (not panic)
		if hash1 == 0 || hash2 == 0 {
			t.Errorf("Expected non-zero hashes for nil values")
		}
	})

	t.Run("nested structures hash correctly", func(t *testing.T) {
		// Create nested structure
		inner := ast.BinaryExpressionNode{
			Left:     ast.IntLiteralNode{Value: 1},
			Operator: ast.TokenPlus,
			Right:    ast.IntLiteralNode{Value: 2},
		}
		outer := ast.BinaryExpressionNode{
			Left:     inner,
			Operator: ast.TokenPlus,
			Right:    ast.IntLiteralNode{Value: 3},
		}

		hash, err := hasher.HashNode(outer)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if hash == 0 {
			t.Errorf("Expected non-zero hash for nested structure")
		}
	})

	t.Run("type identifiers hash correctly", func(t *testing.T) {
		// Test type identifiers
		type1 := ast.TypeNode{Ident: "int"}
		type2 := ast.TypeNode{Ident: "string"}

		hash1, err := hasher.HashNode(type1)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		hash2, err := hasher.HashNode(type2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if hash1 == hash2 {
			t.Errorf("Expected different type identifiers to produce different hashes")
		}
	})

	t.Run("function parameters hash correctly", func(t *testing.T) {
		// Test function parameters
		param1 := ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: "int"},
		}
		param2 := ast.SimpleParamNode{
			Ident: ast.Ident{ID: "y"},
			Type:  ast.TypeNode{Ident: "int"},
		}

		hash1, err := hasher.HashNode(param1)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		hash2, err := hasher.HashNode(param2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if hash1 == hash2 {
			t.Errorf("Expected different parameters to produce different hashes")
		}
	})
}

func TestNodeHash_ToTypeIdent(t *testing.T) {
	hasher := New()
	node := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
	hash, err := hasher.HashNode(node)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test that ToTypeIdent produces a valid type identifier
	typeIdent := hash.ToTypeIdent()
	if typeIdent == "" {
		t.Errorf("Expected non-empty type identifier")
	}
	if typeIdent[0:2] != "T_" {
		t.Errorf("Expected type identifier to start with 'T_', got %s", typeIdent)
	}
}

func TestHashTokenType_distinctTokens(t *testing.T) {
	h := New()
	a := h.HashTokenType(ast.TokenPlus)
	b := h.HashTokenType(ast.TokenMinus)
	if a == b {
		t.Fatal("expected different hashes for different token types")
	}
}

func TestHashNode_nil_maps_to_invalid_type_ident(t *testing.T) {
	h := New()
	nh, err := h.HashNode(nil)
	if err != nil {
		t.Fatal(err)
	}
	if nh.ToTypeIdent() != "T_Invalid" {
		t.Fatalf("got %s", nh.ToTypeIdent())
	}
	if nh.ToGuardIdent() != "G_Invalid" {
		t.Fatalf("got %s", nh.ToGuardIdent())
	}
}

func TestHashNode_unsupported_shape_guard_errors(t *testing.T) {
	h := New()
	sg := ast.ShapeGuardNode{
		TypeGuardNode: ast.TypeGuardNode{
			Ident: "SG",
			Subject: ast.SimpleParamNode{
				Ident: ast.Ident{ID: "x"},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
			Body: []ast.Node{},
		},
		TypeArg:   ast.TypeNode{Ident: ast.TypeString},
		FieldName: "f",
	}
	_, err := h.HashNode(sg)
	if err == nil || !strings.Contains(err.Error(), "unsupported node type") {
		t.Fatalf("expected unsupported error, got %v", err)
	}
}

func TestHashNode_additional_structural_variants(t *testing.T) {
	h := New()
	tests := []struct {
		name string
		node ast.Node
	}{
		{"UnaryExpression", ast.UnaryExpressionNode{
			Operator: ast.TokenMinus,
			Operand:  ast.IntLiteralNode{Value: 1},
		}},
		{"FloatLiteral", ast.FloatLiteralNode{Value: 3.14}},
		{"TypeDefBinaryExpr", ast.TypeDefBinaryExpr{
			Op:    ast.TokenBitwiseAnd,
			Left:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
			Right: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"a": {Type: &ast.TypeNode{Ident: ast.TypeInt}}}}},
		}},
		{"Package", ast.PackageNode{Ident: ast.Ident{ID: "pkg"}}},
		{"Import", ast.ImportNode{Path: "fmt"}},
		{"Return", ast.ReturnNode{Values: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}}},
		{"Dereference", ast.DereferenceNode{Value: ast.VariableNode{Ident: ast.Ident{ID: "p"}}}},
		{"ArrayLiteral", ast.ArrayLiteralNode{Type: ast.TypeNode{Ident: ast.TypeInt}, Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 1}}}},
		{"TypeDefShapeExpr", ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}},
		{"NilLiteral", ast.NilLiteralNode{}},
		{"TypeDefAssertionExpr", ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{}}},
		{"ConstraintNode", ast.ConstraintNode{Name: "Min", Args: []ast.ConstraintArgumentNode{}}},
		{"TypeGuardNode", ast.TypeGuardNode{
			Ident: "G",
			Subject: ast.SimpleParamNode{
				Ident: ast.Ident{ID: "x"},
				Type:  ast.TypeNode{Ident: ast.TypeInt},
			},
			Body: []ast.Node{},
		}},
		{"IfNode", ast.IfNode{
			Condition: ast.BoolLiteralNode{Value: true},
			Body:      []ast.Node{},
		}},
		{"EnsureBlockNode", &ast.EnsureBlockNode{Body: []ast.Node{ast.IntLiteralNode{Value: 1}}}},
		{"AssignmentNode", ast.AssignmentNode{
			LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x"}}},
			RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		}},
		{"DestructuredParamNode", ast.DestructuredParamNode{
			Fields: []string{"a", "b"},
			Type:   ast.TypeNode{Ident: ast.TypeInt},
		}},
		{"ImportNode_alias", func() ast.Node {
			a := ast.Ident{ID: "f"}
			return ast.ImportNode{Path: "fmt", Alias: &a}
		}()},
		{"TypeDef_anonymous", ast.TypeDefNode{
			Ident: "",
			Expr:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
		}},
		{"EnsureNode_error_and_block", func() ast.Node {
			var e ast.EnsureErrorNode = ast.EnsureErrorVar("e")
			return ast.EnsureNode{
				Variable:  ast.VariableNode{Ident: ast.Ident{ID: "x"}},
				Assertion: ast.AssertionNode{},
				Error:     &e,
				Block:     &ast.EnsureBlockNode{Body: []ast.Node{ast.IntLiteralNode{Value: 1}}},
			}
		}()},
		{"IfNode_full", ast.IfNode{
			Init: ast.AssignmentNode{
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "i"}}},
				RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 0}},
			},
			Condition: ast.BoolLiteralNode{Value: true},
			Body:      []ast.Node{ast.IntLiteralNode{Value: 1}},
			ElseIfs: []ast.ElseIfNode{
				{Condition: ast.BoolLiteralNode{Value: false}, Body: []ast.Node{ast.IntLiteralNode{Value: 2}}},
			},
			Else: &ast.ElseBlockNode{Body: []ast.Node{ast.IntLiteralNode{Value: 3}}},
		}},
		{"ImportGroupNode", ast.ImportGroupNode{Imports: []ast.ImportNode{{Path: "a"}, {Path: "b"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nh, err := h.HashNode(tt.node)
			if err != nil {
				t.Fatal(err)
			}
			if nh == 0 {
				t.Fatal("zero hash")
			}
			nh2, err := h.HashNode(tt.node)
			if err != nil {
				t.Fatal(err)
			}
			if nh != nh2 {
				t.Fatalf("HashNode not deterministic: %v vs %v", nh, nh2)
			}
		})
	}
}

func TestStructuralHasher_writeHashAndNode(t *testing.T) {
	h := New()
	var buf bytes.Buffer
	if err := h.writeHashAndNode(&buf, 9, ast.IntLiteralNode{Value: 7}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected non-empty buffer")
	}
}

func TestStructuralHasher_hashNodes_viaFunctionBodyAndCallArgs(t *testing.T) {
	h := New()
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "F"},
		Body: []ast.Node{
			ast.IntLiteralNode{Value: 1},
			ast.IntLiteralNode{Value: 2},
		},
	}
	if _, err := h.HashNode(fn); err != nil {
		t.Fatal(err)
	}
	call := ast.FunctionCallNode{
		Function: ast.Ident{ID: "g"},
		Arguments: []ast.ExpressionNode{
			ast.IntLiteralNode{Value: 3},
			ast.IntLiteralNode{Value: 4},
		},
	}
	if _, err := h.HashNode(call); err != nil {
		t.Fatal(err)
	}
}

func TestNodeHash_ToTypeIdent_zeroUsesBase58SingleDigit(t *testing.T) {
	// uint64(0) is not NilHash; toBase58(0) hits the empty-string branch (uses alphabet[0]).
	id := NodeHash(0).ToTypeIdent()
	if string(id) != "T_1" {
		t.Fatalf("got %q", id)
	}
}

func TestNodeHash_ToGuardIdent(t *testing.T) {
	hasher := New()
	node := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
	hash, err := hasher.HashNode(node)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test that ToGuardIdent produces a valid guard identifier
	guardIdent := hash.ToGuardIdent()
	if guardIdent == "" {
		t.Errorf("Expected non-empty guard identifier")
	}
	if guardIdent[0:2] != "G_" {
		t.Errorf("Expected guard identifier to start with 'G_', got %s", guardIdent)
	}
}

func TestStructuralHasher_ForBreakContinue(t *testing.T) {
	h := New()
	loop := &ast.ForNode{
		IsRange:  true,
		RangeX:   ast.VariableNode{Ident: ast.Ident{ID: "xs"}},
		RangeKey: &ast.Ident{ID: "k"},
		Body:     []ast.Node{&ast.BreakNode{}, &ast.ContinueNode{}},
	}
	h1, err := h.HashNode(loop)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := h.HashNode(loop)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatal("for loop hash not stable")
	}
	br := &ast.BreakNode{Label: &ast.Ident{ID: "L"}}
	if _, err := h.HashNode(br); err != nil {
		t.Fatal(err)
	}
}

func TestStructuralHasher_ForLoop_optionalInitPost(t *testing.T) {
	t.Parallel()
	h := New()
	body := []ast.Node{
		ast.ReturnNode{Values: []ast.ExpressionNode{ast.IntLiteralNode{Value: 0}}},
	}
	initStmt := ast.AssignmentNode{
		LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "i"}}},
		RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 0}},
		IsShort: true,
	}
	noInit := &ast.ForNode{
		Cond: ast.BoolLiteralNode{Value: true},
		Body: body,
	}
	withInit := &ast.ForNode{
		Init: initStmt,
		Cond: ast.BoolLiteralNode{Value: true},
		Body: body,
	}
	a, err := h.HashNode(noInit)
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.HashNode(withInit)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("expected different hash when Init is set")
	}
}

func TestHashNode_okErrAndIndexExpression(t *testing.T) {
	t.Parallel()
	h := New()
	idx := ast.IndexExpressionNode{
		Target: ast.VariableNode{Ident: ast.Ident{ID: "xs"}},
		Index:  ast.IntLiteralNode{Value: 0},
	}
	if _, err := h.HashNode(idx); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HashNode(ast.OkExprNode{Value: ast.IntLiteralNode{Value: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HashNode(ast.ErrExprNode{Value: ast.StringLiteralNode{Value: "e"}}); err != nil {
		t.Fatal(err)
	}
}

func TestHashNode_moreSwitchBranches(t *testing.T) {
	t.Parallel()
	h := New()
	base := ast.TypeString
	tests := []struct {
		name string
		node ast.Node
	}{
		{"TypeDefAssertionExpr_value", ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &base}}},
		{"TypeDefAssertionExpr_ptr", &ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{}}},
		{"TypeDefErrorExpr", ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
			"code": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		}}}},
		{"CommentNode", ast.CommentNode{Text: "lint"}},
		{"DeferNode", &ast.DeferNode{Call: ast.FunctionCallNode{Function: ast.Ident{ID: "close"}, Arguments: []ast.ExpressionNode{}}}},
		{"GoStmtNode", &ast.GoStmtNode{Call: ast.FunctionCallNode{Function: ast.Ident{ID: "run"}, Arguments: []ast.ExpressionNode{}}}},
		{"ForNode_range_short", &ast.ForNode{
			IsRange:    true,
			RangeX:     ast.VariableNode{Ident: ast.Ident{ID: "m"}},
			RangeKey:   &ast.Ident{ID: "k"},
			RangeValue: &ast.Ident{ID: "v"},
			RangeShort: true,
			Body:       []ast.Node{ast.IntLiteralNode{Value: 1}},
		}},
		{"TypeDef_named", ast.TypeDefNode{
			Ident: "Row",
			Expr:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
		}},
		{"ConstraintArgument_value", func() ast.Node {
			var v ast.ValueNode = ast.IntLiteralNode{Value: 42}
			return ast.ConstraintArgumentNode{Value: &v}
		}()},
		{"ConstraintArgument_shape", ast.ConstraintArgumentNode{
			Shape: &ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			}},
		}},
		{"ShapeField_type_only", ast.ShapeFieldNode{
			Type: &ast.TypeNode{Ident: ast.TypeString},
		}},
		{"ShapeField_assertion", ast.ShapeFieldNode{
			Assertion: &ast.AssertionNode{Constraints: []ast.ConstraintNode{{Name: "Min"}}},
		}},
		{"TypeGuard_destructured_param", ast.TypeGuardNode{
			Ident: "DG",
			Subject: ast.DestructuredParamNode{
				Fields: []string{"a", "z"},
				Type:   ast.TypeNode{Ident: ast.TypeInt},
			},
			Body: []ast.Node{},
		}},
		{"MapLiteral_with_type", ast.MapLiteralNode{
			Type: ast.TypeNode{Ident: ast.TypeString},
			Entries: []ast.MapEntryNode{
				{Key: ast.StringLiteralNode{Value: "k"}, Value: ast.IntLiteralNode{Value: 1}},
			},
		}},
		{"Binary_ptr", &ast.BinaryExpressionNode{
			Left: ast.IntLiteralNode{Value: 1}, Operator: ast.TokenPlus, Right: ast.IntLiteralNode{Value: 2},
		}},
		{"Unary_ptr", &ast.UnaryExpressionNode{Operator: ast.TokenMinus, Operand: ast.IntLiteralNode{Value: 3}}},
		{"TypeGuard_ptr", func() ast.Node {
			tg := ast.TypeGuardNode{
				Ident: "PG",
				Subject: ast.SimpleParamNode{
					Ident: ast.Ident{ID: "x"},
					Type:  ast.TypeNode{Ident: ast.TypeInt},
				},
				Body: []ast.Node{},
			}
			return &tg
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := h.HashNode(tt.node)
			if err != nil {
				t.Fatal(err)
			}
			if got == 0 {
				t.Fatal("zero hash")
			}
			got2, err := h.HashNode(tt.node)
			if err != nil {
				t.Fatal(err)
			}
			if got != got2 {
				t.Fatalf("determinism: %v vs %v", got, got2)
			}
		})
	}
}

func TestStructuralHasher_writeHashAndNode_propagates_HashNode_error(t *testing.T) {
	h := New()
	var buf bytes.Buffer
	sg := ast.ShapeGuardNode{
		TypeGuardNode: ast.TypeGuardNode{
			Ident: "SG",
			Subject: ast.SimpleParamNode{
				Ident: ast.Ident{ID: "x"},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
			Body: []ast.Node{},
		},
		TypeArg:   ast.TypeNode{Ident: ast.TypeString},
		FieldName: "f",
	}
	err := h.writeHashAndNode(&buf, 9, sg)
	if err == nil || !strings.Contains(err.Error(), "unsupported node type") {
		t.Fatalf("expected unsupported propagation, got %v", err)
	}
}

func TestHashNode_nil_ElseIf_ElseBlock_pointers(t *testing.T) {
	h := New()
	var ei *ast.ElseIfNode
	h1, err := h.HashNode(ei)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != NodeHash(NilHash) {
		t.Fatalf("nil *ElseIfNode: got %v want NilHash", h1)
	}
	var eb *ast.ElseBlockNode
	h2, err := h.HashNode(eb)
	if err != nil {
		t.Fatal(err)
	}
	if h2 != NodeHash(NilHash) {
		t.Fatalf("nil *ElseBlockNode: got %v want NilHash", h2)
	}
}

func TestHashNode_packageImportTypeDefAndFullIfFor(t *testing.T) {
	t.Parallel()
	h := New()
	var errVar ast.EnsureErrorNode = ast.EnsureErrorCall{ErrorType: "E", ErrorArgs: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}}
	ensureFull := ast.EnsureNode{
		Variable:  ast.VariableNode{Ident: ast.Ident{ID: "v"}},
		Assertion: ast.AssertionNode{Constraints: []ast.ConstraintNode{{Name: "Min"}}},
		Error:     &errVar,
		Block:     &ast.EnsureBlockNode{Body: []ast.Node{ast.IntLiteralNode{Value: 1}}},
	}
	if _, err := h.HashNode(ensureFull); err != nil {
		t.Fatal(err)
	}

	pkg := ast.PackageNode{Ident: ast.Ident{ID: "main"}}
	if _, err := h.HashNode(pkg); err != nil {
		t.Fatal(err)
	}
	imp := ast.ImportNode{Path: "fmt", Alias: &ast.Ident{ID: "f"}}
	if _, err := h.HashNode(imp); err != nil {
		t.Fatal(err)
	}
	ig := ast.ImportGroupNode{Imports: []ast.ImportNode{
		{Path: "a"},
		{Path: "b", Alias: &ast.Ident{ID: "bb"}},
	}}
	if _, err := h.HashNode(ig); err != nil {
		t.Fatal(err)
	}

	tdAnon := ast.TypeDefNode{Ident: "", Expr: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
		"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
	}}}}
	if _, err := h.HashNode(tdAnon); err != nil {
		t.Fatal(err)
	}

	ifFull := ast.IfNode{
		Init: ast.AssignmentNode{
			IsShort: true,
			LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "i"}}},
			RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 0}},
		},
		Condition: ast.BoolLiteralNode{Value: true},
		Body:      []ast.Node{ast.IntLiteralNode{Value: 1}},
		ElseIfs: []ast.ElseIfNode{
			{Condition: ast.BoolLiteralNode{Value: false}, Body: []ast.Node{ast.IntLiteralNode{Value: 2}}},
		},
		Else: &ast.ElseBlockNode{Body: []ast.Node{ast.IntLiteralNode{Value: 3}}},
	}
	if _, err := h.HashNode(ifFull); err != nil {
		t.Fatal(err)
	}

	post := ast.UnaryExpressionNode{Operator: ast.TokenMinusMinus, Operand: ast.VariableNode{Ident: ast.Ident{ID: "i"}}}
	forClassic := &ast.ForNode{
		Init: ifFull.Init,
		Cond: ast.BinaryExpressionNode{
			Left: ast.VariableNode{Ident: ast.Ident{ID: "i"}}, Operator: ast.TokenLess,
			Right: ast.IntLiteralNode{Value: 3},
		},
		Post: post,
		Body: []ast.Node{&ast.ContinueNode{}},
	}
	if _, err := h.HashNode(forClassic); err != nil {
		t.Fatal(err)
	}

	tgTwo := ast.TypeGuardNode{
		Ident: "TG",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "b"}, Type: ast.TypeNode{Ident: ast.TypeInt},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{Ident: ast.Ident{ID: "a"}, Type: ast.TypeNode{Ident: ast.TypeInt}},
		},
		Body: []ast.Node{},
	}
	if _, err := h.HashNode(tgTwo); err != nil {
		t.Fatal(err)
	}
}

func TestStructuralHasher_ElseIfAndElseBlockNodes(t *testing.T) {
	t.Parallel()
	h := New()
	baseStr := ast.TypeString
	ei := ast.ElseIfNode{
		Condition: ast.BinaryExpressionNode{
			Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
			Operator: ast.TokenIs,
			Right:    ast.AssertionNode{BaseType: &baseStr},
		},
		Body: []ast.Node{
			ast.ReturnNode{Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: "ok"}}},
		},
	}
	if _, err := h.HashNode(ei); err != nil {
		t.Fatalf("ElseIfNode must hash (scope collection): %v", err)
	}
	eb := ast.ElseBlockNode{
		Body: []ast.Node{
			ast.ReturnNode{Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: ""}}},
		},
	}
	if _, err := h.HashNode(eb); err != nil {
		t.Fatalf("ElseBlockNode must hash (scope collection): %v", err)
	}
	pEI := &ei
	if _, err := h.HashNode(pEI); err != nil {
		t.Fatalf("*ElseIfNode: %v", err)
	}
	pEB := &eb
	if _, err := h.HashNode(pEB); err != nil {
		t.Fatalf("*ElseBlockNode: %v", err)
	}
}
