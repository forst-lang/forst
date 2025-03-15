package typechecker

import (
	"encoding/binary"
	"forst/pkg/ast"
	"hash/fnv"
)

// StructuralHasher generates and tracks structural hashes
type StructuralHasher struct {
	// Map from hash to type (uint64 is the FNV-1a 64-bit hash)
	hashes map[uint64]ast.TypeNode
}

func NewStructuralHasher() *StructuralHasher {
	return &StructuralHasher{
		hashes: make(map[uint64]ast.TypeNode),
	}
}

type NodeHash uint64

// NodeKind maps AST node types to unique uint8 identifiers for hashing
var NodeKind = map[string]uint8{
	"BinaryExpression": 1,
	"IntLiteral":       2,
	"FloatLiteral":     3,
	"StringLiteral":    4,
	"Variable":         5,
	"UnaryExpression":  6,
	"FunctionCall":     7,
	"BoolLiteral":      8,
}

// Hash generates a structural hash for an AST node
func (h *StructuralHasher) Hash(node ast.Node) NodeHash {
	hasher := fnv.New64a()

	switch n := node.(type) {
	case ast.BinaryExpressionNode:
		// Hash the kind
		binary.Write(hasher, binary.LittleEndian, NodeKind["BinaryExpression"])
		// Hash the operator
		binary.Write(hasher, binary.LittleEndian, h.HashTokenType(n.Operator))
		// Hash the operands recursively
		binary.Write(hasher, binary.LittleEndian, h.Hash(n.Left))
		binary.Write(hasher, binary.LittleEndian, h.Hash(n.Right))

	case ast.IntLiteralNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["IntLiteral"])
		binary.Write(hasher, binary.LittleEndian, n.Value)

	case ast.FloatLiteralNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["FloatLiteral"])
		binary.Write(hasher, binary.LittleEndian, n.Value)

	case ast.StringLiteralNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["StringLiteral"])
		hasher.Write([]byte(n.Value))

	case ast.VariableNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Variable"])
		hasher.Write([]byte(n.Ident.Name))
	}

	return NodeHash(hasher.Sum64())
}

// HashTokenType generates a structural hash for a token type
func (h *StructuralHasher) HashTokenType(tokenType ast.TokenType) NodeHash {
	hasher := fnv.New64a()
	hasher.Write([]byte(string(tokenType)))
	return NodeHash(hasher.Sum64())
}
