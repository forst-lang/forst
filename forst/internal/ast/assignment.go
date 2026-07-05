package ast

import "strings"

// AssignmentNode represents a variable assignment in the AST
type AssignmentNode struct {
	Node
	LValues       []ExpressionNode // Assignable expressions (VariableNode or IndexExpressionNode, etc.)
	RValues       []ExpressionNode // Values being assigned
	ExplicitTypes []*TypeNode      // Optional explicit types for each name
	IsShort       bool             // Whether this is a short := assignment
	CompoundOp    TokenIdent       // Non-empty for +=, -=, etc.
	IsPackageLevel bool            // Top-level var in a .ft file
}

// Kind returns the node kind for an assignment
func (n AssignmentNode) Kind() NodeKind {
	return NodeKindAssignment
}

// String returns a string representation of the assignment
func (n AssignmentNode) String() string {
	var result strings.Builder
	result.WriteString("Assignment(")

	// Build comma-separated list of names and types
	for i, lv := range n.LValues {
		if i > 0 {
			result.WriteString(", ")
		}
		if vn, ok := lv.(VariableNode); ok {
			result.WriteString(vn.String())
			if len(n.ExplicitTypes) > i && n.ExplicitTypes[i] != nil {
				result.WriteString(": " + n.ExplicitTypes[i].String())
			}
		} else {
			result.WriteString(lv.String())
		}
	}

	// Add appropriate assignment operator
	switch {
	case n.IsShort:
		result.WriteString(" := ")
	case n.CompoundOp != "":
		result.WriteString(" " + CompoundAssignOperatorString(n.CompoundOp) + " ")
	default:
		result.WriteString(" = ")
	}

	// Add comma-separated list of values
	for i, val := range n.RValues {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(val.String())
	}

	return result.String() + ")"
}
