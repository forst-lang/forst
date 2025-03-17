package ast

// TypeNode represents a type in the Forst language
type TypeNode struct {
	Node
	Name string
}

const (
	// Built-in types
	TypeInt    = "TYPE_INT"
	TypeFloat  = "TYPE_FLOAT"
	TypeString = "TYPE_STRING"
	TypeBool   = "TYPE_BOOL"
	TypeVoid   = "TYPE_VOID"
	TypeError  = "TYPE_ERROR"

	// Placeholder for a type assertion
	TypeAssertion = "TYPE_ASSERTION"

	// Placeholder for an implicit type
	TypeImplicit = "TYPE_IMPLICIT"
)

// IsExplicit returns true if the type has been specified explicitly
func (t TypeNode) IsExplicit() bool {
	return t.Name != TypeImplicit
}

// IsImplicit returns true if the type has not been specified explicitly
func (t TypeNode) IsImplicit() bool {
	return t.Name == TypeImplicit
}

// NodeType returns the type of this AST node
func (t TypeNode) NodeType() NodeType {
	return NodeTypeType
}

func (t TypeNode) IsError() bool {
	return t.Name == TypeError
}

func (t TypeNode) String() string {
	switch t.Name {
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
	case TypeAssertion:
		return "Assertion"
	case TypeImplicit:
		return "(implicit)"
	default:
		return t.Name
	}
}
