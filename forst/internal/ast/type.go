package ast

import (
	"fmt"
	"strings"
)

// TypeIdent is a unique identifier for a type
type TypeIdent string

// TypeKind represents the origin/kind of a type
// Used for reliable type emission and reasoning
//
//	Builtin: Go/Forst built-in types (string, int, etc.)
//	UserDefined: Named types defined by the user (AppContext, etc.)
//	HashBased: Structural/anonymous types (T_xxx...)
type TypeKind int

const (
	TypeKindBuiltin TypeKind = iota
	TypeKindUserDefined
	TypeKindHashBased

	// TypeInt is the built-in int type
	TypeInt TypeIdent = "TYPE_INT"
	// TypeFloat is the built-in float type
	TypeFloat TypeIdent = "TYPE_FLOAT"
	// TypeString is the built-in string type
	TypeString TypeIdent = "TYPE_STRING"
	// TypeBool is the built-in bool type
	TypeBool TypeIdent = "TYPE_BOOL"
	// TypeVoid is the built-in void type
	TypeVoid TypeIdent = "TYPE_VOID"
	// TypeError is the built-in error type
	TypeError TypeIdent = "TYPE_ERROR"
	// TypeObject is the built-in object type
	TypeObject TypeIdent = "TYPE_OBJECT"
	// TypeArray is the built-in array type
	TypeArray TypeIdent = "TYPE_ARRAY"
	// TypeMap is the built-in map type
	TypeMap TypeIdent = "TYPE_MAP"

	// TypeAssertion is a placeholder for a type assertion
	TypeAssertion TypeIdent = "TYPE_ASSERTION"

	// TypeImplicit is a placeholder for an implicit type
	TypeImplicit TypeIdent = "TYPE_IMPLICIT"

	// TypeShape is a new type added
	TypeShape TypeIdent = "TYPE_SHAPE"

	// TypePointer is a new type added
	TypePointer TypeIdent = "TYPE_POINTER"
)

// TypeNode represents a type in the Forst language
// Kind must be set at construction time and never guessed from Ident
type TypeNode struct {
	Node
	Ident      TypeIdent
	Assertion  *AssertionNode
	TypeParams []TypeNode // Generic type parameters
	TypeKind   TypeKind   // Use TypeKind instead of Kind to avoid conflict
}

// IsExplicit returns true if the type has been specified explicitly
func (t TypeNode) IsExplicit() bool {
	return t.Ident != TypeImplicit
}

// IsImplicit returns true if the type has not been specified explicitly
func (t TypeNode) IsImplicit() bool {
	return t.Ident == TypeImplicit
}

// Kind returns the node kind for a type
func (t TypeNode) Kind() NodeKind {
	return NodeKindType
}

// IsError returns true if the type is an error type
func (t TypeNode) IsError() bool {
	return t.Ident == TypeError
}

func (t TypeNode) String() string {
	switch t.Ident {
	case TypeInt:
		return t.Ident.String()
	case TypeFloat:
		return t.Ident.String()
	case TypeString:
		return t.Ident.String()
	case TypeBool:
		return t.Ident.String()
	case TypeVoid:
		return t.Ident.String()
	case TypeError:
		return t.Ident.String()
	case TypeObject:
		return t.Ident.String()
	case TypeArray:
		if len(t.TypeParams) > 0 {
			return fmt.Sprintf("Array(%s)", t.TypeParams[0].String())
		}
		return "Array(?)"
	case TypeMap:
		if len(t.TypeParams) >= 2 {
			return fmt.Sprintf("Map(%s, %s)", t.TypeParams[0].String(), t.TypeParams[1].String())
		}
		return "Map(?, ?)"
	case TypeAssertion:
		if t.Assertion != nil {
			return fmt.Sprintf("Assertion(%s)", t.Assertion.String())
		}
		return "Assertion(?)"
	case TypeImplicit:
		return "(implicit)"
	case TypeShape:
		if len(t.TypeParams) > 0 {
			return fmt.Sprintf("Shape(%s)", t.TypeParams[0].String())
		}
		return "Shape"
	case TypePointer:
		if len(t.TypeParams) > 0 {
			return fmt.Sprintf("Pointer(%s)", t.TypeParams[0].String())
		}
		return "Pointer"
	default:
		if t.Assertion != nil {
			return fmt.Sprintf("%s(%s)", t.Ident, t.Assertion.String())
		}
		if len(t.TypeParams) > 0 {
			params := make([]string, len(t.TypeParams))
			for i, param := range t.TypeParams {
				params[i] = param.String()
			}
			return fmt.Sprintf("%s<%s>", t.Ident, strings.Join(params, ", "))
		}
		return string(t.Ident)
	}
}

// String returns a string representation of the type ident
func (ti TypeIdent) String() string {
	switch ti {
	case TypeInt:
		return "Int"
	case TypeFloat:
		return "Float"
	case TypeString:
		return "String"
	case TypeBool:
		return "Bool"
	case TypeVoid:
		return "Void"
	case TypeError:
		return "Error"
	case TypeObject:
		return "Object(?)"
	case TypeArray:
		return "Array(?)"
	case TypeMap:
		return "Map(?, ?)"
	case TypeAssertion:
		return "Assertion(?)"
	case TypeImplicit:
		return "(implicit)"
	case TypeShape:
		return "Shape(?)"
	case TypePointer:
		return "Pointer(?)"
	default:
		return string(ti)
	}
}

// IsGoBuiltinType returns true if the type node is a Go builtin type
func IsGoBuiltinType(node TypeNode) bool {
	return node.TypeKind == TypeKindBuiltin
}

// IsHashBasedType returns true if the type node is a hash-based/structural type
func IsHashBasedType(node TypeNode) bool {
	return node.TypeKind == TypeKindHashBased
}

// IsUserDefinedType returns true if the type node is a user-defined named type
func IsUserDefinedType(node TypeNode) bool {
	return node.TypeKind == TypeKindUserDefined
}
