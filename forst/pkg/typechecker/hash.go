package typechecker

import (
	"encoding/binary"
	"fmt"
	"forst/pkg/ast"
	"hash/fnv"
	"io"
	"sort"
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

// writeHash is a helper function to handle binary.Write errors
func writeHash(w io.Writer, data interface{}) {
	if err := binary.Write(w, binary.LittleEndian, data); err != nil {
		panic(fmt.Sprintf("failed to write hash: %v", err))
	}
}

// HashNode generates a structural hash for an AST node
func (h *StructuralHasher) HashNode(node ast.Node) NodeHash {
	hasher := fnv.New64a()

	switch n := node.(type) {
	case ast.BinaryExpressionNode:
		// Hash the kind
		writeHash(hasher, NodeKind["BinaryExpression"])
		// Hash the operator
		writeHash(hasher, h.HashTokenType(n.Operator))
		// Hash the operands recursively
		writeHash(hasher, h.HashNode(n.Left))
		writeHash(hasher, h.HashNode(n.Right))

	case ast.UnaryExpressionNode:
		writeHash(hasher, NodeKind["UnaryExpression"])
		writeHash(hasher, h.HashTokenType(n.Operator))
		writeHash(hasher, h.HashNode(n.Operand))

	case ast.IntLiteralNode:
		writeHash(hasher, NodeKind["IntLiteral"])
		writeHash(hasher, n.Value)

	case ast.FloatLiteralNode:
		writeHash(hasher, NodeKind["FloatLiteral"])
		writeHash(hasher, n.Value)

	case ast.StringLiteralNode:
		writeHash(hasher, NodeKind["StringLiteral"])
		writeHash(hasher, []byte(n.Value))

	case ast.VariableNode:
		writeHash(hasher, NodeKind["Variable"])
		writeHash(hasher, []byte(n.Ident.Id))

	case ast.FunctionNode:
		writeHash(hasher, NodeKind["Function"])
		writeHash(hasher, h.HashNodes(n.Body))

	case ast.FunctionCallNode:
		writeHash(hasher, NodeKind["FunctionCall"])
		writeHash(hasher, []byte(n.Function.Id))
		nodes := make([]ast.Node, len(n.Arguments))
		for i, arg := range n.Arguments {
			nodes[i] = arg
		}
		writeHash(hasher, h.HashNodes(nodes))

	case ast.EnsureNode:
		writeHash(hasher, NodeKind["Ensure"])
		writeHash(hasher, h.HashNode(n.Assertion))
		if n.Error != nil {
			writeHash(hasher, []byte((*n.Error).String()))
		}
		if n.Block != nil {
			writeHash(hasher, h.HashNodes(n.Block.Body))
		}

	case ast.ShapeNode:
		writeHash(hasher, NodeKind["Shape"])
		// Convert map to sorted slice of fields for deterministic ordering
		fields := make([]struct {
			name  string
			field ast.ShapeFieldNode
		}, 0, len(n.Fields))
		for name, field := range n.Fields {
			fields = append(fields, struct {
				name  string
				field ast.ShapeFieldNode
			}{name, field})
		}
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].name < fields[j].name
		})
		// Hash each field in sorted order
		for _, f := range fields {
			writeHash(hasher, []byte(f.name))
			writeHash(hasher, h.HashNode(f.field))
		}

	case ast.ShapeFieldNode:
		writeHash(hasher, NodeKind["ShapeField"])
		if n.Assertion != nil {
			writeHash(hasher, h.HashNode(*n.Assertion))
		}
		if n.Shape != nil {
			writeHash(hasher, h.HashNode(*n.Shape))
		}

	case ast.AssertionNode:
		writeHash(hasher, NodeKind["Assertion"])
		if n.BaseType != nil {
			writeHash(hasher, []byte(*n.BaseType))
		}
		// Sort constraints for deterministic ordering
		constraints := make([]ast.ConstraintNode, len(n.Constraints))
		copy(constraints, n.Constraints)
		sort.Slice(constraints, func(i, j int) bool {
			return constraints[i].Name < constraints[j].Name
		})
		// Hash each constraint in sorted order
		for _, constraint := range constraints {
			writeHash(hasher, h.HashNode(constraint))
		}

	case ast.ConstraintNode:
		writeHash(hasher, NodeKind["Constraint"])
		writeHash(hasher, []byte(n.Name))
		nodes := make([]ast.Node, len(n.Args))
		for i, arg := range n.Args {
			nodes[i] = arg
		}
		writeHash(hasher, h.HashNodes(nodes))

	case ast.ConstraintArgumentNode:
		writeHash(hasher, NodeKind["ConstraintArgument"])
		if n.Value != nil {
			writeHash(hasher, h.HashNode(*n.Value))
		}
		if n.Shape != nil {
			writeHash(hasher, h.HashNode(*n.Shape))
		}
	case ast.PackageNode:
		writeHash(hasher, NodeKind["Package"])
		writeHash(hasher, []byte(n.Ident.Id))
	case ast.ImportNode:
		writeHash(hasher, NodeKind["Import"])
		writeHash(hasher, []byte(n.Path))
		if n.Alias != nil {
			writeHash(hasher, []byte(n.Alias.Id))
		}
	case ast.TypeDefNode:
		writeHash(hasher, NodeKind["TypeDef"])
		writeHash(hasher, []byte(n.Expr.String()))
		writeHash(hasher, []byte(n.Ident))
	case ast.ReturnNode:
		writeHash(hasher, NodeKind["Return"])
		writeHash(hasher, h.HashNode(n.Value))
	case ast.TypeNode:
		writeHash(hasher, NodeKind["Type"])
		writeHash(hasher, []byte(n.Ident))
	case ast.SimpleParamNode:
		writeHash(hasher, NodeKind["SimpleParam"])
		writeHash(hasher, []byte(n.Ident.Id))
		writeHash(hasher, h.HashNode(n.Type))
	case ast.DestructuredParamNode:
		writeHash(hasher, NodeKind["DestructuredParam"])
		// Sort fields for deterministic ordering
		fields := make([]string, len(n.Fields))
		copy(fields, n.Fields)
		sort.Strings(fields)
		for _, field := range fields {
			writeHash(hasher, []byte(field))
		}
		writeHash(hasher, h.HashNode(n.Type))
	case *ast.AssertionNode:
		writeHash(hasher, NodeKind["Assertion"])
		if n.BaseType != nil {
			writeHash(hasher, []byte(*n.BaseType))
		}
		// Sort constraints for deterministic ordering
		constraints := make([]ast.ConstraintNode, len(n.Constraints))
		copy(constraints, n.Constraints)
		sort.Slice(constraints, func(i, j int) bool {
			return constraints[i].Name < constraints[j].Name
		})
		// Hash each constraint in sorted order
		for _, constraint := range constraints {
			writeHash(hasher, h.HashNode(constraint))
		}
	case *ast.ShapeNode:
		writeHash(hasher, NodeKind["Shape"])
		// Convert map to sorted slice of fields for deterministic ordering
		fields := make([]struct {
			name  string
			field ast.ShapeFieldNode
		}, 0, len(n.Fields))
		for name, field := range n.Fields {
			fields = append(fields, struct {
				name  string
				field ast.ShapeFieldNode
			}{name, field})
		}
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].name < fields[j].name
		})
		// Hash each field in sorted order
		for _, f := range fields {
			writeHash(hasher, []byte(f.name))
			writeHash(hasher, h.HashNode(f.field))
		}
	case *ast.ShapeFieldNode:
		writeHash(hasher, NodeKind["ShapeField"])
		if n.Assertion != nil {
			writeHash(hasher, h.HashNode(*n.Assertion))
		}
		if n.Shape != nil {
			writeHash(hasher, h.HashNode(*n.Shape))
		}
	case ast.ImportGroupNode:
		writeHash(hasher, NodeKind["ImportGroup"])
		// Sort imports for deterministic ordering
		imports := make([]ast.ImportNode, len(n.Imports))
		copy(imports, n.Imports)
		sort.Slice(imports, func(i, j int) bool {
			return imports[i].Path < imports[j].Path
		})
		for _, importNode := range imports {
			writeHash(hasher, h.HashNode(importNode))
		}
	case ast.AssignmentNode:
		writeHash(hasher, NodeKind["Assignment"])
		for _, lValue := range n.LValues {
			writeHash(hasher, h.HashNode(lValue))
		}
		for _, rValue := range n.RValues {
			writeHash(hasher, h.HashNode(rValue))
		}
	case *ast.EnsureBlockNode:
		writeHash(hasher, NodeKind["EnsureBlock"])
		for _, node := range n.Body {
			writeHash(hasher, h.HashNode(node))
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
