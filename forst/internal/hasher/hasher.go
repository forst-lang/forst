package hasher

import (
	"encoding/binary"
	"fmt"
	"forst/internal/ast"
	"hash/fnv"
	"io"
	"reflect"
	"sort"
)

// NodeHash is a unique identifier for an AST node
type NodeHash uint64

// StructuralHasher generates and tracks structural hashes
type StructuralHasher struct {
	// Map from hash to type (uint64 is the FNV-1a 64-bit hash)
	hashes map[NodeHash]ast.TypeNode
}

// New creates a new StructuralHasher
func New() *StructuralHasher {
	return &StructuralHasher{
		hashes: make(map[NodeHash]ast.TypeNode),
	}
}

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
	"NilLiteral":       16,
}

// hashNodes generates a structural hash for multiple AST nodes
func (h *StructuralHasher) hashNodes(nodes []ast.Node) (NodeHash, error) {
	hasher := fnv.New64a()
	for _, node := range nodes {
		hash, err := h.HashNode(node)
		if err != nil {
			return 0, err
		}
		if err := writeHash(hasher, hash); err != nil {
			return 0, err
		}
	}
	return NodeHash(hasher.Sum64()), nil
}

// writeHash is a helper function to handle binary.Write errors
func writeHash(w io.Writer, data interface{}) error {
	if err := binary.Write(w, binary.LittleEndian, data); err != nil {
		return fmt.Errorf("failed to write hash: %v", err)
	}
	return nil
}

// writeHashes writes multiple values to the hasher, handling errors
func (h *StructuralHasher) writeHashes(w io.Writer, values ...interface{}) error {
	for _, v := range values {
		if err := writeHash(w, v); err != nil {
			return err
		}
	}
	return nil
}

// writeHashAndNode is a helper that writes a hash and a node's hash, handling errors
func (h *StructuralHasher) writeHashAndNode(w io.Writer, kind uint8, node ast.Node) error {
	if err := writeHash(w, kind); err != nil {
		return err
	}
	hash, err := h.HashNode(node)
	if err != nil {
		return err
	}
	return writeHash(w, hash)
}

// HashNode generates a structural hash for an AST node
func (h *StructuralHasher) HashNode(node ast.Node) (NodeHash, error) {
	hasher := fnv.New64a()

	// Handle nil and typed nils
	if node == nil || (isNilPointer(node)) {
		return NodeHash(NilHash), nil
	}

	switch n := node.(type) {
	case ast.TypeDefAssertionExpr:
		if err := h.writeHashes(hasher, NodeKind["TypeDefAssertion"]); err != nil {
			return 0, err
		}
		hash, err := h.HashNode(n.Assertion)
		if err != nil {
			return 0, err
		}
		if err := h.writeHashes(hasher, hash); err != nil {
			return 0, err
		}

	case *ast.TypeDefAssertionExpr:
		return h.HashNode(*n)

	case ast.TypeDefBinaryExpr:
		if err := h.writeHashes(hasher,
			NodeKind["TypeDefBinaryExpr"],
			h.HashTokenType(n.Op),
		); err != nil {
			return 0, err
		}
		leftHash, err := h.HashNode(n.Left.(ast.Node))
		if err != nil {
			return 0, err
		}
		rightHash, err := h.HashNode(n.Right.(ast.Node))
		if err != nil {
			return 0, err
		}
		if err := h.writeHashes(hasher, leftHash, rightHash); err != nil {
			return 0, err
		}

	case ast.BinaryExpressionNode:
		if err := h.writeHashes(hasher,
			NodeKind["BinaryExpression"],
			h.HashTokenType(n.Operator),
		); err != nil {
			return 0, err
		}
		leftHash, err := h.HashNode(n.Left)
		if err != nil {
			return 0, err
		}
		rightHash, err := h.HashNode(n.Right)
		if err != nil {
			return 0, err
		}
		if err := h.writeHashes(hasher, leftHash, rightHash); err != nil {
			return 0, err
		}

	case ast.UnaryExpressionNode:
		if err := h.writeHashes(hasher,
			NodeKind["UnaryExpression"],
			h.HashTokenType(n.Operator),
		); err != nil {
			return 0, err
		}
		operandHash, err := h.HashNode(n.Operand)
		if err != nil {
			return 0, err
		}
		if err := h.writeHashes(hasher, operandHash); err != nil {
			return 0, err
		}

	case ast.IntLiteralNode:
		if err := h.writeHashes(hasher,
			NodeKind["IntLiteral"],
			n.Value,
		); err != nil {
			return 0, err
		}

	case ast.BoolLiteralNode:
		if err := h.writeHashes(hasher,
			NodeKind["BoolLiteral"],
			n.Value,
		); err != nil {
			return 0, err
		}

	case ast.FloatLiteralNode:
		if err := h.writeHashes(hasher,
			NodeKind["FloatLiteral"],
			n.Value,
		); err != nil {
			return 0, err
		}

	case ast.StringLiteralNode:
		if err := h.writeHashes(hasher,
			NodeKind["StringLiteral"],
			[]byte(n.Value),
		); err != nil {
			return 0, err
		}

	case ast.VariableNode:
		if err := h.writeHashes(hasher,
			NodeKind["Variable"],
			[]byte(n.Ident.ID),
		); err != nil {
			return 0, err
		}

	case ast.FunctionNode:
		if err := h.writeHashes(hasher, NodeKind["Function"]); err != nil {
			return 0, err
		}
		hash, err := h.hashNodes(n.Body)
		if err != nil {
			return 0, err
		}
		if err := h.writeHashes(hasher, hash); err != nil {
			return 0, err
		}

	case ast.FunctionCallNode:
		if err := h.writeHashes(hasher,
			NodeKind["FunctionCall"],
			[]byte(n.Function.ID),
		); err != nil {
			return 0, err
		}
		nodes := make([]ast.Node, len(n.Arguments))
		for i, arg := range n.Arguments {
			nodes[i] = arg
		}
		hash, err := h.hashNodes(nodes)
		if err != nil {
			return 0, err
		}
		if err := h.writeHashes(hasher, hash); err != nil {
			return 0, err
		}

	case ast.EnsureNode:
		if err := h.writeHashes(hasher, NodeKind["Ensure"]); err != nil {
			return 0, err
		}
		hash, err := h.HashNode(n.Assertion)
		if err != nil {
			return 0, err
		}
		if err := h.writeHashes(hasher, hash); err != nil {
			return 0, err
		}
		if n.Error != nil {
			if err := h.writeHashes(hasher, []byte((*n.Error).String())); err != nil {
				return 0, err
			}
		}
		if n.Block != nil {
			hash, err := h.hashNodes(n.Block.Body)
			if err != nil {
				return 0, err
			}
			if err := h.writeHashes(hasher, hash); err != nil {
				return 0, err
			}
		}

	case ast.ShapeNode:
		h.writeHashes(hasher, NodeKind["Shape"])
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
			h.writeHashes(hasher, []byte(f.name))
			hash, err := h.HashNode(f.field)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}

	case ast.ShapeFieldNode:
		h.writeHashes(hasher, NodeKind["ShapeField"])
		if n.Assertion != nil {
			hash, err := h.HashNode(*n.Assertion)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
		if n.Shape != nil {
			hash, err := h.HashNode(*n.Shape)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}

	case ast.AssertionNode:
		h.writeHashes(hasher, NodeKind["Assertion"])
		if n.BaseType != nil {
			h.writeHashes(hasher, []byte(*n.BaseType))
		}
		// Sort constraints for deterministic ordering
		constraints := make([]ast.ConstraintNode, len(n.Constraints))
		copy(constraints, n.Constraints)
		sort.Slice(constraints, func(i, j int) bool {
			return constraints[i].Name < constraints[j].Name
		})
		// Hash each constraint in sorted order
		for _, constraint := range constraints {
			hash, err := h.HashNode(constraint)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}

	case ast.ConstraintNode:
		h.writeHashes(hasher, NodeKind["Constraint"])
		h.writeHashes(hasher, []byte(n.Name))
		nodes := make([]ast.Node, len(n.Args))
		for i, arg := range n.Args {
			nodes[i] = arg
		}
		hash, err := h.hashNodes(nodes)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)

	case ast.ConstraintArgumentNode:
		h.writeHashes(hasher, NodeKind["ConstraintArgument"])
		if n.Value != nil {
			hash, err := h.HashNode(*n.Value)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
		if n.Shape != nil {
			hash, err := h.HashNode(*n.Shape)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case ast.PackageNode:
		h.writeHashes(hasher, NodeKind["Package"])
		h.writeHashes(hasher, []byte(n.Ident.ID))
	case ast.ImportNode:
		h.writeHashes(hasher, NodeKind["Import"])
		h.writeHashes(hasher, []byte(n.Path))
		if n.Alias != nil {
			h.writeHashes(hasher, []byte(n.Alias.ID))
		}
	case ast.TypeDefNode:
		if n.Ident != "" {
			// For named types, hash only the identifier
			h.writeHashes(hasher, NodeKind["TypeDef"])
			h.writeHashes(hasher, []byte(n.Ident))
			break
		}
		h.writeHashes(hasher, NodeKind["TypeDef"])
		h.writeHashes(hasher, []byte(n.Expr.String()))
		h.writeHashes(hasher, []byte(n.Ident))
	case ast.ReturnNode:
		h.writeHashes(hasher, NodeKind["Return"])
		// Hash all return values
		for _, value := range n.Values {
			hash, err := h.HashNode(value)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case ast.TypeNode:
		if n.Ident != "" {
			// For named types, hash only the identifier
			h.writeHashes(hasher, NodeKind["Type"])
			h.writeHashes(hasher, []byte(n.Ident))
			break
		}
		h.writeHashes(hasher, NodeKind["Type"])
		h.writeHashes(hasher, []byte(n.Ident))
	case ast.SimpleParamNode:
		h.writeHashes(hasher, NodeKind["SimpleParam"])
		h.writeHashes(hasher, []byte(n.Ident.ID))
		hash, err := h.HashNode(n.Type)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
	case ast.DestructuredParamNode:
		h.writeHashes(hasher, NodeKind["DestructuredParam"])
		// Sort fields for deterministic ordering
		fields := make([]string, len(n.Fields))
		copy(fields, n.Fields)
		sort.Strings(fields)
		for _, field := range fields {
			h.writeHashes(hasher, []byte(field))
		}
		hash, err := h.HashNode(n.Type)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
	case *ast.AssertionNode:
		h.writeHashes(hasher, NodeKind["Assertion"])
		if n.BaseType != nil {
			h.writeHashes(hasher, []byte(*n.BaseType))
		}
		// Sort constraints for deterministic ordering
		constraints := make([]ast.ConstraintNode, len(n.Constraints))
		copy(constraints, n.Constraints)
		sort.Slice(constraints, func(i, j int) bool {
			return constraints[i].Name < constraints[j].Name
		})
		// Hash each constraint in sorted order
		for _, constraint := range constraints {
			hash, err := h.HashNode(constraint)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case *ast.ShapeNode:
		h.writeHashes(hasher, NodeKind["Shape"])
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
			h.writeHashes(hasher, []byte(f.name))
			hash, err := h.HashNode(f.field)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case *ast.ShapeFieldNode:
		h.writeHashes(hasher, NodeKind["ShapeField"])
		if n.Assertion != nil {
			hash, err := h.HashNode(*n.Assertion)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
		if n.Shape != nil {
			hash, err := h.HashNode(*n.Shape)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case ast.ImportGroupNode:
		h.writeHashes(hasher, NodeKind["ImportGroup"])
		// Sort imports for deterministic ordering
		imports := make([]ast.ImportNode, len(n.Imports))
		copy(imports, n.Imports)
		sort.Slice(imports, func(i, j int) bool {
			return imports[i].Path < imports[j].Path
		})
		for _, importNode := range imports {
			hash, err := h.HashNode(importNode)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case ast.AssignmentNode:
		h.writeHashes(hasher, NodeKind["Assignment"])
		for _, lValue := range n.LValues {
			hash, err := h.HashNode(lValue)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
		for _, rValue := range n.RValues {
			hash, err := h.HashNode(rValue)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case *ast.EnsureBlockNode:
		h.writeHashes(hasher, NodeKind["EnsureBlock"])
		for _, node := range n.Body {
			hash, err := h.HashNode(node)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case ast.TypeGuardNode:
		h.writeHashes(hasher, NodeKind["TypeGuard"])
		h.writeHashes(hasher, []byte(n.Ident))
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
			hash, err := h.HashNode(param)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
		hash, err := h.hashNodes(n.Body)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
	case *ast.TypeGuardNode:
		return h.HashNode(*n)
	case ast.IfNode:
		h.writeHashes(hasher, NodeKind["If"])
		if n.Init != nil {
			hash, err := h.HashNode(n.Init)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
		hash, err := h.HashNode(n.Condition)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
		hash, err = h.hashNodes(n.Body)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
		for _, elseIf := range n.ElseIfs {
			hash, err := h.HashNode(elseIf.Condition)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
			hash, err = h.hashNodes(elseIf.Body)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
		if n.Else != nil {
			hash, err = h.hashNodes(n.Else.Body)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case *ast.IfNode:
		return h.HashNode(*n)
	case ast.ReferenceNode:
		h.writeHashes(hasher, NodeKind["Reference"])
		hash, err := h.HashNode(n.Value)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
	case ast.MapLiteralNode:
		h.writeHashes(hasher, NodeKind["MapLiteral"])
		// Hash the type
		hash, err := h.HashNode(n.Type)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
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
			hash, err := h.HashNode(entry.key)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
			hash, err = h.HashNode(entry.value)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
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
	case ast.ArrayLiteralNode:
		h.writeHashes(hasher, NodeKind["ArrayLiteral"])
		hash, err := h.HashNode(n.Type)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
		for _, value := range n.Value {
			hash, err := h.HashNode(value)
			if err != nil {
				return 0, err
			}
			h.writeHashes(hasher, hash)
		}
	case *ast.ArrayLiteralNode:
		return h.HashNode(*n)
	case ast.DereferenceNode:
		h.writeHashes(hasher, NodeKind["Dereference"])
		hash, err := h.HashNode(n.Value)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
	case ast.TypeDefShapeExpr:
		h.writeHashes(hasher, NodeKind["TypeDefShapeExpr"])
		hash, err := h.HashNode(n.Shape)
		if err != nil {
			return 0, err
		}
		h.writeHashes(hasher, hash)
	case ast.NilLiteralNode:
		if err := h.writeHashes(hasher, NodeKind["NilLiteral"]); err != nil {
			return 0, err
		}
		// All NilLiteralNode instances are structurally identical
		return NodeHash(hasher.Sum64()), nil
	default:
		return 0, fmt.Errorf("unsupported node type: %T", n)
	}

	return NodeHash(hasher.Sum64()), nil
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
// with T_ prefix
func (h NodeHash) ToTypeIdent() ast.TypeIdent {
	if uint64(h) == NilHash {
		return ast.TypeIdent("T_Invalid")
	}
	return ast.TypeIdent("T_" + h.toBase58())
}

// ToGuardIdent generates a string name for a guard function based on its hash value
// with G_ prefix
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
