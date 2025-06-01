package ast

import (
	"fmt"
)

// TypeIdent is a unique identifier for a type
type TypeIdent string

// TypeNode represents a type in the Forst language
type TypeNode struct {
	Node
	Ident      TypeIdent
	Assertion  *AssertionNode
	TypeParams []TypeNode // Generic type parameters
}

const (
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
)

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
		return "Object"
	case TypeArray:
		return fmt.Sprintf("Array(%s)", t.TypeParams[0].String())
	case TypeMap:
		return fmt.Sprintf("Map(%s, %s)", t.TypeParams[0].String(), t.TypeParams[1].String())
	case TypeAssertion:
		return fmt.Sprintf("Assertion(%s)", t.Assertion.String())
	case TypeImplicit:
		return "(implicit)"
	default:
		if t.Assertion != nil {
			return fmt.Sprintf("%s(%s)", t.Ident, t.Assertion.String())
		}
		return string(t.Ident)
	}
}
