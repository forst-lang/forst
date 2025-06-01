package typechecker

import (
	"encoding/binary"
	"fmt"
	"forst/internal/ast"
	"hash/fnv"
	"io"
	"reflect"
	"sort"
)

// StructuralHasher generates and tracks structural hashes
type StructuralHasher struct {
	// Map from hash to type (uint64 is the FNV-1a 64-bit hash)
	hashes map[uint64]ast.TypeNode
}

// NewStructuralHasher creates a new StructuralHasher
func NewStructuralHasher() *StructuralHasher {
	return &StructuralHasher{
		hashes: make(map[uint64]ast.TypeNode),
	}
}

// NodeHash is a unique identifier for an AST node
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
	"TypeGuard":        11,
	"TypeDefAssertion": 12,
	"If":               13,
	"Reference":        14,
	"MapLiteral":       15,
}

// hashNodes generates a structural hash for multiple AST nodes
func (h *StructuralHasher) hashNodes(nodes []ast.Node) NodeHash {
	hasher := fnv.New64a()
	for _, node := range nodes {
		writeHash(hasher, h.HashNode(node))
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

	// Handle nil and typed nils
	if node == nil || (isNilPointer(node)) {
		return NodeHash(NilHash)
	}

	switch n := node.(type) {
	case ast.TypeDefAssertionExpr:
		writeHash(hasher, NodeKind["TypeDefAssertion"])
		writeHash(hasher, h.HashNode(n.Assertion))

	case *ast.TypeDefAssertionExpr:
		return h.HashNode(*n)

	case ast.TypeDefBinaryExpr:
		writeHash(hasher, NodeKind["TypeDefBinaryExpr"])
		writeHash(hasher, h.HashTokenType(n.Op))
		writeHash(hasher, h.HashNode(n.Left.(ast.Node)))
		writeHash(hasher, h.HashNode(n.Right.(ast.Node)))

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

	case ast.BoolLiteralNode:
		writeHash(hasher, NodeKind["BoolLiteral"])
		writeHash(hasher, n.Value)

	case ast.FloatLiteralNode:
		writeHash(hasher, NodeKind["FloatLiteral"])
		writeHash(hasher, n.Value)

	case ast.StringLiteralNode:
		writeHash(hasher, NodeKind["StringLiteral"])
		writeHash(hasher, []byte(n.Value))

	case ast.VariableNode:
		writeHash(hasher, NodeKind["Variable"])
		writeHash(hasher, []byte(n.Ident.ID))

	case ast.FunctionNode:
		writeHash(hasher, NodeKind["Function"])
		writeHash(hasher, h.hashNodes(n.Body))

	case ast.FunctionCallNode:
		writeHash(hasher, NodeKind["FunctionCall"])
		writeHash(hasher, []byte(n.Function.ID))
		nodes := make([]ast.Node, len(n.Arguments))
		for i, arg := range n.Arguments {
			nodes[i] = arg
		}
		writeHash(hasher, h.hashNodes(nodes))

	case ast.EnsureNode:
		writeHash(hasher, NodeKind["Ensure"])
		writeHash(hasher, h.HashNode(n.Assertion))
		if n.Error != nil {
			writeHash(hasher, []byte((*n.Error).String()))
		}
		if n.Block != nil {
			writeHash(hasher, h.hashNodes(n.Block.Body))
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
		writeHash(hasher, h.hashNodes(nodes))

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
		writeHash(hasher, []byte(n.Ident.ID))
	case ast.ImportNode:
		writeHash(hasher, NodeKind["Import"])
		writeHash(hasher, []byte(n.Path))
		if n.Alias != nil {
			writeHash(hasher, []byte(n.Alias.ID))
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
		writeHash(hasher, []byte(n.Ident.ID))
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
	case ast.TypeGuardNode:
		writeHash(hasher, NodeKind["TypeGuard"])
		writeHash(hasher, []byte(n.Ident))
		// Sort parameters for deterministic ordering
		params := make([]ast.ParamNode, len(n.Parameters()))
		copy(params, n.Parameters())
		sort.Slice(params, func(i, j int) bool {
			var iName, jName string
			switch p := params[i].(type) {
			case ast.SimpleParamNode:
				iName = string(p.Ident.ID)
			case ast.DestructuredParamNode:
				iName = p.Fields[0] // Use first field name for sorting
			}
			switch p := params[j].(type) {
			case ast.SimpleParamNode:
				jName = string(p.Ident.ID)
			case ast.DestructuredParamNode:
				jName = p.Fields[0] // Use first field name for sorting
			}
			return iName < jName
		})
		for _, param := range params {
			writeHash(hasher, h.HashNode(param))
		}
		writeHash(hasher, h.hashNodes(n.Body))
	case *ast.TypeGuardNode:
		return h.HashNode(*n)
	case ast.IfNode:
		writeHash(hasher, NodeKind["If"])
		if n.Init != nil {
			writeHash(hasher, h.HashNode(n.Init))
		}
		writeHash(hasher, h.HashNode(n.Condition))
		writeHash(hasher, h.hashNodes(n.Body))
		for _, elseIf := range n.ElseIfs {
			writeHash(hasher, h.HashNode(elseIf.Condition))
			writeHash(hasher, h.hashNodes(elseIf.Body))
		}
		if n.Else != nil {
			writeHash(hasher, h.hashNodes(n.Else.Body))
		}
	case *ast.IfNode:
		return h.HashNode(*n)
	case ast.ReferenceNode:
		writeHash(hasher, NodeKind["Reference"])
		writeHash(hasher, h.HashNode(n.Value))
	case ast.MapLiteralNode:
		writeHash(hasher, NodeKind["MapLiteral"])
		// Hash the type
		writeHash(hasher, h.HashNode(n.Type))
		// Sort entries by key for deterministic ordering
		entries := make([]struct {
			key   ast.Node
			value ast.Node
		}, len(n.Entries))
		for i, entry := range n.Entries {
			entries[i] = struct {
				key   ast.Node
				value ast.Node
			}{entry.Key, entry.Value}
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].key.String() < entries[j].key.String()
		})
		// Hash each entry in sorted order
		for _, entry := range entries {
			writeHash(hasher, h.HashNode(entry.key))
			writeHash(hasher, h.HashNode(entry.value))
		}
	case *ast.MapLiteralNode:
		return h.HashNode(*n)
	case *ast.VariableNode:
		return h.HashNode(*n)
	case *ast.BinaryExpressionNode:
		return h.HashNode(*n)
	case *ast.UnaryExpressionNode:
		return h.HashNode(*n)
	case *ast.IntLiteralNode:
		return h.HashNode(*n)
	case *ast.FloatLiteralNode:
		return h.HashNode(*n)
	case *ast.StringLiteralNode:
		return h.HashNode(*n)
	case *ast.BoolLiteralNode:
		return h.HashNode(*n)
	case *ast.FunctionNode:
		return h.HashNode(*n)
	case *ast.FunctionCallNode:
		return h.HashNode(*n)
	case *ast.EnsureNode:
		return h.HashNode(*n)
	case *ast.ReferenceNode:
		return h.HashNode(*n)
	default:
		panic(fmt.Sprintf("unsupported node type: %T", n))
	}

	return NodeHash(hasher.Sum64())
}

// HashTokenType generates a structural hash for a token type
func (h *StructuralHasher) HashTokenType(tokenType ast.TokenIdent) NodeHash {
	hasher := fnv.New64a()
	hasher.Write([]byte(string(tokenType)))
	return NodeHash(hasher.Sum64())
}

// NilHash is the initial hash value from fnv.New64a() when no data is written
// It represents a nil/empty hash, so we give it a special type name
const NilHash = uint64(14695981039346656037)

// toBase58 converts a NodeHash to a base58 string
func (h NodeHash) toBase58() string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	num := uint64(h)
	b58 := ""

	for num > 0 {
		remainder := num % 58
		b58 = string(alphabet[remainder]) + b58
		num = num / 58
	}

	if b58 == "" {
		b58 = string(alphabet[0])
	}
	return b58
}

// ToTypeIdent generates a string name for a type based on its hash value
func (h NodeHash) ToTypeIdent() ast.TypeIdent {
	if uint64(h) == NilHash {
		return ast.TypeIdent("T_Invalid")
	}
	return ast.TypeIdent("T_" + h.toBase58())
}

// ToGuardIdent generates a string name for a guard function based on its hash value
func (h NodeHash) ToGuardIdent() ast.TypeIdent {
	if uint64(h) == NilHash {
		return ast.TypeIdent("G_Invalid")
	}
	return ast.TypeIdent("G_" + h.toBase58())
}

// Helper to check for typed nil pointers
func isNilPointer(i interface{}) bool {
	if i == nil {
		return true
	}
	// Use reflection to check for nil pointer
	// (avoiding import cycle by not using ast.Node directly)
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Ptr && v.IsNil()
}
