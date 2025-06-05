package ast

import (
	"fmt"
)

// TypeGuardNode represents a type guard declaration
type TypeGuardNode struct {
	Node
	// Name of the type guard
	Ident Identifier
	// Subject / Receiver parameter - the value being validated
	Subject ParamNode
	// Additional parameters used in validation
	Params []ParamNode
	// Body of the type guard
	Body []Node
}

// Kind returns the node kind for a type guard
func (t TypeGuardNode) Kind() NodeKind {
	return NodeKindTypeGuard
}

// GetIdent returns the identifier for the type guard
func (t TypeGuardNode) GetIdent() string {
	return string(t.Ident)
}

// String returns a string representation of the type guard
func (t TypeGuardNode) String() string {
	return fmt.Sprintf("TypeGuardNode(%s)", t.Ident)
}

// Parameters returns all parameters in order (subject first, then additional)
func (t TypeGuardNode) Parameters() []ParamNode {
	params := make([]ParamNode, 0, 1+len(t.Params))
	params = append(params, t.Subject)
	params = append(params, t.Params...)
	return params
}

// ShapeGuardNode represents a shape guard declaration
type ShapeGuardNode struct {
	TypeGuardNode
	// The type argument that will be inserted as a field
	TypeArg TypeNode
	// The field name to insert the type as
	FieldName Identifier
}

// Kind returns the node kind for a shape guard
func (s ShapeGuardNode) Kind() NodeKind {
	return NodeKindShapeGuard
}

// String returns a string representation of the shape guard
func (s ShapeGuardNode) String() string {
	return fmt.Sprintf("ShapeGuardNode(%s, %s: %s)", s.Ident, s.FieldName, s.TypeArg)
}

// ValidateShapeGuard validates that a type guard is a valid shape guard
func ValidateShapeGuard(node TypeGuardNode) error {
	// Check if the receiver type is a Shape type
	if !isShapeType(node.Subject.GetType()) {
		return fmt.Errorf("shape guard can only be used on Shape types, got %s", node.Subject.GetType())
	}

	// Check if the body contains exactly one return statement
	if len(node.Body) != 1 {
		return fmt.Errorf("shape guard must contain exactly one return statement")
	}

	// Check if the return statement contains a shape refinement
	returnStmt, ok := node.Body[0].(ReturnNode)
	if !ok {
		return fmt.Errorf("shape guard must return a shape refinement")
	}

	// Check if the return value is a shape refinement
	if !isShapeRefinement(returnStmt.Value) {
		return fmt.Errorf("shape guard must return a shape refinement")
	}

	return nil
}

// isShapeType checks if a type is a Shape type or a subtype thereof
func isShapeType(t TypeNode) bool {
	// TODO: Implement proper shape type checking
	return t.Ident == "Shape"
}

// isShapeRefinement checks if a node is a shape refinement
func isShapeRefinement(n Node) bool {
	// A shape refinement is a binary expression with:
	// 1. Left operand is a variable reference
	// 2. Operator is "is"
	// 3. Right operand is a Shape type with field refinements
	binExpr, ok := n.(BinaryExpressionNode)
	if !ok {
		return false
	}

	if binExpr.Operator != TokenIs {
		return false
	}

	// Check left operand is a variable reference
	_, ok = binExpr.Left.(VariableNode)
	if !ok {
		return false
	}

	// Check right operand is a Shape type
	shapeType, ok := binExpr.Right.(ShapeNode)
	if !ok {
		return false
	}

	// Check that the shape has at least one field refinement
	return len(shapeType.Fields) > 0
}

// Example usage:
// s is Shape({ field: String })  // Valid shape refinement
// s is Shape({})                 // Invalid (no refinements)
// s is Int()                     // Invalid (not a shape)
// s & t is Shape({ field: String })  // Valid (using & for composition)
