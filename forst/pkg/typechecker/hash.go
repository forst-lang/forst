package typechecker

import (
	"encoding/binary"
	"fmt"
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
	"Function":         9,
	"Ensure":           10,
}

// HashNodes generates a structural hash for multiple AST nodes
func (h *StructuralHasher) HashNodes(nodes []ast.Node) NodeHash {
	hasher := fnv.New64a()
	for _, node := range nodes {
		binary.Write(hasher, binary.LittleEndian, h.HashNode(node))
	}
	return NodeHash(hasher.Sum64())
}

// HashNode generates a structural hash for an AST node
func (h *StructuralHasher) HashNode(node ast.Node) NodeHash {
	hasher := fnv.New64a()

	switch n := node.(type) {
	case ast.BinaryExpressionNode:
		// Hash the kind
		binary.Write(hasher, binary.LittleEndian, NodeKind["BinaryExpression"])
		// Hash the operator
		binary.Write(hasher, binary.LittleEndian, h.HashTokenType(n.Operator))
		// Hash the operands recursively
		binary.Write(hasher, binary.LittleEndian, h.HashNode(n.Left))
		binary.Write(hasher, binary.LittleEndian, h.HashNode(n.Right))

	case ast.UnaryExpressionNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["UnaryExpression"])
		binary.Write(hasher, binary.LittleEndian, h.HashTokenType(n.Operator))
		binary.Write(hasher, binary.LittleEndian, h.HashNode(n.Operand))

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
		hasher.Write([]byte(n.Ident.Id))

	case ast.FunctionNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Function"])
		binary.Write(hasher, binary.LittleEndian, h.HashNodes(n.Body))

	case ast.FunctionCallNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["FunctionCall"])
		hasher.Write([]byte(n.Function.Id))
		nodes := make([]ast.Node, len(n.Arguments))
		for i, arg := range n.Arguments {
			nodes[i] = arg
		}
		binary.Write(hasher, binary.LittleEndian, h.HashNodes(nodes))

	case ast.EnsureNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Ensure"])
		binary.Write(hasher, binary.LittleEndian, h.HashNode(n.Assertion))
		if n.Error != nil {
			binary.Write(hasher, binary.LittleEndian, []byte((*n.Error).String()))
		}
		if n.Block != nil {
			binary.Write(hasher, binary.LittleEndian, h.HashNodes(n.Block.Body))
		}

	case ast.ShapeNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Shape"])
		// Convert map to slice of nodes
		fields := make([]ast.Node, 0, len(n.Fields))
		for _, field := range n.Fields {
			fields = append(fields, field)
		}
		binary.Write(hasher, binary.LittleEndian, h.HashNodes(fields))

	case ast.ShapeFieldNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["ShapeField"])
		if n.Assertion != nil {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(*n.Assertion))
		}
		if n.Shape != nil {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(*n.Shape))
		}

	case ast.AssertionNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Assertion"])
		if n.BaseType != nil {
			binary.Write(hasher, binary.LittleEndian, []byte(*n.BaseType))
		}
		// Convert constraints to []ast.Node
		nodes := make([]ast.Node, len(n.Constraints))
		for i, constraint := range n.Constraints {
			nodes[i] = constraint
		}
		binary.Write(hasher, binary.LittleEndian, h.HashNodes(nodes))

	case ast.ConstraintNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Constraint"])
		binary.Write(hasher, binary.LittleEndian, []byte(n.Name))
		// Convert args to []ast.Node
		nodes := make([]ast.Node, len(n.Args))
		for i, arg := range n.Args {
			nodes[i] = arg
		}
		binary.Write(hasher, binary.LittleEndian, h.HashNodes(nodes))

	case ast.ConstraintArgumentNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["ConstraintArgument"])
		if n.Value != nil {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(*n.Value))
		}
		if n.Shape != nil {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(*n.Shape))
		}
	case ast.PackageNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Package"])
		binary.Write(hasher, binary.LittleEndian, []byte(n.Ident.Id))
	case ast.ImportNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Import"])
		binary.Write(hasher, binary.LittleEndian, []byte(n.Path))
		if n.Alias != nil {
			binary.Write(hasher, binary.LittleEndian, []byte(n.Alias.Id))
		}
	case ast.TypeDefNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["TypeDef"])
		binary.Write(hasher, binary.LittleEndian, []byte(n.Expr.String()))
		binary.Write(hasher, binary.LittleEndian, []byte(n.Ident))
	case ast.ReturnNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Return"])
		binary.Write(hasher, binary.LittleEndian, h.HashNode(n.Value))
	case ast.TypeNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Type"])
		binary.Write(hasher, binary.LittleEndian, []byte(n.Ident))
	case ast.SimpleParamNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["SimpleParam"])
		binary.Write(hasher, binary.LittleEndian, []byte(n.Ident.Id))
		binary.Write(hasher, binary.LittleEndian, h.HashNode(n.Type))
	case ast.DestructuredParamNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["DestructuredParam"])
		for _, field := range n.Fields {
			binary.Write(hasher, binary.LittleEndian, []byte(field))
		}
		binary.Write(hasher, binary.LittleEndian, h.HashNode(n.Type))
	case *ast.AssertionNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Assertion"])
		if n.BaseType != nil {
			binary.Write(hasher, binary.LittleEndian, []byte(*n.BaseType))
		}
		nodes := make([]ast.Node, len(n.Constraints))
		for i, constraint := range n.Constraints {
			nodes[i] = constraint
		}
		binary.Write(hasher, binary.LittleEndian, h.HashNodes(nodes))
	case *ast.ShapeNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Shape"])
		fields := make([]ast.Node, 0, len(n.Fields))
		for _, field := range n.Fields {
			fields = append(fields, field)
		}
		binary.Write(hasher, binary.LittleEndian, h.HashNodes(fields))
	case *ast.ShapeFieldNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["ShapeField"])
		if n.Assertion != nil {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(*n.Assertion))
		}
		if n.Shape != nil {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(*n.Shape))
		}
	case ast.ImportGroupNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["ImportGroup"])
		for _, importNode := range n.Imports {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(importNode))
		}
	case ast.AssignmentNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["Assignment"])
		for _, lValue := range n.LValues {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(lValue))
		}
		for _, rValue := range n.RValues {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(rValue))
		}
	case *ast.EnsureBlockNode:
		binary.Write(hasher, binary.LittleEndian, NodeKind["EnsureBlock"])
		for _, node := range n.Body {
			binary.Write(hasher, binary.LittleEndian, h.HashNode(node))
		}
	default:
		panic(fmt.Sprintf("unsupported node type: %T", n))
	}

	return NodeHash(hasher.Sum64())
}

// Generates a structural hash for a token type
func (h *StructuralHasher) HashTokenType(tokenType ast.TokenIdent) NodeHash {
	hasher := fnv.New64a()
	hasher.Write([]byte(string(tokenType)))
	return NodeHash(hasher.Sum64())
}

// This is the initial hash value from fnv.New64a() when no data is written
// It represents a nil/empty hash, so we give it a special type name
const NIL_HASH = uint64(14695981039346656037)

// Generates a string name for a type based on its hash value
func (h NodeHash) ToTypeIdent() ast.TypeIdent {
	// Convert hash to base58 string
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	num := uint64(h)
	b58 := ""

	if num == NIL_HASH {
		return ast.TypeIdent("T_Invalid")
	}

	for num > 0 {
		remainder := num % 58
		b58 = string(alphabet[remainder]) + b58
		num = num / 58
	}

	if b58 == "" {
		b58 = string(alphabet[0])
	}

	return ast.TypeIdent("T_" + b58)
}
