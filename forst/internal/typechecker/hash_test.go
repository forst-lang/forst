package typechecker

import (
	"forst/internal/ast"
	"testing"
)

func TestStructuralHasher_Consistency(t *testing.T) {
	hasher := NewStructuralHasher()

	// Test that same structures produce same hashes
	t.Run("same structures produce same hashes", func(t *testing.T) {
		// Create two identical variable nodes
		node1 := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
		node2 := ast.VariableNode{Ident: ast.Ident{ID: "x"}}

		hash1 := hasher.HashNode(node1)
		hash2 := hasher.HashNode(node2)

		if hash1 != hash2 {
			t.Errorf("Expected identical nodes to produce same hash, got %v and %v", hash1, hash2)
		}
	})

	t.Run("different structures produce different hashes", func(t *testing.T) {
		// Create two different variable nodes
		node1 := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
		node2 := ast.VariableNode{Ident: ast.Ident{ID: "y"}}

		hash1 := hasher.HashNode(node1)
		hash2 := hasher.HashNode(node2)

		if hash1 == hash2 {
			t.Errorf("Expected different nodes to produce different hashes")
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

		hash1 := hasher.HashNode(map1)
		hash2 := hasher.HashNode(map2)

		if hash1 != hash2 {
			t.Errorf("Expected maps with same fields in different order to produce same hash")
		}
	})

	t.Run("pointer vs value types produce same hash", func(t *testing.T) {
		// Create same structure as pointer and value
		valueNode := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
		ptrNode := &valueNode

		hash1 := hasher.HashNode(valueNode)
		hash2 := hasher.HashNode(ptrNode)

		if hash1 != hash2 {
			t.Errorf("Expected pointer and value types to produce same hash")
		}
	})

	t.Run("nil values handled correctly", func(t *testing.T) {
		// Test nil pointer
		var nilNode *ast.VariableNode
		hash1 := hasher.HashNode(nilNode)

		// Test node with nil fields
		node := ast.EnsureNode{
			Variable:  ast.VariableNode{Ident: ast.Ident{ID: "x"}},
			Assertion: ast.AssertionNode{},
			Error:     nil,
			Block:     nil,
		}
		hash2 := hasher.HashNode(node)

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

		hash := hasher.HashNode(outer)
		if hash == 0 {
			t.Errorf("Expected non-zero hash for nested structure")
		}
	})

	t.Run("type identifiers hash correctly", func(t *testing.T) {
		// Test type identifiers
		type1 := ast.TypeNode{Ident: "int"}
		type2 := ast.TypeNode{Ident: "string"}

		hash1 := hasher.HashNode(type1)
		hash2 := hasher.HashNode(type2)

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

		hash1 := hasher.HashNode(param1)
		hash2 := hasher.HashNode(param2)

		if hash1 == hash2 {
			t.Errorf("Expected different parameters to produce different hashes")
		}
	})
}

func TestNodeHash_ToTypeIdent(t *testing.T) {
	hasher := NewStructuralHasher()
	node := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
	hash := hasher.HashNode(node)

	// Test that ToTypeIdent produces a valid type identifier
	typeIdent := hash.ToTypeIdent()
	if typeIdent == "" {
		t.Errorf("Expected non-empty type identifier")
	}
	if typeIdent[0:2] != "T_" {
		t.Errorf("Expected type identifier to start with 'T_', got %s", typeIdent)
	}
}

func TestNodeHash_ToGuardIdent(t *testing.T) {
	hasher := NewStructuralHasher()
	node := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
	hash := hasher.HashNode(node)

	// Test that ToGuardIdent produces a valid guard identifier
	guardIdent := hash.ToGuardIdent()
	if guardIdent == "" {
		t.Errorf("Expected non-empty guard identifier")
	}
	if guardIdent[0:2] != "G_" {
		t.Errorf("Expected guard identifier to start with 'G_', got %s", guardIdent)
	}
}
